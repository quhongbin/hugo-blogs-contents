---
title: "C++ void* Pointers"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["pointer"]
categories: ["CPP"]
draft: false
logs: "power by AI"
---

## C++ void\* Pointers {#c-plus-plus-void-pointers}

`void*` is a pointer type in C++ that represents "pointer to anything" — it discards the type information of the data it points to. It is sometimes called a generic pointer, untyped pointer, or opaque pointer. Because the compiler does not know what type of data lives at the address, `void*` carries zero type-safety guarantees and imposes strict constraints on what you can do with it directly.

`void*` sits at the boundary between C++'s strong type system and the low-level reality of raw memory. It is the lingua franca of C APIs, custom allocators, callback contexts, and any situation where you need to pass an address without committing to a type at the API boundary. Understanding `void*` is essential for reading real-world C and C++ codebases, including Godot, Linux kernel code, and the memory-management layers of game engines.


### Key Properties {#key-properties}

The core properties of `void*` stem from the compiler's lack of type knowledge:

-   **No type information** — A `void*` can hold the address of any data type (`int`, `double`, `struct MyClass`, etc.), but the compiler forgets what type it was. This is the essence of type erasure at the pointer level.

-   **No dereference without a cast** — You cannot write `*ptr` on a `void*` because the compiler does not know how many bytes to read or how to interpret them. You must first cast the `void*` back to a typed pointer: `*(int*)ptr`.

-   **No pointer arithmetic** — Expressions like `ptr + 1` or `ptr++` are illegal on `void*`. The compiler does not know `sizeof(*ptr)`, so it cannot compute the byte offset for the next element. (GCC has an extension that treats `void*` arithmetic as byte-level, but this is non-standard and non-portable.)

-   **Implicit conversion from any data pointer** — Any typed pointer (`int*`, `double*`, `MyClass*`) implicitly converts to `void*` without a cast. The reverse — `void*` to typed pointer — requires an explicit cast in C++ (in C it is implicit, which is a key difference).

-   **Cannot point to function** — A `void*` stores a data pointer. Function pointers are a distinct category in the C++ type system and are not guaranteed to be representable in a `void*` (POSIX systems typically allow it, but the C++ standard does not). Use a function-pointer typedef or `std::function` for storing callables.

-   **No ownership semantics** — `void*` is just a raw address. It does not know whether the memory should be freed, by whom, or when. You must track ownership separately.


### Example {#example}

The basic pattern: store in `void*`, cast back to use.

```cpp
int value = 42;
void* ptr = &value;       // implicit conversion: int* → void*

// Must cast before use — the programmer must know the original type
int* intPtr = static_cast<int*>(ptr);   // preferred in C++
// int* intPtr = (int*)ptr;             // C-style cast, also works
std::cout << *intPtr;  // prints 42
```

A more realistic example — a generic callback context in a C-style API:

```cpp
// API that stores user data as void*
void register_callback(void (*callback)(void*), void* user_data);

// User code
struct MyContext {
    int id;
    const char* name;
};

MyContext ctx{42, "player"};
register_callback([](void* data) {
    auto* ctx = static_cast<MyContext*>(data);
    printf("id=%d name=%s\n", ctx->id, ctx->name);
}, &ctx);
```


### Type Erasure via void\* {#type-erasure-via-void}

The pattern of hiding a concrete type behind a `void*` is called the **opaque pointer** or **Pimpl** (pointer-to-implementation) idiom. The public API only exposes `void*` (or a wrapper `struct` containing one), while the implementation file knows the real type.

```cpp
// handle.h — public header
struct Handle;  // forward declaration, incomplete type
Handle* create_handle(int param);
void    use_handle(Handle* h);
void    destroy_handle(Handle* h);

// handle.cpp — implementation (hidden from users)
struct Handle {
    int    id;
    double data;
    // ... complex internals ...
};

Handle* create_handle(int param) {
    auto* h = new Handle{param, param * 3.14};
    return h;   // caller treats this as opaque
}
```

This pattern has two major benefits: (1) **ABI stability** — changing Handle's internals does not require recompiling code that only sees the forward declaration; (2) **build-time isolation** — the public header stays clean without pulling in internal dependencies (libraries, platform headers, etc.).

However, `void*`-based type erasure has a critical weakness compared to templates: it moves type checking from compile time to runtime. If you cast to the wrong type, you get undefined behavior with no compiler warning. Templates ([C++ Templates]({{< relref "cpp-templates.md" >}})) provide compile-time type erasure with full safety — the tradeoff is potential code bloat from multiple instantiations.


### std::any — The Modern Alternative {#std-any-the-modern-alternative}

