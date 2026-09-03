---
title: "C++ Memory Management"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["Memory Management"]
categories: ["CPP"]
draft: false
logs: "power by AI"
---

## C++ Memory Management {#c-plus-plus-memory-management}

C++ gives the programmer direct, fine-grained control over memory — where objects live, how long they survive, and when they are reclaimed. This is both C++'s greatest strength (zero-overhead resource control, no mandatory GC) and its greatest source of bugs (use-after-free, double-free, memory leaks, buffer overflows).

This note serves as the hub for all memory-related topics. Modern C++ (C++11 onward) provides tools — smart pointers, RAII, move semantics — that make manual memory management largely unnecessary for application code while preserving the low-level control for systems programming and game engines.


### Stack vs Heap — Detailed Comparison {#stack-vs-heap-detailed-comparison}

C++ objects can live in two fundamentally different memory regions:

| Property             | Stack                                        | Heap                                        |
|----------------------|----------------------------------------------|---------------------------------------------|
| **Allocation speed** | ~1 CPU instruction (move stack pointer)      | Hundreds of cycles (`malloc~/~new`)         |
| **Deallocation**     | Automatic (stack pointer restored on return) | Manual (`delete~/~free`) or smart pointer   |
| **Lifetime**         | Tied to scope (`{}`)                         | Arbitrary — lives until explicitly freed    |
| **Size limit**       | Typically 1-8 MB (OS-dependent)              | Limited by system RAM + swap                |
| **Fragmentation**    | None (LIFO discipline)                       | Possible — external fragmentation           |
| **Cache locality**   | Excellent (contiguous, hot in cache)         | Variable — depends on allocation pattern    |
| **Thread safety**    | Per-thread — no contention                   | Allocator must be thread-safe (`malloc` is) |

```cpp
void example() {
    int stack_var = 42;                    // stack — ~0 cost, auto-freed
    int* heap_var = new int(42);           // heap — malloc + constructor overhead
    delete heap_var;                       // must free or leak
}   // stack_var dies here automatically
```

General rule: **prefer stack allocation**. It is faster, safer, and automatic. Use the heap only when:

-   The object's size is unknown at compile time (dynamic arrays, strings).
-   The object must outlive the creating scope (factory functions, shared state).
-   The object is too large for the stack (megabytes+).
-   You need runtime polymorphism (storing derived objects in a base-class container).


### Memory Alignment and Padding {#memory-alignment-and-padding}

Every C++ type has an **alignment requirement** — the address at which it must begin must be a multiple of its alignment. This exists because CPUs read memory in aligned chunks (typically 4, 8, 16, or 32 bytes), and misaligned access is either slower (x86) or a hard fault (ARM, SPARC).

```cpp
struct Misaligned {
    char   c;   // 1 byte,  alignment 1
    int    i;   // 4 bytes, alignment 4
    double d;   // 8 bytes, alignment 8
};

// Actual layout (compiler inserts padding):
// offset 0: c (1 byte)
// offset 1-3: padding (3 bytes)   ← pad to align int at offset 4
// offset 4-7: i (4 bytes)
// offset 8-15: d (8 bytes)
// Total: 16 bytes

std::cout << sizeof(Misaligned);  // 16, not 1+4+8=13
```

You can query alignment at compile time with `alignof(T)` and override it with `alignas(N)`:

```cpp
alignas(64) char cache_line[64];  // force 64-byte alignment (prevents false sharing)
```

Alignment matters for performance (cache-line alignment avoids false sharing in multi-threaded code) and for correctness when doing low-level memory work (casting `void*` to typed pointers — see [C++ void\* Pointers]({{< relref "cpp-void-ptr.md" >}})).


### RAII — Resource Acquisition Is Initialization {#raii-resource-acquisition-is-initialization}

RAII is the single most important idiom in C++. The core idea: **bind the lifetime of a resource to the lifetime of an object**. Acquire the resource in the constructor, release it in the destructor. The destructor is guaranteed to run when the object goes out of scope — whether by normal return, exception, or early return.

```cpp
class FileHandle {
    FILE* f;
public:
    FileHandle(const char* path) : f(fopen(path, "r")) {
        if (!f) throw std::runtime_error("open failed");
    }
    ~FileHandle() { if (f) fclose(f); }  // guaranteed cleanup
};

void readConfig() {
    FileHandle fh("config.json");  // file opened
    // ... read and parse ...
    // exception thrown here? fclose still called
}  // fh's destructor runs → fclose called
```

