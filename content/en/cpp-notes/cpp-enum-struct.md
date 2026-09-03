---
title: "C++ Enum vs Struct Class"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["Data Type", "Data Struct"]
categories: ["CPP"]
draft: false
logs: "power by AI"
---

## C++ Enum vs Struct Class {#c-plus-plus-enum-vs-struct-class}

`enum` and `struct` are both user-defined types in C++, but they serve fundamentally different roles: `enum` defines a set of named constants (values), while `struct` defines a compound data type (a bundle of data members). Understanding both — and their modern variants — is essential for writing clear, type-safe C++.


### Enum {#enum}

An `enum` (enumeration) is a distinct type whose values are restricted to a set of named constants. C++ has two enum forms:

```cpp
enum Color { RED, GREEN, BLUE };        // "plain" enum (C-compatible)
enum class Direction { UP, DOWN, LEFT, RIGHT }; // "scoped" enum (C++11)
```

Plain enums are legacy from C and have significant drawbacks: their enumerators leak into the surrounding scope (RED, not Color::RED), and they implicitly convert to `int`, enabling nonsensical operations like `Color c = RED + 5`.

`enum class` (scoped enumeration) fixes both issues: enumerators are accessed with the scope operator (`Direction::UP`), and there is no implicit conversion to `int`.


### Struct {#struct}

A `struct` (structure) groups related data members into a single compound type. In C++, `struct` and `class` are nearly identical — the only difference is that `struct` members are `public` by default, while `class` members are `private` by default.

```cpp
struct Point {
    double x;  // public by default
    double y;
};

class Circle {
    double radius;  // private by default
public:
    Circle(double r) : radius(r) {}
    double area() const { return 3.14159 * radius * radius; }
};
```

The convention in most C++ codebases: use `struct` for passive data holders (PODs, aggregates, value types) and `class` for types with invariants, private state, and behavior. This is a convention only — the compiler treats them identically after access specifiers are resolved.


### Plain Enum vs Enum Class {#plain-enum-vs-enum-class}