C++17 introduced `std::any`, which provides type-erased storage with **runtime type checking**. Unlike `void*`, `std::any` remembers what type it holds and throws `std::bad_any_cast` if you try to extract the wrong type.

```cpp
#include <any>

std::any a = 42;                     // holds int
a = 3.14;                            // now holds double
a = std::string("hello");            // now holds string

// Safe extraction with type check
try {
    auto s = std::any_cast<std::string>(a);   // OK
    auto i = std::any_cast<int>(a);           // throws bad_any_cast
} catch (const std::bad_any_cast& e) {
    // handle type mismatch
}
```

`std::any` is heavier than `void*` (it allocates on the heap for large types via small-buffer optimization), but it eliminates the biggest danger of `void*` — silent type mismatches that corrupt memory. For small, copyable types, consider `std::variant<Ts...>`, which is a closed-set type-safe union without heap allocation.


### void\* in C ABI and Interoperability {#void-in-c-abi-and-interoperability}

`void*` is the standard way to pass user-defined context through C-style callback APIs:

-   **POSIX threads**: `pthread_create(&thread, NULL, thread_func, void* arg)` — the `arg` is your per-thread context.
-   **C standard library**: `qsort(base, nmemb, size, compar)` — the comparator receives `const void*` pointers to elements.
-   **Dynamic library loading**: `dlopen` / `dlsym` return `void*` handles and function pointers.
-   **Godot engine**: bindings and callbacks frequently use `void*` for user-data parameters.

The fundamental limitation is that **only data pointers** can round-trip through `void*`. Function pointers, member pointers, and pointers-to-members have implementation-defined sizes and are not guaranteed to survive a `void*` conversion. On most platforms (x86-64 Linux, Windows, macOS) they happen to work, but relying on this is technically undefined behavior per the C++ standard.


### reinterpret_cast and Alignment Pitfalls {#reinterpret-cast-and-alignment-pitfalls}

When casting `void*` to a concrete type, use `static_cast` if you stored a pointer of that exact type. Use `reinterpret_cast` only when the stored address genuinely points to a different type (e.g., raw memory reinterpreted as a struct layout).

```cpp
char buffer[sizeof(int)];
void* ptr = buffer;

// static_cast is correct — we stored a char*, we get a char* back
char* cp = static_cast<char*>(ptr);

// reinterpret_cast is needed when interpreting raw bytes as int
int* ip = reinterpret_cast<int*>(ptr);  // OK, but alignment must be checked
```

Two major footguns with raw `void*` casting:

1.  **Alignment**: If a `void*` points to an address that is not a multiple of `alignof(T)`, accessing it as `T*` is undefined behavior on most architectures (SIGBUS on SPARC, potentially silent corruption on x86 with SIMD types). Always ensure alignment before casting: `assert(reinterpret_cast<uintptr_t>(ptr) % alignof(T) == 0)`.

2.  **Strict aliasing**: The compiler assumes that pointers of unrelated types never alias (point to the same memory). If you write through an `int*` and read through a `float*` that both point to the same address, the compiler may reorder or eliminate loads/stores in ways that break your program. The only legal way to alias is through `char*`, `unsigned char*`, or `std::byte*`.


### Warning {#warning}

`void*` completely removes type safety from the compiler's view. Every `void*` in your code is a place where a wrong cast can silently corrupt memory, produce garbage results, or crash far from the actual bug site.

In modern C++, `void*` should be reserved for:

-   Interfacing with C libraries that require it
-   Writing custom allocators ([C++ Memory Management]({{< relref "cpp-memory-management.md" >}}))
-   Low-level memory utilities where type erasure is the explicit goal
-   Very hot paths where `std::any` or `std::variant` overhead is unacceptable

For all other cases, prefer:

-   `std::any` (C++17) — type-erased single value with runtime checking
-   `std::variant<A, B, C>` (C++17) — closed-set type-safe union, stack-allocated
-   Templates — compile-time type erasure with zero runtime overhead ([C++ Templates]({{< relref "cpp-templates.md" >}}))
-   Smart pointers — typed ownership with automatic cleanup ([C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}}))


### Related Notes {#related-notes}

-   [C++ Templates]({{< relref "cpp-templates.md" >}}) — compile-time type erasure vs void\* runtime type erasure
-   [C++ Memory Management]({{< relref "cpp-memory-management.md" >}}) — void\* is the return type of malloc/calloc, central to allocator design
-   [C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}}) — smart pointers eliminate the need for void\* in ownership scenarios
-   [C++ Function Pointers]({{< relref "cpp-func-pointers.md" >}}) — function pointers cannot reliably be stored in void\*; use std::function