RAII applies to any resource, not just memory:

-   **File handles** (`std::fstream`, `std::FILE*`)
-   **Locks** (`std::lock_guard`, `std::unique_lock`)
-   **Sockets** and network connections
-   **GPU resources** (textures, buffers, shaders)
-   **Database transactions** (commit on success, rollback on exception)

Smart pointers ([C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}})) are the canonical RAII wrapper for heap memory: the pointer is the resource, the constructor takes ownership, the destructor calls `delete`.


### Custom Allocators and Memory Pools {#custom-allocators-and-memory-pools}

The default heap allocator (`malloc~/~new`) is a general-purpose solution that must handle arbitrary sizes, fragmentation, and multi-threading. For performance-critical code, **custom allocators** can dramatically outperform the general allocator by exploiting domain-specific allocation patterns:

-   **Bump allocator** (arena): maintains a pointer into a pre-allocated block. Allocation is a pointer increment — 1-2 instructions. Deallocation is all-or-nothing (the entire arena is freed at once). Ideal for per-frame allocations in game engines or request-scoped allocations in servers.

-   **Pool allocator** (slab): pre-allocates many fixed-size slots. Allocation is popping from a free list (O(1)). Deallocation returns the slot to the free list. No fragmentation by design. Ideal when you allocate many objects of the same size (game entities, network packets).

-   **Stack allocator**: like a bump allocator but supports LIFO deallocation (free the most recent allocation only). Used in Godot and Unreal Engine for temporary per-frame data.

-   **STL allocator adaptor**: all C++ standard containers accept a custom allocator via their template parameter. `std::vector<int, MyAllocator<int>>` will use your allocator for all its heap operations.


### Placement new and In-Place Construction {#placement-new-and-in-place-construction}

Placement `new` constructs an object at a specific memory address without allocating new memory. It is the fundamental building block of custom allocators, memory pools, and container implementations:

```cpp
#include <new>

alignas(Widget) char buffer[sizeof(Widget)];  // stack-allocated raw storage

Widget* w = new (buffer) Widget(42, "name");  // construct Widget in buffer
w->doSomething();
w->~Widget();  // explicit destructor call — no delete (delete would free the stack buffer!)
```

Critically, placement `new` returns the same pointer it was given — the address is already known. The only effect is calling the constructor. The matching cleanup is an **explicit destructor call**, not `delete` (which would try to `free` the stack-allocated buffer).

Standard containers use placement `new` internally: `std::vector::emplace_back` constructs the new element directly in the pre-allocated storage at the end of the vector, avoiding a temporary and a move.


### Key Topics {#key-topics}

-   [C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}}) — automatic memory management through RAII; unique_ptr, shared_ptr, weak_ptr, and Godot's Ref&lt;T&gt;
-   [Reference Counting]({{< relref "cpp-ref-counting.md" >}}) — shared ownership tracking; atomic vs non-atomic; intrusive vs non-intrusive
-   [C++ void\* Pointers]({{< relref "cpp-void-ptr.md" >}}) — raw untyped memory access; the foundation of `malloc~/~free` and custom allocators
-   [Copy Constructors]({{< relref "cpp-copy-constructors.md" >}}) — controlling how objects are copied; deep vs shallow copy; move semantics


### Related Language Features {#related-language-features}

-   [C++ Templates]({{< relref "cpp-templates.md" >}}) — custom allocators are templated on the type they allocate
-   [C++ Function Pointers]({{< relref "cpp-func-pointers.md" >}}) — factory functions and custom deleters use function pointers
-   [C++ Enum vs Struct Class]({{< relref "cpp-enum-struct.md" >}}) — struct layout and alignment relate to memory layout and padding


### Key Rule {#key-rule}

Always pair `new` with `delete` and `malloc` with `free`. Never mix them (`new` + `free`, or `malloc` + `delete`). In modern C++, avoid writing raw `new` and `delete` in application code entirely — use `std::make_unique`, `std::make_shared`, and standard containers (`std::vector`, `std::string`, `std::map`) which handle allocation and deallocation correctly by default.

If you find yourself writing `new` or `delete` outside of a custom allocator, a resource-handle class, or very low-level code, pause and ask: can a smart pointer or standard container do this instead? The answer is almost always yes.
