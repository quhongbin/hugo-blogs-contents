---
title: "nullptr"
author: ["Hongbin Qu"]
date: 2026-07-11T00:00:00+08:00
tags: ["CPP"]
draft: false
logs: "power by AI"
---

## C++ nullptr {#c-plus-plus-nullptr}

> This word is supplied for C++11 standard


### Background and Motivation {#background-and-motivation}

In C program language, NULL is a constant integral and macro, the NULL is placed 0 or (void\*)0 in pre-handle time.

In C++ nullptr is a basic type nullptr_t, NULL is number of 0(as it stronge language with forbidding void\* convert onther pointers of types implicitly)

> (void\*)0 means that the number 0 was converted pointer of void, it's behalf of any types(int,char,so forth). it indicate that virtual address was marked non-write,non-read,non-execute on the top of User Address Space


### Problems with NULL in C++ {#problems-with-null-in-c-plus-plus}

The traditional NULL macro has several issues in C++:


#### 1. Ambiguity with Integer Zero {#1-dot-ambiguity-with-integer-zero}

NULL can be implicitly converted to both pointer types and integer types, leading to unexpected behavior:

```cpp
void foo(int x) { cout << "int version" << endl; }
void foo(char* p) { cout << "pointer version" << endl; }

foo(NULL);  // Ambiguous! Calls foo(int), not foo(char*)
```


#### 2. Type Safety Issues {#2-dot-type-safety-issues}

NULL doesn't have a distinct type, making it impossible to distinguish between a null pointer and the integer 0:

```cpp
int* ptr = NULL;
if (ptr == 0) { ... }  // Compiles, but conceptually confusing
if (ptr == NULL) { ... }  // Same as above
```


#### 3. Template Deduction Problems {#3-dot-template-deduction-problems}

Templates cannot properly deduce types when NULL is used:

```cpp
template<typename T>
void process(T* ptr);

process(NULL);  // Error: cannot deduce T
```


### The nullptr Solution {#the-nullptr-solution}

C++11 introduced `nullptr` as a keyword with type `std::nullptr_t` to solve these problems:


#### Key Characteristics {#key-characteristics}

-   **Type-safe**: `nullptr` has its own type `std::nullptr_t`
-   **Implicit conversion**: Can be implicitly converted to any pointer type
-   **No integer conversion**: Cannot be implicitly converted to integer types
-   **Boolean conversion**: Converts to `false` in boolean context
-   **Size**: `sizeof(nullptr_t)` is typically 8 bytes on 64-bit systems


#### Usage Examples {#usage-examples}

```cpp
#include <iostream>
#include <cstddef>

// Function overloading works correctly now
void process(int x) {
    std::cout << "int version: " << x << std::endl;
}

void process(char* ptr) {
    std::cout << "pointer version" << std::endl;
}

int main() {
    int* ptr1 = nullptr;      // OK
    char* ptr2 = nullptr;     // OK
    void* ptr3 = nullptr;     // OK

    process(nullptr);         // Calls process(char*), unambiguous

    // nullptr is not an integer
    // int x = nullptr;       // Error!
    // process(0);            // Calls process(int)

    // Boolean context
    if (ptr1 == nullptr) {
        std::cout << "ptr1 is null" << std::endl;
    }

    if (!ptr2) {
        std::cout << "ptr2 is null" << std::endl;
    }

    return 0;
}
```


#### Template Deduction {#template-deduction}

```cpp
template<typename T>
void process(T* ptr) {
    if (ptr == nullptr) {
        std::cout << "null pointer" << std::endl;
    }
}

int main() {
    int* p = nullptr;
    process(p);              // T deduced as int
    process(nullptr);        // Still ambiguous, need explicit cast
    process(static_cast<int*>(nullptr));  // OK
    return 0;
}
```


### nullptr_t Type {#nullptr-t-type}

`std::nullptr_t` is defined in `<cstddef>` header:

```cpp
#include <cstddef>

void func(std::nullptr_t) {
    std::cout << "nullptr_t version" << std::endl;
}

int main() {
    func(nullptr);  // Calls func(std::nullptr_t)
    return 0;
}
```


### Comparison: NULL vs nullptr {#comparison-null-vs-nullptr}

| Feature             | NULL (C-style)         | nullptr (C++11)           |
|---------------------|------------------------|---------------------------|
| Type                | Macro (0 or (void\*)0) | std::nullptr_t            |
| Type safety         | Low                    | High                      |
| Overload resolution | Ambiguous              | Unambiguous               |
| Template deduction  | Fails                  | Works (with explicit ptr) |
| Integer conversion  | Implicit               | Not allowed               |
| Pointer conversion  | Implicit               | Implicit                  |


### Best Practices {#best-practices}

1.  **Always use nullptr** for null pointer values in C++11 and later
2.  **Avoid NULL macro** in modern C++ code
3.  **Use nullptr in comparisons** instead of checking against 0
4.  **Initialize pointers to nullptr** when declaring them without initialization
5.  **Use nullptr with smart pointers** (`std::unique_ptr`, `std::shared_ptr`)

<!--listend-->

```cpp
// Good practices
int* ptr = nullptr;                    // Initialize to nullptr
if (ptr != nullptr) { ... }            // Explicit comparison
std::unique_ptr<int> uptr = nullptr;   // Works with smart pointers
std::shared_ptr<int> sptr = nullptr;   // Works with smart pointers

// Avoid
int* ptr2 = NULL;                      // Old style
int* ptr3 = 0;                         // Confusing
if (ptr) { ... }                       // Less explicit (though valid)
```


### Common Pitfalls {#common-pitfalls}


#### 1. Comparing nullptr with Integer {#1-dot-comparing-nullptr-with-integer}

```cpp
// int x = nullptr;    // Error: cannot convert nullptr_t to int
// if (nullptr == 0)   // Error: no match for operator==
```


#### 2. Using nullptr in Arithmetic {#2-dot-using-nullptr-in-arithmetic}

```cpp
// int* ptr = nullptr;
// ptr++;              // Undefined behavior!
// int x = ptr + 5;    // Undefined behavior!
```


#### 3. Overloading with nullptr_t {#3-dot-overloading-with-nullptr-t}

```cpp
void func(int x);
void func(std::nullptr_t);

func(0);         // Calls func(int)
func(nullptr);   // Calls func(std::nullptr_t)
```


### Related Concepts {#related-concepts}

-   **Smart Pointers**: `std::unique_ptr`, `std::shared_ptr` work seamlessly with nullptr
-   **Optional**: `std::optional` provides a safer alternative for nullable values
-   **Variant**: `std::variant` can represent multiple types including "no value"
-   **References**: Unlike pointers, references cannot be null (must always refer to an object)

<!--listend-->

```cpp
#include <memory>
#include <optional>

std::unique_ptr<int> uptr = nullptr;
std::shared_ptr<int> sptr = nullptr;
std::optional<int> opt = std::nullopt;  // Alternative to nullptr for optional values
```


### Summary {#summary}

`nullptr` is a type-safe null pointer constant introduced in C++11 that:

-   Eliminates ambiguity between null pointers and integer zero
-   Provides proper type safety through `std::nullptr_t`
-   Works correctly with function overloading and templates
-   Should be used instead of NULL in all modern C++ code
