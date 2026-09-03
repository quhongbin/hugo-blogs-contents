---
title: "Copy Constructors"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["Memory Management", "RAII"]
categories: ["CPP"]
draft: false
logs: "power by AI"
---

## Copy Constructors {#copy-constructors}

A copy constructor creates a new object by copying the state of an existing object of the same type. It is one of the **special member functions** that the compiler can generate automatically — but knowing when to rely on the compiler's version and when to write your own is a core C++ skill.

Copy constructors are everywhere: passing by value, returning by value, initializing with `=`, throwing exceptions. Getting them wrong leads to double-free, dangling pointers, or silent data corruption — some of the hardest bugs to diagnose.


### Syntax {#syntax}

A copy constructor takes a const lvalue reference to its own type:

```cpp
class MyClass {
public:
    MyClass(const MyClass& other);  // copy constructor — must take by reference!
    // Note: taking by value would require a copy constructor → infinite recursion
};
```

The parameter **must** be a reference. If it were by value, the compiler would need to copy the argument to pass it — calling the copy constructor — which would need to copy the argument — infinite recursion at compile time (caught as an error, not a runtime loop).


### When the Copy Constructor Is Called {#when-the-copy-constructor-is-called}

The copy constructor is invoked in these situations:

1.  Direct initialization from another object: `MyClass b(a);`
2.  Copy initialization: `MyClass b = a;`
3.  Passing an object by value to a function: `void f(MyClass x); f(a);`
4.  Returning an object by value (pre-C++17; modern C++ uses move or elision): `return obj;`
5.  Throwing an exception by value: `throw MyClass();` (copy into the exception object)
6.  Brace initialization from an lvalue: `MyClass b{a};`

Crucially, `MyClass b = a;` calls the copy constructor, **not** the copy assignment operator. This is a common point of confusion: initialization (creating a new object) uses constructors; assignment (modifying an existing object) uses `operator=`.


### Default Copy Constructor {#default-copy-constructor}

If you do not declare a copy constructor, the compiler generates one automatically. The default copy constructor performs a **memberwise copy** — it calls the copy constructor of each member (or performs a bitwise copy for fundamental types like `int`, `double`).

```cpp
struct Safe {
    std::string name;
    int         id;
    // Compiler-generated copy constructor:
    // Safe(const Safe& other) : name(other.name), id(other.id) {}
    // This is correct — std::string handles its own deep copy.
};

struct Dangerous {
    char* buffer;  // raw pointer — the default copy copies the pointer, not the data
    size_t size;
    // Compiler-generated copy constructor:
    // Dangerous(const Dangerous& other) : buffer(other.buffer), size(other.size) {}
    // BUG: two objects point to the same buffer → double-free on destruction
};
```

The default copy constructor is correct when all members are **value-semantic types** (`std::string`, `std::vector`, `int`, `double`, smart pointers). It is dangerous when the class manages raw resources (raw pointers, file handles, sockets) — the copy only duplicates the handle, not the resource, leading to double-close or double-free.


### Rule of Three / Five {#rule-of-three-five}

The **Rule of Three**: if a class defines any one of the following, it almost certainly needs all three:

1.  Destructor
2.  Copy constructor
3.  Copy assignment operator

The reasoning: if you need a custom destructor (to release a resource), the default memberwise copy will duplicate the handle, leading to double release. You must define copy semantics that either deep-copy the resource or prevent copying.

The **Rule of Five** extends this for modern C++ (C++11+):

1.  Move constructor
2.  Move assignment operator

If your class manages a resource, providing move semantics allows the resource to be transferred efficiently instead of deep-copied.

```cpp
class Buffer {
    char*  data;
    size_t size;

public:
    // Constructor
    Buffer(size_t sz) : data(new char[sz]), size(sz) {}

    // Destructor (Rule of Three trigger)
    ~Buffer() { delete[] data; }

    // Copy constructor (deep copy)
    Buffer(const Buffer& other) : data(new char[other.size]), size(other.size) {
        std::copy(other.data, other.data + size, data);
    }

    // Copy assignment (deep copy with self-assignment check)
    Buffer& operator=(const Buffer& other) {
        if (this != &other) {
            delete[] data;
            data = new char[other.size];
            size = other.size;
            std::copy(other.data, other.data + size, data);
        }
        return *this;
    }

    // Move constructor (Rule of Five — transfer ownership)
    Buffer(Buffer&& other) noexcept : data(other.data), size(other.size) {
        other.data = nullptr;
        other.size = 0;
    }

    // Move assignment
    Buffer& operator=(Buffer&& other) noexcept {
        if (this != &other) {
            delete[] data;
            data = other.data;
            size = other.size;
            other.data = nullptr;
            other.size = 0;
        }
        return *this;
    }
};
```


