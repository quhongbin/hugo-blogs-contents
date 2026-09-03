---
title: "C++ Function Pointers"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["pointer"]
categories: ["CPP"]
draft: false
logs: "power by AI"
---

## C++ Function Pointers {#c-plus-plus-function-pointers}

A function pointer stores the memory address of a function, allowing you to call the function indirectly. This is the foundation of callbacks, strategy patterns, plugin systems, and any situation where the code to execute is not known at compile time.

In C, function pointers are the only way to pass behavior as data. In C++, they are one of several callable types — joined by `std::function`, lambdas, and function objects (functors) — but they remain the zero-overhead, ABI-stable baseline.


### Basic Syntax {#basic-syntax}

A function pointer type encodes the return type and parameter types of the function it can point to:

```cpp
int add(int a, int b) { return a + b; }

// Function pointer: "pointer to function taking two ints and returning int"
int (*funcPtr)(int, int) = &add;       // explicit address-of
// int (*funcPtr)(int, int) = add;     // implicit function-to-pointer decay also works

int result = funcPtr(3, 4);            // call through the pointer
// int result = (*funcPtr)(3, 4);      // explicit dereference also valid
```

The syntax is famously cryptic — the parentheses around `*funcPtr` are mandatory. Without them, `int *funcPtr(int, int)` declares a **function** named `funcPtr` that returns `int*`, not a pointer to a function.


### Using Typedef {#using-typedef}

Typedef (or `using` in modern C++) makes function pointer types readable and reusable:

```cpp
// C-style typedef
typedef int (*MathFunc)(int, int);

// C++11 using alias (preferred)
using MathFunc = int (*)(int, int);

MathFunc f = add;
int result = f(3, 4);
```

This pattern is essential when passing function pointers as arguments or storing them in containers:

```cpp
using Callback = void (*)(int event_id, void* user_data);

void registerHandler(Callback cb, void* ctx);
std::vector<Callback> handlers;
```


### Factory Pattern with Function Pointers {#factory-pattern-with-function-pointers}

From practice notes: `typedef Ref(*CreateFunc)();`

This defines a type called `CreateFunc` — a pointer to a function that takes no arguments and returns `Ref` (a Godot smart pointer). This is the factory pattern: a table of creation functions that produce objects of different types, all through the same unified interface.

```cpp
typedef Ref(*CreateFunc)();

Ref createNode()      { return Ref(memnew(Node)); }
Ref createSprite()    { return Ref(memnew(Sprite)); }
Ref createAnimation() { return Ref(memnew(AnimationPlayer)); }

std::map<std::string, CreateFunc> factory = {
    {"Node",     createNode},
    {"Sprite",   createSprite},
    {"Animation", createAnimation},
};

Ref obj = factory["Sprite"]();  // creates a Sprite through the factory
```

This pattern is used extensively in Godot's class registration system, plugin architectures, and serialization/deserialization frameworks.


### std::function — Type-Erased Callable Wrapper {#std-function-type-erased-callable-wrapper}

`std::function<R(Args...)>` is a C++11 class template that can hold **any** callable with a compatible signature — function pointers, lambdas, functors, `std::bind` results — in a type-erased wrapper.

```cpp
#include <functional>

std::function<int(int, int)> op;

op = add;                                  // function pointer
op = [](int a, int b) { return a * b; };   // lambda
op = std::multiplies<int>{};                // standard functor

int result = op(3, 4);  // works regardless of what 'op' holds
```

`std::function` achieves this through **type erasure** — internally it stores a pointer to a heap-allocated wrapper that knows how to call the actual callable. This flexibility has a cost:

-   **Size**: typically 32 bytes on 64-bit (vs 8 bytes for a raw function pointer) — it uses small-buffer optimization to avoid heap allocation for tiny callables.
-   **Indirection**: calling through `std::function` involves a virtual-call-like dispatch, which is slower than a direct function pointer call and cannot be inlined.
-   **Heap allocation**: large callables (lambdas with many captures) require a heap allocation inside `std::function`.

Use `std::function` for storage and interface boundaries (class members, callback registries). Use templates for hot paths where the callable type is known at compile time — the compiler can inline through a template parameter but not through `std::function`.


### Lambdas, Captures, and Closures {#lambdas-captures-and-closures}

C++11 lambdas are syntactic sugar for generating anonymous function objects (functors) at the point of use. The compiler generates a unique class with an `operator()`, and the capture list determines which variables from the enclosing scope become data members.

```cpp
int threshold = 10;
auto pred = [threshold](int x) { return x > threshold; };
// Compiler generates (conceptually):
// struct __lambda { int threshold; bool operator()(int x) const { return x > threshold; } };
```

Capture modes:

| Syntax               | Meaning                                             |
|----------------------|-----------------------------------------------------|
| `[=]`                | Capture all used variables by copy                  |
| `[&]`                | Capture all used variables by reference             |
| `[x]`                | Capture `x` by copy                                 |
| `[&x]`               | Capture `x` by reference                            |
| `[this]`             | Capture `this` by reference (C++17)                 |
| `[*this]`            | Capture `*this` by copy (C++17) — copies the object |
| `[=, &x]`            | Copy all except `x` (captured by ref)               |
| `[x = std::move(x)]` | Move capture (C++14 init capture)                   |