| Feature                 | `enum` (plain)                                | `enum class` (scoped)                  |
|-------------------------|-----------------------------------------------|----------------------------------------|
| Enumerator scope        | Enclosing scope (RED)                         | Enum scope (Color::RED)                |
| Implicit int conversion | Yes                                           | No (requires `static_cast`)            |
| Forward declaration     | No (C++11: yes with fixed underlying type)    | Yes                                    |
| Underlying type         | Implementation-defined (not smaller than int) | `int` by default, can be specified     |
| Name collisions         | Possible (two enums can't share RED)          | Safe (Direction::UP ≠ Orientation::UP) |
| Comparison with int     | Allowed (Color::RED == 0)                     | Not allowed                            |

The scoping and type-safety improvements are so significant that most C++ style guides (Google, LLVM, C++ Core Guidelines) recommend always using `enum class` unless C compatibility is required.


### Enum Underlying Type and std::underlying_type {#enum-underlying-type-and-std-underlying-type}

Both `enum` and `enum class` allow specifying the underlying integer type explicitly. This is critical for ABI stability (the enum's size becomes part of the interface) and for serialization (you know the exact byte layout):

```cpp
enum class Permission : uint8_t {
    READ    = 0x01,
    WRITE   = 0x02,
    EXECUTE = 0x04
};
static_assert(sizeof(Permission) == 1);  // exactly one byte

enum class ErrorCode : int32_t {
    OK = 0,
    NotFound = -1,
    PermissionDenied = -2
};
```

`std::underlying_type_t<E>` retrieves the underlying type at compile time — essential for template metaprogramming and serialization frameworks:

```cpp
template <typename E>
constexpr auto to_underlying(E e) {
    return static_cast<std::underlying_type_t<E>>(e);
}

auto val = to_underlying(Permission::READ);  // val = 1 (uint8_t)
```


### Enum as Bitmask / Flags Pattern {#enum-as-bitmask-flags-pattern}

When an enum represents bit flags (each value is a power of 2), you can combine them with bitwise operators — but you must overload those operators manually for `enum class`:

```cpp
enum class FileMode : uint8_t {
    Read    = 1 << 0,  // 0x01
    Write   = 1 << 1,  // 0x02
    Append  = 1 << 2,  // 0x04
    Binary  = 1 << 3,  // 0x08
};

// Enable bitwise operators for enum class flags
constexpr FileMode operator|(FileMode a, FileMode b) {
    return static_cast<FileMode>(
        static_cast<uint8_t>(a) | static_cast<uint8_t>(b)
    );
}

constexpr FileMode operator&(FileMode a, FileMode b) {
    return static_cast<FileMode>(
        static_cast<uint8_t>(a) & static_cast<uint8_t>(b)
    );
}

constexpr bool hasFlag(FileMode flags, FileMode flag) {
    return (flags & flag) == flag;
}

// Usage
auto mode = FileMode::Read | FileMode::Binary;  // 0x09
if (hasFlag(mode, FileMode::Read)) { /* ... */ }
```

This pattern is used extensively in Godot (e.g., `MethodFlags`, `PropertyUsageFlags`), Windows API (`FILE_ATTRIBUTE_*`, `GENERIC_READ | GENERIC_WRITE`), and almost every C API that accepts flag combinations.


### Struct Padding, Alignment, and Reordering {#struct-padding-alignment-and-reordering}

The compiler inserts **padding bytes** between struct members to satisfy alignment requirements. This means the size of a struct is not simply the sum of its members' sizes — and the order of members affects the total size:

```cpp
struct Bad {       // 24 bytes on 64-bit
    char   a;      // 1 byte  + 7 padding (align double to multiple of 8)
    double b;      // 8 bytes
    char   c;      // 1 byte  + 7 padding (align struct size to largest alignment)
};

struct Good {      // 16 bytes — same data, reordered
    double b;      // 8 bytes
    char   a;      // 1 byte
    char   c;      // 1 byte
    // + 6 padding for alignment
};

// Rule: order members by descending alignment (largest first) to minimize padding
static_assert(sizeof(Bad)  == 24);
static_assert(sizeof(Good) == 16);
```

Key alignment rules:

-   Each member must be at an offset that is a multiple of its `alignof`.
-   The struct's total size must be a multiple of the largest member's alignment (so arrays of structs are properly aligned).
-   A struct's alignment equals the maximum alignment of its members.

C++11 added `alignof(T)` and `alignas(N)` for explicit alignment queries and requirements. You can use `alignas` to force cache-line alignment (prevents false sharing in multi-threaded code) or to satisfy hardware requirements (SIMD types often require 16- or 32-byte alignment).


### Designated Initializers (C++20) {#designated-initializers--c-plus-plus-20}

C++20 introduced **designated initializers**, allowing you to initialize struct (aggregate) members by name, skipping defaults and improving readability:

```cpp
struct WindowConfig {
    int    width   = 800;
    int    height  = 600;
    const char* title = "Untitled";
    bool   resizable = true;
    double opacity   = 1.0;
};

// With designated initializers (C++20)
WindowConfig cfg = {
    .title = "My App",
    .opacity = 0.8
    // width, height, resizable use their defaults
};

// Pre-C++20: positional only — must list all values up to the last non-default
WindowConfig cfg2 = {800, 600, "My App", true, 0.8};  // fragile, easy to reorder
```

C++ designated initializers have a key restriction compared to C: members must be initialized in declaration order. This prevents ambiguities but means you cannot reorder the designators arbitrarily.


### Union — Shared Memory, Type Punning, std::variant {#union-shared-memory-type-punning-std-variant}

A `union` is a special struct-like type where all members share the same memory — at any given time, only one member (the "active member") is valid:

```cpp
union Value {
    int    i;
    double d;
    char   str[16];
};  // sizeof(Value) == 16 (largest member)

Value v;
v.i = 42;     // active member is 'i'
// v.d = 3.14; // now 'i' is overwritten; reading 'i' is undefined behavior
```

Unions are useful for memory-constrained systems, representing tagged/variant types, and low-level type punning (reinterpreting bits as a different type). However, raw `union` is error-prone — nothing tells you which member is active.

C++17's `std::variant<A, B, C>` is a type-safe tagged union that tracks the active member and throws on incorrect access:

```cpp
std::variant<int, double, std::string> val;

val = 42;                            // holds int
auto i = std::get<int>(val);         // OK
// auto d = std::get<double>(val);   // throws std::bad_variant_access

// Pattern matching with std::visit (C++17)
std::visit([](auto&& arg) {
    std::cout << arg << '\n';
}, val);
```

`std::variant` compiles to essentially the same layout as a hand-written tagged union (`sizeof(variant) = sizeof(largest type) + sizeof(tag) + padding`) but provides full type safety. Prefer `std::variant` over raw `union` in modern C++.


### Recommendation {#recommendation}

Always prefer `enum class` over plain `enum` in modern C++. It avoids name clashes, prevents implicit int conversion bugs, and integrates better with the type system. Similarly, prefer `std::variant` over raw `union` for tagged union use cases. Use `struct` for simple data aggregates and `class` for types with invariants and behavior — this is a strong convention that makes code intent clear at a glance.
