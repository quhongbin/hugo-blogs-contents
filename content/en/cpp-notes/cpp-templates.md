---
author: Hongbin Qu
title: C++ Templates
tags: [cs, C++]
categories: []
---

# C++ Templates

Templates are C++\'s mechanism for **generic programming** --- writing
code that works with any type without repeating yourself. Unlike runtime
polymorphism (virtual functions), templates are resolved entirely at
compile time: the compiler generates a separate copy of the function or
class for each set of type arguments actually used. This is called
**template instantiation**.

Templates are the foundation of the C++ Standard Library:
`std::vector<int>`, `std::map<std::string, double>`,
`std::sort(begin, end)` --- all templates. They power everything from
containers to algorithms to smart pointers. Understanding templates
deeply means understanding how modern C++ achieves its combination of
abstraction and zero-overhead performance.

[[cpp-void-ptr]]

## Basic Syntax

A function template parameterizes a function over one or more types (or
non-type values).

```cpp
template <typename T>
T max(T a, T b) {
    return (a > b) ? a : b;
}

// Usage — the compiler deduces T from the argument types
int    x = max(3, 5);       // T = int
double y = max(3.14, 2.72); // T = double
```

Class templates parameterize entire classes:

```cpp
template <typename T>
class Stack {
    std::vector<T> data;
public:
    void push(const T& val) { data.push_back(val); }
    T    pop()              { T v = data.back(); data.pop_back(); return v; }
    bool empty() const      { return data.empty(); }
};

Stack<int>         intStack;
Stack<std::string> strStack;
```

Template parameters can also be non-type values --- integers, enums,
pointers, or (in C++20) floating-point and class types:

```cpp
template <typename T, size_t N>
class Array {
    T data[N];
public:
    size_t size() const { return N; }
};

Array<int, 100> buf;  // stack-allocated array of 100 ints
```

## Templates in Godot

Godot uses templates heavily throughout its core. A prime example is the
memory allocation layer: `Memory::alloc_static<true>(size)` uses a
`bool` template parameter to decide **at compile time** whether to
zero-fill the allocated memory.

This compile-time dispatch means no runtime branch, no `if` check, no
function-pointer indirection --- the compiler generates two versions,
and the one actually called is resolved during compilation. This pattern
is called **static polymorphism** and is a key performance technique in
game engines.

Another critical Godot template is `Ref<T>` ([C++ Smart Pointers &
Ref](id:1d07decd-4659-494a-835e-defca639a771)), which is a
reference-counted smart pointer template. It only accepts classes
derived from `RefCounted`, enforced through template constraints.

## Template Specialization --- Full and Partial

You can provide a specialized implementation for specific types or
families of types. **Full specialization** replaces the template
entirely for one concrete type:

```cpp
// Primary template
template <typename T>
struct TypeName {
    static const char* get() { return "unknown"; }
};

// Full specialization for int
template <>
struct TypeName<int> {
    static const char* get() { return "int"; }
};

// Full specialization for double
template <>
struct TypeName<double> {
    static const char* get() { return "double"; }
};
```

**Partial specialization** specializes for a family of types --- for
example, all pointer types, all `std::vector<T>`, or all types matching
a pattern:

```cpp
// Partial specialization: matches T* for any T
template <typename T>
struct TypeName<T*> {
    static const char* get() { return "pointer"; }
};

// Partial specialization: matches std::vector<T> for any T
template <typename T>
struct TypeName<std::vector<T>> {
    static const char* get() { return "vector"; }
};
```

The compiler always picks the **most specialized** match. Partial
specialization only works for class templates, not function templates
(function templates use overloading instead).

## SFINAE and enable~if~