**Lambdas are not function pointers** — a lambda with captures generates a class with data members, which cannot convert to a raw function pointer. A captureless lambda **can** convert to a function pointer because it is stateless:

```cpp
int (*fp)(int, int) = [](int a, int b) { return a + b; };  // OK — no captures

int n = 42;
// int (*fp2)(int) = [n](int x) { return x + n; };  // ERROR: lambda with capture ≠ function ptr
std::function<int(int)> f2 = [n](int x) { return x + n; }; // OK — std::function wraps it
```


### Member Function Pointers {#member-function-pointers}

Member function pointers are a distinct category in C++ — they are not convertible to regular function pointers because they need a `this` object to be called on.

```cpp
struct Widget {
    void draw(int x, int y);
    int  getId() const;
};

// Pointer to member function: "void (Widget::*)(int, int)"
void (Widget::*drawPtr)(int, int) = &Widget::draw;

Widget w;
(w.*drawPtr)(10, 20);     // call on object — note the .* operator

Widget* pw = &w;
(pw->*drawPtr)(10, 20);   // call on pointer — note the ->* operator
```

Member function pointers are typically twice the size of regular pointers (16 bytes on 64-bit) because they must handle virtual functions (which require a vtable offset, not a code address). Calling through a member function pointer is slower than a direct call — the compiler must check whether the function is virtual and dispatch accordingly.

C++17's `std::invoke` provides a uniform syntax for calling any callable, including member function pointers:

```cpp
#include <functional>
std::invoke(&Widget::draw, w, 10, 20);   // unified call syntax
```


### Performance: Function Pointer vs std::function vs Lambda {#performance-function-pointer-vs-std-function-vs-lambda}

Calling through different callable abstractions has different costs:

| Mechanism            | Call cost       | Inlineable? | Storage size (64-bit) |
|----------------------|-----------------|-------------|-----------------------|
| Direct function call | ~0 (inlined)    | Always      | N/A                   |
| Template parameter   | ~0 (inlined)    | Always      | N/A                   |
| Lambda (known type)  | ~0 (inlined)    | Always      | Varies (captures)     |
| Raw function pointer | 1 indirection   | Rarely      | 8 bytes               |
| std::function        | 2+ indirections | Never       | 32 bytes (typical)    |
| Virtual function     | 1 indirection   | Sometimes   | 8 bytes (vtable ptr)  |

The key insight: when the callable type is known at compile time (templates, auto-lambda), the compiler can inline the call, eliminating all overhead. When the type is erased (function pointer, `std::function`), the compiler cannot see through the indirection and must emit an actual call instruction.

For performance-critical callbacks, prefer templates over `std::function`:

```cpp
// FAST — compiler knows Pred type, can inline
template <typename Pred>
void forEach(FastVector& v, Pred pred) { ... }

// SLOW — type-erased, cannot inline the call
void forEach(FastVector& v, std::function<void(Item&)> pred) { ... }
```


### Calling Conventions {#calling-conventions}

The **calling convention** defines how function arguments are passed (registers vs stack), who cleans the stack, and how the return value is delivered. They are platform-specific and ABI-critical:

| Convention   | Common on   | Arg passing            | Stack cleanup | Notes                               |
|--------------|-------------|------------------------|---------------|-------------------------------------|
| `__cdecl`    | x86 Windows | Right-to-left stack    | Caller        | Default for C/C++; supports varargs |
| `__stdcall`  | x86 Windows | Right-to-left stack    | Callee        | Used by Win32 API                   |
| `__fastcall` | x86 Windows | First 2 in ECX/EDX     | Callee        | Rarely used; compiler-specific      |
| `__thiscall` | x86 Windows | `this` in ECX          | Callee        | Non-static member functions         |
| x86-64 SysV  | Linux/macOS | First 6 in RDI/RSI/... | Caller        | Single unified convention           |
| x86-64 MS    | Windows     | First 4 in RCX/RDX/... | Caller        | Single unified convention           |

On x86-64, calling conventions are largely unified (one convention per OS), but on x86 (32-bit) they differ and mismatches cause stack corruption. This is why casting between function pointer types with different calling conventions is undefined behavior — the caller and callee disagree on who cleans the stack and in what order arguments were pushed.


### Related Notes {#related-notes}

-   [C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}}) — custom deleters can be function pointers; factory functions return smart pointers
-   [C++ Templates]({{< relref "cpp-templates.md" >}}) — templates subsume many function-pointer use cases (std::function is a template)
-   [C++ void\* Pointers]({{< relref "cpp-void-ptr.md" >}}) — function pointers and data pointers (void\*) are distinct types
-   [C++ Memory Management]({{< relref "cpp-memory-management.md" >}}) — factory functions with function pointers manage ownership
-   [Copy Constructors]({{< relref "cpp-copy-constructors.md" >}}) — lambdas with copy captures invoke copy constructors
-   [C++ Enum vs Struct Class]({{< relref "cpp-enum-struct.md" >}}) — enums as non-type template parameters can select function-pointer specializations
