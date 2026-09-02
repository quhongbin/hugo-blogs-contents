---
title: "C++ Smart Pointers & Ref"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["CPP"]
draft: false
logs: "test"
logs_descripe: "中文测试"
---

## C++ Smart Pointers &amp; Ref {#c-plus-plus-smart-pointers-and-ref}

Smart pointers are RAII wrappers around raw pointers that automatically manage the lifetime of dynamically allocated objects. They embody the principle that **ownership should be expressed in the type system** — when you see a `unique_ptr<T>`, you know exactly who owns the object and when it will be destroyed. No manual `delete`, no dangling pointers, no double-free.

They are not just convenience — they are the foundation of exception-safe C++. If an exception is thrown between a `new` and its matching `delete` in raw-pointer code, the object leaks. Smart pointers guarantee cleanup regardless of how a scope is exited (normal return, exception, early return).


### Types of Smart Pointers {#types-of-smart-pointers}

C++11 introduced three core smart pointer types in `<memory>`:

| Type            | Ownership            | Cloneable      | Overhead                      | Use case                        |
|-----------------|----------------------|----------------|-------------------------------|---------------------------------|
| `unique_ptr<T>` | Exclusive            | No (move-only) | Zero (raw ptr size)           | Single owner; factory functions |
| `shared_ptr<T>` | Shared (ref-counted) | Yes            | 2 pointers (ptr + ctrl block) | Shared ownership graphs         |
| `weak_ptr<T>`   | Non-owning observer  | Yes            | Same as shared_ptr            | Breaking cycles; caching        |

The choice between them is a design decision about ownership, not performance:

-   **If only one place owns the object** → `unique_ptr`. It is as cheap as a raw pointer (same size, no atomic ops) and enforces sole ownership at compile time.
-   **If multiple places genuinely co-own the object and the last one to go should clean up** → `shared_ptr`. Be aware of the hidden cost: atomic reference count operations on every copy/destroy.
-   **If you need to observe without extending lifetime** → `weak_ptr`. Always created from a `shared_ptr`; does not keep the object alive.


### Godot Ref&lt;T&gt; {#godot-ref-t}

Godot engine uses its own smart pointer template `Ref<T>` instead of `std::shared_ptr<T>`. It works exclusively with classes that inherit from `RefCounted`, which embeds the reference counter directly inside the object (intrusive reference counting — see [Reference Counting]({{< relref "cpp-ref-counting.md" >}})).

The key difference from `shared_ptr`:

-   `Ref<T>` stores a single pointer — the counter is inside the object, not in a separate control block.
-   There is no equivalent of `weak_ptr<T>` for `Ref<T>` (though Godot provides `WeakRef` as a separate mechanism).
-   `Ref<T>` is **not** thread-safe by default — reference count operations are not atomic.


### Custom Deleters with unique_ptr {#custom-deleters-with-unique-ptr}

`unique_ptr` accepts an optional **deleter** type as its second template parameter. By default it uses `std::default_delete<T>` which calls `delete`, but you can replace it to handle non-standard resource cleanup:

```cpp
// Deleter for C FILE* — calls fclose instead of delete
auto fileDeleter = [](FILE* f) { if (f) fclose(f); };
std::unique_ptr<FILE, decltype(fileDeleter)> file(fopen("data.txt", "r"), fileDeleter);

// Deleter for malloc'd memory
auto freeDeleter = [](void* p) { free(p); };
std::unique_ptr<void, decltype(freeDeleter)> buf(malloc(1024), freeDeleter);

// Stateless lambda → unique_ptr is same size as raw pointer
// Stateful lambda or function pointer → unique_ptr grows by sizeof(deleter)
```

This makes `unique_ptr` a universal RAII handle — not just for `new~/~delete`, but for any resource with an allocate/release pair: file handles, sockets, GPU buffers, database connections.


### shared_ptr Control Block Internals {#shared-ptr-control-block-internals}

A `shared_ptr<T>` stores two pointers internally:

1.  A pointer to the managed object (the `T*` you see with `.get()`)
2.  A pointer to a **control block** — a heap-allocated structure containing:
    -   **Strong reference count** — how many `shared_ptr` instances share ownership
    -   **Weak reference count** — how many `weak_ptr` instances observe the object
    -   **Deleter** — the function to call when the strong count hits zero
    -   **Allocator** — (optional) the allocator used for the control block

The control block is created once — either by `std::make_shared` (which allocates the object and control block together in a single heap allocation) or when you construct a `shared_ptr` from a raw pointer (which requires a separate allocation for the control block).

Crucially, the **object** is destroyed when the strong count reaches zero, but the **control block itself** survives as long as there are weak pointers, because `weak_ptr::lock()` needs to check whether the object still exists by reading the strong count from the control block.


### weak_ptr::lock() and Observer Pattern {#weak-ptr-lock-and-observer-pattern}

`weak_ptr<T>` does not keep the object alive. To access the object, you must call `.lock()`, which returns a `shared_ptr<T>`:

-   If the object still exists → `lock()` increments the strong count and returns a valid `shared_ptr`.
-   If the object has been destroyed → `lock()` returns an empty (null) `shared_ptr`.

<!--listend-->

```cpp
std::weak_ptr<Widget> observer = someSharedPtr;

if (auto sp = observer.lock()) {  // try to promote to shared_ptr
    sp->doSomething();            // safe — object is alive and pinned
} else {
    // object has been destroyed
}
```

This is the foundation of the **observer pattern without dangling pointers** — a cache or listener registry can store `weak_ptr` entries and safely detect when the target has been cleaned up. Compare to raw pointers, where a cache entry could silently become dangling.

`.expired()` checks whether the object is gone, but it has a TOCTOU (time-of-check-time-of-use) race in multi-threaded code — prefer `.lock()` directly.


### enable_shared_from_this {#enable-shared-from-this}

A common pitfall: inside a member function, you have `this` — a raw pointer. If you create a `shared_ptr` from `this`, you get a **second, independent control block**, leading to double-delete when both `shared_ptr` groups reach zero.

```cpp
// WRONG — double control block, double delete
class Widget {
public:
    std::shared_ptr<Widget> getShared() {
        return std::shared_ptr<Widget>(this);  // BUG!
    }
};
```

The fix is to inherit from `std::enable_shared_from_this<Widget>` and call `shared_from_this()`:

```cpp
class Widget : public std::enable_shared_from_this<Widget> {
public:
    std::shared_ptr<Widget> getShared() {
        return shared_from_this();  // correct — returns the existing control block
    }
};

// Must be managed by shared_ptr before calling shared_from_this()
auto w = std::make_shared<Widget>();
auto w2 = w->getShared();  // w2 shares the same control block as w
```

`enable_shared_from_this` works by storing a `weak_ptr` inside the base class. The first `shared_ptr` that takes ownership of the object initializes this `weak_ptr`. `shared_from_this()` calls `weak_ptr::lock()` to produce a `shared_ptr` that shares the existing control block. If called on an object not managed by `shared_ptr`, it throws `std::bad_weak_ptr`.


### make_unique / make_shared — Why Prefer Them {#make-unique-make-shared-why-prefer-them}

Always prefer `std::make_unique<T>(args...)` and `std::make_shared<T>(args...)` over raw `new`:

```cpp
// Preferred
auto p = std::make_unique<Widget>(42, "name");
auto s = std::make_shared<Widget>(42, "name");

// Avoid — more typing, exception-unsafe, potentially less efficient
std::unique_ptr<Widget> p2(new Widget(42, "name"));
std::shared_ptr<Widget> s2(new Widget(42, "name"));
```

Reasons:

1.  **Exception safety**: In a function call like `f(shared_ptr<X>(new X), g())`, if `g()` throws after `new X` but before the `shared_ptr` constructor completes, the `X` object leaks. `make_shared` guarantees the allocation and construction happen as one atomic step.

2.  **Fewer allocations**: `make_shared` allocates the object and its control block in a single heap allocation (one `malloc` instead of two). `make_unique` has no such advantage (unique_ptr has no control block), but it is still cleaner.

3.  **No repetition of the type**: `auto p = make_unique<Widget>()` vs `unique_ptr<Widget> p(new Widget)` — the type name appears only once.

The one case where you **must** use raw `new` with `shared_ptr` is when you need a custom deleter — `make_shared` cannot accept one.


### Example {#example}

```cpp
Ref<MyNode> node = memnew(MyNode);
node->do_something();
// node is automatically freed when 'node' goes out of scope
```

In standard C++ (no Godot):

```cpp
// unique_ptr — exclusive ownership
auto widget = std::make_unique<Widget>();
widget->draw();
auto widget2 = std::move(widget);  // ownership transferred; widget is now null

// shared_ptr — shared ownership
auto s1 = std::make_shared<Texture>();
auto s2 = s1;                      // reference count now 2
s1.reset();                        // count drops to 1
// s2 goes out of scope → count hits 0 → Texture is deleted
```


### Related Notes {#related-notes}

-   [Reference Counting]({{< relref "cpp-ref-counting.md" >}}) — the mechanism shared_ptr and Ref&lt;T&gt; use to track ownership
-   [C++ Templates]({{< relref "cpp-templates.md" >}}) — all smart pointers are class templates
-   [Copy Constructors]({{< relref "cpp-copy-constructors.md" >}}) — smart pointers implement copy and move semantics
-   [C++ Memory Management]({{< relref "cpp-memory-management.md" >}}) — parent topic; RAII and resource lifetime
-   [C++ Function Pointers]({{< relref "cpp-func-pointers.md" >}}) — custom deleters can be function pointers or lambdas
-   [C++ void\* Pointers]({{< relref "cpp-void-ptr.md" >}}) — raw void\* is the pre-modern way to handle type-erased pointers; smart pointers replace it
