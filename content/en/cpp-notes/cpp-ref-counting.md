---
title: "Reference Counting"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["Memory Management"]
categories: ["CPP"]
draft: false
logs: "power by AI"
---

## Reference Counting {#reference-counting}

Reference counting is a deterministic memory management technique where each heap object carries an integer counter tracking how many active references point to it. When a new reference is created, the counter increments; when a reference is destroyed, the counter decrements. The moment the counter reaches zero, the object is immediately and deterministically destroyed.

This stands in contrast to tracing garbage collection (Java, C#, Go), where objects accumulate until a GC cycle scans the heap to identify unreachable objects. Reference counting gives you predictable, immediate cleanup — no GC pauses, no finalizers running at unpredictable times — at the cost of per-operation overhead and the fundamental problem of cycles.


### How It Works {#how-it-works}

The algorithm is simple but its correctness depends on careful implementation of increment and decrement at every reference boundary:

1.  Object is allocated with counter = 1 (the initial reference owns it).
2.  Each copy of a smart pointer/smart reference increments the counter (atomically for thread safety).
3.  Each destruction or reassignment of a smart pointer decrements the counter.
4.  When the counter transitions from 1 to 0, the object is deleted.

The critical invariant: the counter must exactly equal the number of live references at all times. A missing increment → premature deletion (use-after-free). A missing decrement → memory leak. Both are catastrophic in different ways.


### Where It Is Used {#where-it-is-used}

Reference counting is pervasive across systems programming and game engines:

-   `std::shared_ptr<T>` in the C++ standard library — non-intrusive, with a separate control block.
-   `Ref<T>` in the Godot engine — intrusive, counter embedded inside `RefCounted` objects.
-   **Python's CPython implementation** — every PyObject has an `ob_refcnt` field; objects are freed when it hits zero (plus a cycle-detecting GC for container objects).
-   **Objective-C's manual retain/release** (pre-ARC) and Swift's ARC (Automatic Reference Counting) — the compiler inserts retain/release calls at compile time.
-   **COM (Component Object Model)** on Windows — `IUnknown::AddRef()` and `IUnknown::Release()`.
-   **Linux kernel** — `kref` (kernel reference counting) for shared kernel objects like `struct file` and `struct net_device`.


### Atomic Reference Counting and Thread Safety {#atomic-reference-counting-and-thread-safety}

When multiple threads share a `shared_ptr`, increments and decrements on the reference counter must be **atomic** to avoid data races. `std::shared_ptr` guarantees this: the control block's counters use `std::atomic<int>` (or equivalent) with `memory_order_relaxed` for increments and `memory_order_acq_rel` for decrements.

```cpp
// Simplified view of shared_ptr's internal ref-counting
void increment_ref() {
    strong_refs.fetch_add(1, std::memory_order_relaxed);
}

void decrement_ref() {
    if (strong_refs.fetch_sub(1, std::memory_order_acq_rel) == 1) {
        // I was the last strong reference — delete the object
        delete_object();
        // If no weak refs remain, also delete the control block
        if (weak_refs.load(std::memory_order_acquire) == 0)
            delete_control_block();
    }
}
```

The memory ordering choices are carefully tuned:

-   `relaxed` for increments: we don't need synchronization with other memory operations at increment time.
-   `acq_rel` for the decrement that reaches zero: we need an acquire barrier to see all prior writes to the object before we destroy it, and a release barrier so other threads see the count hit zero.

This is why `shared_ptr` has a measurable cost compared to `unique_ptr` — atomic operations are 10-100x more expensive than non-atomic integer ops, and they also inhibit certain compiler optimizations.

Godot's `Ref<T>` is a noted exception: it uses non-atomic reference counting for performance. This makes `Ref<T>` **not thread-safe** — sharing a `Ref<T>` across threads requires external synchronization.


### Control Block Layout in shared_ptr {#control-block-layout-in-shared-ptr}

The control block is the hidden heap allocation that enables non-intrusive reference counting for `shared_ptr`. Its layout (simplified):

```cpp
struct control_block {
    std::atomic<long> strong_count;   // number of shared_ptr instances
    std::atomic<long> weak_count;     // number of weak_ptr instances + (strong_count > 0 ? 1 : 0)
    deleter_fn         deleter;       // type-erased deleter (function pointer or similar)
    allocator_fn       allocator;     // optional — for allocator-aware construction
};
```

The weak count is always ≥ the number of `weak_ptr` instances. When the strong count hits zero, the **object** is destroyed, but the **control block** persists until the weak count also reaches zero. This design ensures that `weak_ptr::lock()` can always atomically check the strong count in the still-alive control block.

When you use `std::make_shared<T>(args...)`, the object and control block are allocated together in a single contiguous block — one heap allocation instead of two. This improves cache locality and reduces allocation overhead, but it means the object's memory cannot be freed until all ~weak_ptr~s are also gone (the control block and object share the same allocation).


### Intrusive vs Non-Intrusive Reference Counting {#intrusive-vs-non-intrusive-reference-counting}

| Property               | Non-Intrusive (`shared_ptr`)               | Intrusive (`Ref<T>`, COM, Linux kref)    |
|------------------------|--------------------------------------------|------------------------------------------|
| Counter location       | Separate control block (heap)              | Embedded in the object itself            |
| Can retroactively use? | Yes — works with any type                  | No — object must inherit from base       |
| Memory overhead        | 2 heap allocations (or 1 with make_shared) | 1 allocation + sizeof(counter) in object |
| Size of smart ptr      | 2 pointers (16 bytes on 64-bit)            | 1 pointer (8 bytes on 64-bit)            |
| Thread safety          | Atomic by default (std)                    | Depends on implementation                |
| External APIs          | Works with any `T`                         | Requires `RefCounted` base class         |

Intrusive counting (Godot's `RefCounted`) is lighter weight — no separate control block allocation, no double indirection — but it requires buy-in from the type system. You cannot retroactively make a `std::string` ref-counted with an intrusive scheme. Non-intrusive counting (`shared_ptr`) works with any type, including fundamental types like `int`, but carries the double-pointer and separate-allocation overhead.


### Deferred Reclamation and Hazard Pointers {#deferred-reclamation-and-hazard-pointers}

Reference counting has a fundamental scalability problem: every copy and destruction of a `shared_ptr` touches globally shared atomic variables, causing cache-line contention across cores. For high-throughput systems (databases, network stacks), this contention can dominate performance.

Advanced techniques address this:

-   **Deferred reference counting** — bulk-update counters periodically rather than on every operation. Used in some Python implementations and research systems.
-   **Hazard pointers** — threads publish which objects they are currently accessing ("hazards"); a reclamation thread only frees objects not in any thread's hazard list. Used by the **folly** library and proposed for the C++26 standard.
-   **RCU (Read-Copy-Update)** — readers access data without any atomic operations; writers create new versions and defer reclamation until all pre-existing readers have finished. Used in the Linux kernel.
-   **`std::atomic_shared_ptr<T>` (C++20)** — allows atomic load/store/exchange on shared_ptr, enabling lock-free data structures that share ownership.


### Circular Reference Problem {#circular-reference-problem}

The fundamental limitation of reference counting: if A holds a `shared_ptr` to B and B holds a `shared_ptr` back to A, neither counter can ever reach zero, even if no external code references either object. This is a **memory leak**, not a crash — the objects simply live forever.

```cpp
struct Node {
    std::shared_ptr<Node> parent;
    std::shared_ptr<Node> child;
};

auto a = std::make_shared<Node>();
auto b = std::make_shared<Node>();
a->child = b;
b->parent = a;  // cycle: a ↔ b
// When a and b go out of scope, counters drop to 1, never 0 — leak!
```

The fix: replace one direction of the cycle with `std::weak_ptr` (or, in Godot, `WeakRef`):

```cpp
struct Node {
    std::weak_ptr<Node>   parent;   // non-owning — breaks the cycle
    std::shared_ptr<Node> child;    // owning
};
```

The rule of thumb: the "natural owner" direction gets `shared_ptr`; the "back-reference" direction gets `weak_ptr`. In tree structures, parent→child is owning (shared_ptr), child→parent is non-owning (weak_ptr).


### Related Notes {#related-notes}

-   [C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}}) — shared_ptr and Ref&lt;T&gt; use reference counting
-   [Copy Constructors]({{< relref "cpp-copy-constructors.md" >}}) — copy constructors trigger reference count increments
-   [C++ Memory Management]({{< relref "cpp-memory-management.md" >}}) — parent topic; reference counting vs other GC strategies
-   [C++ Templates]({{< relref "cpp-templates.md" >}}) — shared_ptr's control block is a template implementation detail