SFINAE stands for \"Substitution Failure Is Not An Error\" --- when the
compiler tries to substitute template parameters and the substitution
fails (e.g., a type doesn\'t have a required member), it does not
produce an error. Instead, it simply removes that overload from
consideration and tries the next one.

This enables conditional template instantiation --- including or
excluding overloads based on type properties:

```cpp
// Enable this overload only for integral types
template <typename T>
typename std::enable_if<std::is_integral<T>::value, T>::type
divide(T a, T b) {
    return a / b;  // integer division
}

// Enable this overload only for floating-point types
template <typename T>
typename std::enable_if<std::is_floating_point<T>::value, T>::type
divide(T a, T b) {
    return a / b;  // floating-point division
}
```

C++14 simplified this with `std::enable_if_t`, and C++17 introduced
`if constexpr` which often replaces SFINAE for simpler cases:

```cpp
template <typename T>
T divide(T a, T b) {
    if constexpr (std::is_integral_v<T>) {
        return a / b;  // integer path
    } else {
        return a / b;  // floating-point path
    }
}
```

SFINAE is still essential for controlling which template overloads exist
--- `if constexpr` can\'t remove an overload from the overload set, it
can only branch inside a single instantiation.

## Variadic Templates and Parameter Packs

C++11 introduced **variadic templates** --- templates that accept an
arbitrary number of type or value parameters. The syntax uses `...`
(ellipsis) to denote a **parameter pack**:

```cpp
// Base case — recursion termination
void print() { std::cout << '\n'; }

// Variadic — one or more arguments
template <typename T, typename... Args>
void print(T first, Args... rest) {
    std::cout << first << ' ';
    print(rest...);  // recursive expansion
}

print(1, 2.5, "hello", 'c');  // prints: 1 2.5 hello c
```

C++17 introduced **fold expressions**, which simplify operations over
parameter packs:

```cpp
template <typename... Args>
auto sum(Args... args) {
    return (args + ...);  // fold: args1 + args2 + ... + argsN
}

auto total = sum(1, 2, 3, 4, 5);  // total = 15
```

Variadic templates power `std::tuple`, `std::variant`,
`std::make_shared`, `emplace_back`, and nearly every variadic function
in the standard library. They eliminate the pre-C++11 practice of
copy-pasting overloads for 1 arg, 2 args, 3 args, ... up to some
arbitrary limit.

## C++20 Concepts

C++20 **Concepts** are a major evolution of templates. They let you
specify **requirements** on template parameters, producing readable
error messages when a type doesn\'t satisfy those requirements ---
instead of pages of inscrutable template instantiation backtraces.

```cpp
// Define a concept: T must be comparable with <
template <typename T>
concept Comparable = requires(T a, T b) {
    { a < b } -> std::convertible_to<bool>;
};

// Use the concept in place of typename
template <Comparable T>
T max(T a, T b) {
    return (a > b) ? a : b;
}

struct NotComparable {};
// max(NotComparable{}, NotComparable{});  // ERROR: NotComparable does not satisfy Comparable
```

Concepts can also be used with the terser `auto` syntax:

```cpp
auto max(Comparable auto a, Comparable auto b) {
    return (a > b) ? a : b;
}
```

Concepts subsume most use cases of SFINAE and `enable_if`, providing
clearer intent, better error messages, and faster compilation (the
compiler can reject non-matching overloads earlier).

## Key Point

Templates execute entirely at compile time. Each unique set of template
arguments produces a separate instantiation --- separate machine code in
the binary. This means:

- **Zero runtime overhead** --- no virtual dispatch, no type erasure
  wrappers, no indirection. Template code is as fast as hand-written
  code for each specific type.
- **Potentially larger binaries** --- if you instantiate
  `std::vector<int>`, `std::vector<double>`, and
  `std::vector<std::string>`, you get three copies of `vector`\'s
  code. This is called **code bloat**. In practice, modern linkers can
  fold identical instantiations (e.g., `vector<int*>` and
  `vector<const int*>` on some platforms).
- **Longer compile times** --- each instantiation requires the
  compiler to parse, type-check, and codegen the template body for
  each combination. Large template-heavy codebases (LLVM, Godot) can
  take minutes or hours to build.
- **Error messages at instantiation, not definition** --- if a
  template body contains an error that only surfaces for specific
  types (e.g., `T::value` when T is `int`), the error only appears
  when you actually instantiate with `int`. This makes debugging
  template errors notoriously difficult --- a key motivation for C++20
  Concepts.

## Related Notes

- [C++ Smart Pointers & Ref](id:1d07decd-4659-494a-835e-defca639a771)
  --- Ref\<T\>, unique~ptr~\<T\>, shared~ptr~\<T\> are all class
  templates
- [C++ void\* Pointers](id:dae30869-f9e6-43e5-b188-661d59b714c4) ---
  compile-time type erasure (templates) vs runtime type erasure
  (void\*)
- [C++ Function Pointers](id:470066fc-cee3-48da-a217-9abc41c4818d) ---
  std::function is a class template; function pointer types can be
  template parameters
- [C++ Enum vs Struct Class](id:2ae84663-9f28-426d-a4eb-a8ffebae71b2)
  --- enums and structs work as template type parameters and non-type
  template arguments
- [C++ Memory Management](id:d90d89de-f807-4187-a9f7-a1501e9dcda5) ---
  custom allocators are templated on the type they allocate
- [Copy Constructors](id:f5899ea2-320f-4845-8ed3-68b68ba1cdcd) ---
  templates interact with copy semantics; template constructors are
  never copy constructors