### Copy Elision — RVO and NRVO {#copy-elision-rvo-and-nrvo}

The compiler is allowed — and since C++17, **required** in some cases — to eliminate copy and move operations by constructing the returned object directly in the caller's memory. This is **copy elision**.

Two forms:

-   **RVO** (Return Value Optimization): a temporary object is constructed directly in the caller's stack frame, skipping the copy/move entirely.
-   **NRVO** (Named RVO): the same optimization, but for a named local variable being returned.

<!--listend-->

```cpp
Widget makeWidget() {
    return Widget(42);  // RVO: Widget constructed directly in caller's frame
}

Widget makeWidget2() {
    Widget w(42);
    w.setup();
    return w;           // NRVO: 'w' is constructed directly in caller's frame
}

Widget w = makeWidget();  // No copy or move — guaranteed in C++17
```

Since C++17, RVO is **mandatory** for prvalues (temporaries). NRVO is permitted but not guaranteed — the compiler may choose to move instead if elision is not possible. The important implication: **do not rely on the copy/move constructor being called for debugging side effects** — the compiler may elide it.


### Move Constructor and Move Assignment {#move-constructor-and-move-assignment}

Move semantics (C++11) allow transferring ownership of a resource from one object to another without copying. A move constructor takes an **rvalue reference** (`~&`) and typically "steals" the guts of the source, leaving it in a valid-but-unspecified state (usually null/empty).

```cpp
std::vector<int> v1 = {1, 2, 3, 4, 5};
std::vector<int> v2 = std::move(v1);  // move — v1's internal buffer is transferred to v2
// v1 is now empty (valid, but size() == 0)
```

Move operations are especially important for types that manage heap resources, large buffers, or non-copyable handles (e.g., `std::unique_ptr`, which is move-only). Without move semantics, returning a large `std::vector` from a function would require copying every element — with move, only three pointers are swapped.

`std::move` does not actually move anything. It is a cast to an rvalue reference, enabling move semantics if the target type has a move constructor/assignment. The actual transfer happens in the move constructor or move assignment operator.


### Why noexcept Matters for Move {#why-noexcept-matters-for-move}

Marking move constructors and move assignment operators as `noexcept` is critical for performance and correctness with standard containers:

```cpp
class Widget {
public:
    Widget(Widget&& other) noexcept;  // mark as noexcept!
    Widget& operator=(Widget&& other) noexcept;
};
```

Why it matters: when `std::vector` reallocates, it must move existing elements to the new buffer. If the move constructor is `noexcept`, the vector uses move; if it is not (or not declared), the vector falls back to **copy** to preserve the strong exception guarantee (if a move throws partway through, the original buffer is corrupted and cannot be restored).

This has a massive performance impact: moving a `std::vector<std::string>` can be 100x faster than copying it, but only if the string's move constructor is `noexcept`.

The rule: move constructors and move assignment operators should always be `noexcept` unless there is a strong reason they cannot be. Most moves that simply transfer pointers/ints are naturally noexcept.


### Rule of Zero — Let the Compiler Do the Work {#rule-of-zero-let-the-compiler-do-the-work}

The **Rule of Zero** states: classes that do not manage resources directly should **not** declare any of the special member functions. Let the compiler generate them. This is the modern ideal:

```cpp
// Rule of Zero — all members are value-semantic or smart pointers
class Person {
    std::string              name;
    int                      age;
    std::vector<std::string> hobbies;
    std::unique_ptr<Address> address;  // unique_ptr is move-only — correct default behavior
    // No destructor, no copy/move constructors, no assignment — compiler handles all
};

// The compiler-generated destructor destroys each member.
// The compiler-generated copy constructor copies each member.
// The compiler-generated move constructor moves each member.
// All are correct because std::string, std::vector, and unique_ptr manage themselves.
```

The Rule of Zero works when every member is a **well-behaved value type** — standard library containers, smart pointers, or your own types that also follow the Rule of Zero. If any member is a raw resource handle (raw pointer, file descriptor, OS handle), you must drop to the Rule of Five and manage it explicitly.


### Copy-and-Swap Idiom {#copy-and-swap-idiom}

The copy-and-swap idiom provides a clean, exception-safe way to implement copy assignment by reusing the copy constructor and a `swap` function:

