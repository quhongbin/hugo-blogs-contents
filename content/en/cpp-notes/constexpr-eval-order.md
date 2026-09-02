---
title: constexpr 与编译期求值顺序
date: 2026-08-31
# slug 覆盖 URL 里那段路径。中文标题会被编码成很长的 %XX 串，
# 指定 slug 后链接变成 /notes/constexpr-eval-order/，便于分享。
slug: constexpr-eval-order
categories: []
tags: [C++, constexpr]
---

`constexpr` 函数**不保证**在编译期求值。只有在需要常量表达式的场合——数组长度、模板实参、`static_assert`——编译器才强制走编译期；其它情况下它可以退化成普通的函数调用。

想强制编译期求值，用 C++20 的 `consteval`：这类函数在编译期之外调用会直接编译失败。

```cpp
constexpr int f(int x) { return x * 2; }   // 可能退化为运行期
consteval int g(int x) { return x * 2; }   // 必须在编译期求值

int a[f(21)];        // 强制编译期，OK
int b = f(21);       // 编译器可自由选择
int c = g(21);       // OK，编译期求值
```
