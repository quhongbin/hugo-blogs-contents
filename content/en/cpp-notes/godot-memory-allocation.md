---
title: "Godot Memory Allocation"
author: ["Hongbin Qu"]
date: 2025-05-27T00:00:00+08:00
tags: ["Memory Management"]
categories: ["CPP"]
draft: false
logs: "power by AI"
logs_descripe: "The design about Memory Management of Godot game engine"
---

## Godot Memory Allocation {#godot-memory-allocation}

Godot provides its own memory allocation system in `memory.h` and `memory.cpp`. It wraps `malloc` and `calloc` with additional debugging features.


### Memory::alloc_static {#memory-alloc-static}

This template function is the core allocator. It accepts a boolean template parameter `p_ensure_zero`.

```cpp
template <bool p_ensure_zero>
void *Memory::alloc_static(size_t p_bytes, bool p_pad_align)
```


### Key Features {#key-features}

1.  ****Zero-fill option**** — If `p_ensure_zero` is `true`, it uses `calloc` (zero-initialized). Otherwise `malloc`.
2.  ****Padding / Alignment**** — When `p_pad_align` is `true` (or always in debug mode), it adds `DATA_OFFSET` extra bytes before the actual data. This stores the allocation size.
3.  ****Size tracking**** — The real size is written at `SIZE_OFFSET` bytes before the returned pointer. This helps detect buffer overflows.
4.  ****Profiling**** — `GodotProfileAlloc` tracks memory usage statistics.
5.  ****Debug stats**** — In debug mode, `_current_mem_usage` and `_max_mem_usage` track total memory.


### Memory Layout {#memory-layout}

\`\`\`

| SIZE_OFFSET | DATA_OFFSET |                |
|-------------|-------------|----------------|
| [real size] | [padding]   | [user data...] |

malloc ptr     return ptr     user starts here
\`\`\`


### Related Notes {#related-notes}

-   [C++ void\* Pointers]({{< relref "cpp-void-ptr.md" >}}) — alloc_static returns void\*
-   [C++ Templates]({{< relref "cpp-templates.md" >}}) — the bool template parameter enables compile-time dispatch
-   [Copy Constructors]({{< relref "cpp-copy-constructors.md" >}}) — Godot objects rely on proper copy semantics
-   [C++ Smart Pointers &amp; Ref]({{< relref "cpp-smart-pointers.md" >}}) — Ref works with Godot's memory system
-   [Reference Counting]({{< relref "cpp-ref-counting.md" >}}) — many Godot objects are ref-counted
-   [C++ Memory Management]({{< relref "cpp-memory-management.md" >}}) — parent topic