```cpp
class Buffer {
    char*  data;
    size_t size;

public:
    Buffer(const Buffer& other);  // copy constructor
    ~Buffer();

    friend void swap(Buffer& a, Buffer& b) noexcept {
        std::swap(a.data, b.data);
        std::swap(a.size, b.size);
    }

    // Copy assignment via copy-and-swap
    Buffer& operator=(Buffer other) {  // pass by value → copy is made here
        swap(*this, other);            // swap *this with the copy
        return *this;
    }  // 'other' (the old *this) is destroyed here
};
```

How it works:

1.  The parameter is passed **by value**, triggering the copy constructor.
2.  `swap` exchanges `*this`'s guts with the copy's guts.
3.  The copy goes out of scope and its destructor cleans up `*this`'s old data.

This provides the **strong exception guarantee** — if the copy throws, `*this` is untouched. It also handles self-assignment naturally (`a = a` makes a copy, swaps with itself, destroys the copy — net no change). The tradeoff is an extra allocation (the copy) compared to the manual check-then-copy approach.


### Implicit Generation Rules (=default, =delete) {#implicit-generation-rules--default-delete}

The compiler's rules for generating special member functions are nuanced. Declaring any of them can suppress generation of others:

| If you declare... | Compiler does NOT generate...                          |
|-------------------|--------------------------------------------------------|
| Destructor        | Move constructor, move assignment                      |
| Copy constructor  | Move constructor, move assignment, default constructor |
| Copy assignment   | Move constructor, move assignment                      |
| Move constructor  | Copy constructor, copy assignment, default constructor |
| Move assignment   | Copy constructor, copy assignment, default constructor |

`=default` explicitly requests the compiler-generated version (even when it would otherwise be suppressed). `=delete` explicitly forbids it:

```cpp
class NonCopyable {
public:
    NonCopyable() = default;
    NonCopyable(const NonCopyable&) = delete;            // no copying
    NonCopyable& operator=(const NonCopyable&) = delete; // no copy assignment
    NonCopyable(NonCopyable&&) = default;                // but moving is fine
    NonCopyable& operator=(NonCopyable&&) = default;
};
```

This is the idiomatic way to express move-only types. `std::unique_ptr`, `std::thread`, `std::mutex` are all move-only.


### Shallow vs Deep Copy {#shallow-vs-deep-copy}

| Copy Type   | What It Duplicates            | Risk                           | When to Use                |
|-------------|-------------------------------|--------------------------------|----------------------------|
| **Shallow** | Pointer/handle (address only) | Double-free, dangling pointer  | Shared immutable data, COW |
| **Deep**    | The pointed-to data itself    | Performance (extra allocation) | Exclusive ownership        |

The default copy constructor always performs a shallow (memberwise) copy. For classes managing heap resources, a deep copy is usually required — but modern C++ makes this rare by replacing raw pointers with value-semantic types (`std::string` instead of `char*`, `std::vector` instead of `new[]`).


### Best Practice {#best-practice}

Follow the Rule of Zero by default. Prefer `std::string`, `std::vector`, smart pointers ([C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}})), and other RAII types over raw pointers. These types handle copy and move semantics correctly — you inherit correct behavior without writing a single special member function.

When you must manage a raw resource (OS handles, custom allocators, C library wrappers), apply the Rule of Five:

1.  Write a destructor to release the resource.
2.  Write a copy constructor to deep-copy the resource (or `=delete` it if copying should be forbidden).
3.  Write a copy assignment (use copy-and-swap for exception safety).
4.  Write a move constructor (`noexcept`) to transfer the resource.
5.  Write a move assignment (`noexcept`).

If you find yourself writing Rule of Five for a single resource, consider extracting it into a small RAII wrapper class — then your main class can return to the Rule of Zero.


### Related Notes {#related-notes}

-   [C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}}) — smart pointers implement correct copy/move semantics (unique_ptr is move-only)
-   [Reference Counting]({{< relref "cpp-ref-counting.md" >}}) — shared_ptr's copy constructor increments the reference count
-   [C++ Memory Management]({{< relref "cpp-memory-management.md" >}}) — parent topic; RAII binds resource lifetime to object lifetime
-   [C++ Templates]({{< relref "cpp-templates.md" >}}) — template constructors are never copy constructors; template classes follow the same rules
-   [C++ Enum vs Struct Class]({{< relref "cpp-enum-struct.md" >}}) — structs with raw pointers need custom copy constructors (Rule of Three)
-   [C++ Function Pointers]({{< relref "cpp-func-pointers.md" >}}) — lambdas with copy captures invoke copy constructors of captured objects
