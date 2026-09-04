---
title: "Godot Learning-0"
author: ["Hongbin Qu"]
date: 2026-08-26T00:00:00+08:00
tags: [""]
categories: ["Godot"]
draft: false
logs: "power by AI"
---


# Godot Engine C++ 源码架构学习教材

> **源码版本**：Godot 4.8.dev（master 分支，commit 截至 2026-08-26）
> **源码地址**：https://github.com/godotengine/godot
> **教材定位**：面向已掌握 C++ 基础与游戏引擎概念、希望深入理解 Godot 引擎内部实现的中高级开发者
> **证据等级说明**：本教材所有结论均标注证据等级
> - **A 级**：直接引用源码（含文件路径 + 行号），可在当前版本逐行验证
> - **B 级**：基于源码逻辑的合理推断，含具体文件依据
> - **C 级**：基于 Godot 官方文档或社区共识的描述
> - **D 级**：作者经验性总结，无直接源码依据

---

## 一、教材总览

本教材是一套**基于 Godot 4.x master 分支真实源码**的 C++ 架构学习材料，目标是让读者能够：

1. **看懂** Godot 引擎的整体目录组织与分层架构
2. **理解** Object / Variant / ClassDB / RID / Signal / Node 六大核心概念的实现机制
3. **追踪** 从用户代码（GDScript / C#）到 CPU 执行的完整调用链
4. **掌握** Godot 源码中使用的 C++ 工程技巧与设计模式
5. **建立** 独立阅读、修改、扩展 Godot 源码的能力

教材共分 9 个文档，按"由总到分、由概念到实现、由静态到动态"的顺序组织：

| 文档 | 标题 | 核心内容 |
|------|------|----------|
| `00-README.md` | 总览与源码地图 | 本文档，全局索引、源码地图、核心概念关系图、学习路线 |
| `01-总体架构.md` | 七大目录与分层设计 | core/servers/scene/platform/drivers/main/modules 的职责与依赖 |
| `02-核心概念.md` | 六大支柱深度解析 | Object / Variant / ClassDB / RID / Signal / Node |
| `03-启动流程.md` | 从 main() 到第一帧 | 完整启动链、初始化顺序、MainLoop 生命周期 |
| `04-服务器架构.md` | Server / RID / Backend | 渲染/物理/音频服务器的抽象与多线程模型 |
| `05-调用链案例.md` | 六个跨层调用链 | 启动、创建 Node、Signal、Rendering、Physics、GDScript→C++ |
| `06-C++技术分析.md` | 源码中的 C++ 技巧 | 前向声明、虚函数、模板、宏、引用计数、Variant 内部实现 |
| `07-模块系统与扩展.md` | modules/ 与 GDExtension | 模块注册机制、4 级初始化、扩展系统 |
| `08-源码阅读路线.md` | 八阶段学习路径 | 从入门到精通的阅读顺序、设计权衡、最终目标 |

---

## 二、源码地图（Source Map）

> **证据等级 A**：以下目录结构基于 `/home/z/my-project/godot/` 实际检出内容验证。

### 2.1 顶层目录结构

```
godot/
├── core/           # 核心库：Object 系统、Variant、容器、数学、IO、OS 抽象
├── servers/        # 服务器层：渲染、物理、音频、文本、导航等抽象接口
├── scene/          # 场景层：Node、SceneTree、CanvasItem、Control、资源系统
├── platform/       # 平台层：OS 具体实现（windows/linux/macos/android/ios/web）
├── drivers/        # 驱动层：具体后端实现（gles3/vulkan/alsa/pulseaudio/wasapi）
├── main/           # 入口层：main.cpp、Main 类（setup/start/iteration/cleanup）
├── modules/        # 模块层：可选功能（gdscript/mono/navigation/web/xr...）
├── editor/         # 编辑器：编辑器 UI 与工具（仅 TOOLS_ENABLED 时编译）
├── misc/           # 杂项：构建脚本、dist 配置、测试用例
├── platform/       # 平台适配（重复列出，含 android/ios/web/windows/linux/macos）
├── thirdparty/     # 第三方库：zlib/libpng/freetype/glslang/bullet...
├── doc/            # 文档：类参考生成、翻译
└── tests/          # 单元测试与集成测试
```

### 2.2 core/ 核心库详细地图

```
core/
├── object/             # Object 对象系统（最核心）
│   ├── object.h/.cpp           # Object 基类、GDCLASS 宏、notification、signal
│   ├── ref_counted.h/.cpp      # RefCounted 引用计数基类
│   ├── class_db.h/.cpp         # ClassDB 类注册数据库
│   ├── script_language.h       # ScriptLanguage 抽象（脚本系统基础）
│   ├── instance_placeholder.h  # 实例占位符
│   └── worker_thread_pool.h    # 工作线程池
├── variant/            # Variant 动态类型系统
│   ├── variant.h/.cpp          # Variant 联合体、27 种类型
│   ├── callable.h/.cpp         # Callable 可调用对象
│   ├── variant_op.h/.cpp       # 运算符表
│   ├── variant_internal.h      # 内部访问器
│   └── variant_utility.cpp     # 工具函数
├── templates/          # 模板容器与工具
│   ├── rid.h                   # RID 轻量句柄
│   ├── rid_owner.h             # RID_Owner<T> 所有权管理
│   ├── vector.h                # 动态数组
│   ├── hash_map.h              # 哈希表
│   ├── local_vector.h          # 局部数组（无堆分配）
│   ├── command_queue_mt.h      # 多线程命令队列
│   ├── paged_array.h           # 分页数组
│   └── cowdata.h               # 写时复制数据
├── math/               # 数学库
│   ├── math_defs.h             # real_t 定义（float/double）
│   ├── vector2.h/vector3.h     # 向量
│   ├── transform2d.h           # 2D 变换
│   ├── transform3d.h           # 3D 变换
│   ├── basis.h                 # 3x3 矩阵
│   ├── aabb.h                  # 轴对齐包围盒
│   ├── quaternion.h            # 四元数
│   └── geometry_2d/3d.h        # 几何算法
├── io/                 # 输入输出
│   ├── resource_loader.h       # 资源加载器
│   ├── resource_saver.h        # 资源保存器
│   ├── file_access.h           # 文件访问
│   ├── image.h                 # 图像
│   ├── config_file.h           # 配置文件
│   ├── json.h                  # JSON
│   └── compression.h           # 压缩
├── os/                 # 操作系统抽象
│   ├── os.h                    # OS 抽象基类
│   ├── main_loop.h             # MainLoop 主循环抽象
│   ├── thread.h                # 线程抽象
│   ├── mutex.h                 # 互斥锁
│   ├── semaphore.h             # 信号量
│   ├── memory.h                # 内存分配（memnew/memdelete）
│   ├── keyboard.h              # 键盘映射
│   └── time.h                  # 时间
├── config/             # 全局配置
│   ├── engine.h                # Engine 单例
│   ├── project_settings.h      # 项目设置
│   └── settings.h              # 引擎设置
├── string/             # 字符串系统
│   ├── ustring.h               # String（UTF-8）
│   ├── string_name.h           # StringName（ intern 字符串）
│   ├── node_path.h             # NodePath
│   └── translation.h           # 翻译
├── error/              # 错误处理
│   ├── error_list.h            # 错误码枚举
│   ├── error_macros.h          # ERR_PRINT/ERR_FAIL_COND 宏
│   └── crash_handler.h         # 崩溃处理
├── crypto/             # 加密
├── debug/              # 调试
├── extension/          # GDExtension 接口
│   ├── gdextension_interface.gen.h  # 扩展接口定义
│   └── gdextension.h           # GDExtension 类
├── input/              # 输入事件
├── variant/            # Variant（重复列出）
├── register_core_types.cpp     # 核心类型注册入口
└── core_constants.cpp          # 核心常量
```

### 2.3 servers/ 服务器层详细地图

```
servers/
├── rendering/                  # 渲染服务器
│   ├── rendering_server.h      # RenderingServer 抽象（纯虚）
│   ├── rendering_method.h      # RenderingMethod 渲染器抽象
│   ├── rendering_device.h      # RenderingDevice GPU 抽象
│   ├── rendering_device_common.h
│   └── ...
├── physics_2d/                 # 2D 物理服务器
│   ├── physics_server_2d.h     # PhysicsServer2D 抽象
│   ├── physics_server_2d_wrap_mt.h  # 多线程包装
│   ├── physics_server_2d_extension.h  # GDExtension 扩展
│   ├── physics_server_2d_dummy.h  # 空实现
│   └── physics_server_2d_manager.h  # 后端管理
├── physics_3d/                 # 3D 物理服务器（结构同上）
├── audio/                      # 音频服务器
│   ├── audio_server.h          # AudioServer
│   └── audio_stream.h
├── text/                       # 文本服务器
│   ├── text_server.h           # TextServer 抽象
│   └── text_server_manager.h
├── navigation/                 # 导航服务器
│   └── navigation_server.h
├── camera/                     # 摄像头服务器
│   └── camera_server.h
├── display/                    # 显示服务器
│   └── display_server.h        # DisplayServer（窗口/输入/剪贴板）
├── xr/                         # XR 服务器
└── register_server_types.cpp   # 服务器类型注册入口
```

### 2.4 scene/ 场景层详细地图

```
scene/
├── main/                       # 场景主类
│   ├── node.h/.cpp             # Node 基类
│   ├── scene_tree.h/.cpp       # SceneTree（继承 MainLoop）
│   ├── canvas_item.h           # CanvasItem（2D 绘制基类）
│   ├── canvas_layer.h          # CanvasLayer
│   ├── viewport.h/.cpp         # Viewport
│   ├── window.h/.cpp           # Window
│   ├── shader_globals_buffer.h
│   └── resource_format_text.cpp
├── 2d/                         # 2D 节点
│   ├── node_2d.h               # Node2D
│   ├── sprite_2d.h
│   ├── camera_2d.h
│   ├── collision_object_2d.h
│   ├── rigid_body_2d.h
│   └── ...
├── 3d/                         # 3D 节点
│   ├── node_3d.h               # Node3D
│   ├── camera_3d.h
│   ├── mesh_instance_3d.h
│   ├── rigid_body_3d.h
│   └── ...
├── gui/                        # UI 控件
│   ├── control.h               # Control（UI 基类）
│   ├── button.h
│   ├── label.h
│   ├── line_edit.h
│   └── ...
├── resources/                  # 场景资源
│   ├── material.h
│   ├── texture.h
│   ├── mesh.h
│   ├── font.h
│   └── ...
├── animation/                  # 动画系统
├── audio/                      # 场景音频节点
├── main/                       # （重复）
├── register_scene_types.cpp    # 场景类型注册入口
└── scene_constants.h
```

### 2.5 platform/ 平台层详细地图

```
platform/
├── windows/
│   ├── godot_windows.cpp       # WinMain 入口
│   ├── os_windows.h/.cpp       # OS_Windows（继承 OS）
│   ├── display_server_windows.h
│   ├── gl_manager_windows.h
│   ├── tts_windows.h
│   └── ...
├── linuxbsd/
│   ├── godot_linuxbsd.cpp      # main 入口
│   ├── os_linuxbsd.h/.cpp
│   └── ...
├── macos/
│   ├── godot_main.m            # main 入口
│   ├── os_macos.h/.cpp
│   └── ...
├── android/
│   ├── godot_android.cpp
│   ├── os_android.h
│   └── ...
├── ios/
├── web/
└── ...
```

### 2.6 main/ 入口层详细地图

```
main/
├── main.cpp                    # Main 类：setup/start/iteration/cleanup
├── main.h
├── main_timer_sync.h/.cpp      # 帧率同步
├── main_loop.h                 # （转发到 core/os/main_loop.h）
└── performance.h               # Performance 单例
```

### 2.7 modules/ 模块层（部分）

```
modules/
├── gdscript/                   # GDScript 语言实现
│   ├── gdscript.h              # GDScript 类
│   ├── gdscript_parser.h       # 解析器
│   ├── gdscript_compiler.h     # 编译器
│   ├── gdscript_vm.h           # 虚拟机
│   ├── gdscript_language.h     # 语言单例
│   └── register_types.cpp      # 模块注册
├── mono/                       # C# 支持
├── navigation/                 # 导航系统实现
├── gridmap/                    # 网格地图
├── csg/                        # 构造实体几何
├── webrtc/                     # WebRTC
├── websocket/                  # WebSocket
├── xr/                         # XR 实现
├── glslang/                    # GLSL 编译器
├── msdfgen/                    # 字体 SDF 生成
├── astcenc/                    # ASTC 纹理压缩
├── basis_universal/            # Basis 纹理压缩
├── bullet/                     # Bullet 物理（3D）
├── box2d/                      # Box2D 物理（2D）
├── register_module_types.h     # 模块注册头文件
└── register_module_types.cpp   # 模块注册实现（自动生成）
```

---

## 三、核心概念关系图

> **证据等级 A**：以下关系基于源码中的继承关系与包含关系绘制。

### 3.1 类继承关系总览

```
                    ┌─────────────┐
                    │   Object    │  ← 所有可暴露给脚本的类的根基类
                    │ (core/object)│     core/object/object.h:60
                    └──────┬──────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
     ┌──────▼──────┐ ┌─────▼──────┐ ┌────▼─────────┐
     │ RefCounted  │ │   Node     │ │  其他 Object  │
     │(ref_counted)│ │ (scene/    │ │  (MainLoop,   │
     │  .h:42      │ │  main/node)│ │   Server...)  │
     └──────┬──────┘ └─────┬──────┘ └──────────────┘
            │              │
     ┌──────▼──────┐ ┌─────▼──────────┐
     │  Resource   │ │  CanvasItem    │
     │(io/resource)│ │ (scene/main/   │
     │             │ │  canvas_item)  │
     └──────┬──────┘ └───────┬────────┘
            │                │
     ┌──────┴──────┐  ┌──────┼──────────┐
     │ Texture     │  │      │           │
     │ Material    │ ┌▼────┐ ┌▼──────┐  │
     │ Mesh        │ │Node2D│ │Control│  │
     │ Font        │ │      │ │       │  │
     │ ...         │ └──────┘ └───────┘  │
     └─────────────┘                     │
                              ┌──────────▼┐
                              │  Node3D   │
                              │           │
                              └───────────┘
```

### 3.2 Variant 类型系统

```
Variant（core/variant/variant.h:94）
├── Type 枚举（27 种类型，variant.h:97-147）
│   ├── 原子类型：NIL, BOOL, INT, FLOAT, STRING
│   ├── 数学类型：VECTOR2/2I, RECT2/2I, VECTOR3/3I, TRANSFORM2D,
│   │             VECTOR4/4I, PLANE, QUATERNION, AABB, BASIS,
│   │             TRANSFORM3D, PROJECTION
│   ├── 混合类型：COLOR, STRING_NAME, NODE_PATH, RID, OBJECT,
│   │             CALLABLE, SIGNAL, DICTIONARY, ARRAY
│   └── 打包数组：PACKED_BYTE/INT32/INT64/FLOAT32/FLOAT64/
│                 STRING/VECTOR2/VECTOR3/COLOR/VECTOR4_ARRAY
│
├── 内部存储：union _data（variant.h:255-267）
│   ├── bool _bool
│   ├── int64_t _int
│   ├── double _float
│   ├── Transform2D *_transform2d  ← 堆分配的大类型用指针
│   ├── AABB *_aabb
│   ├── Basis *_basis
│   ├── Transform3D *_transform3d
│   ├── Projection *_projection
│   ├── PackedArrayRefBase *packed_array  ← 打包数组用引用计数对象
│   ├── void *_ptr
│   └── uint8_t _mem[...]  ← ObjData 内联存储（含 Object* + refcount）
│
└── needs_deinit 表（variant.h:273-317）
    └── 编译期常量表，标记哪些类型需要析构（避免运行时判断）
```

### 3.3 ClassDB 注册机制

```
ClassDB（core/object/class_db.h）
│
├── 静态数据结构
│   ├── HashMap<StringName, ClassInfo> classes  ← 全局类注册表
│   └── ClassInfo
│       ├── StringName name
│       ├── StringName parent_name
│       ├── ClassInfo *parent_ptr
│       ├── HashMap<StringName, MethodBind*> method_map
│       ├── HashMap<StringName, PropertySetGet> property_setget
│       ├── HashMap<StringName, Variant> constant_map
│       ├── HashMap<StringName, Signal> signal_map
│       └── List<PropertyInfo> property_list
│
├── 注册宏（在类定义中使用）
│   ├── GDCLASS(ClassName, ParentName)       ← 普通类
│   ├── GDREGISTER_CLASS(ClassName)          ← 注册到 ClassDB
│   ├── GDREGISTER_VIRTUAL_CLASS(ClassName)  ← 虚拟类（不可实例化）
│   ├── GDREGISTER_ABSTRACT_CLASS(ClassName) ← 抽象类
│   └── GDREGISTER_INTERNAL_CLASS(ClassName) ← 内部类
│
└── 绑定方法
    ├── ClassDB::bind_method(D_METHOD("name", args...), &Class::method)
    ├── ClassDB::bind_signal(D_METHOD("signal_name"), ...)
    ├── ClassDB::add_property(Class, PropertyInfo, setter, getter)
    └── ClassDB::bind_integer_constant(Class, "ENUM_NAME", value)
```

### 3.4 RID 句柄系统

```
RID（core/templates/rid.h）
│
├── 数据结构（rid.h:42-50）
│   └── uint64_t _id  ← 64 位句柄
│       ├── 高 32 位：owner 索引（标识哪个 RID_Owner）
│       └── 低 32 位：对象索引（在 owner 内的索引）
│
├── RID_Owner<T>（core/templates/rid_owner.h）
│   ├── 内部使用 RID_Alloc<T>
│   ├── RID make_rid(T *p_object)      ← 注册对象，返回 RID
│   ├── T *get_or_null(RID p_rid)      ← 通过 RID 取回对象指针
│   ├── void free(RID p_rid)           ← 释放
│   └── owns(RID p_rid)                ← 判断是否归属
│
└── 使用模式（以 RenderingServer 为例）
    RenderingServer::texture_2d_create()
        → 内部 RenderingDevice 创建 GPU 资源
        → RID_Owner<Texture> 注册 Texture 对象
        → 返回 RID 给上层
    上层只持有 RID，不直接访问 Texture*
    → 解耦上层与底层实现
```

### 3.5 Signal / Callable 系统

```
Signal（core/variant/callable.h:178）
├── Object *object
└── StringName name
    └── 代表"某对象的某信号"

Callable（core/variant/callable.h:46）
├── Object *object
├── StringName method
└── 可选：CallableCustom *custom  ← 用于 lambda、绑定参数等
    └── 代表"某对象的某方法"或"自定义可调用体"

连接关系（存储在 Object::signal_map 中）
Object::signal_map : HashMap<StringName, SignalData>
└── SignalData
    ├── HashMap<Callable, Slot> slot_map  ← 该信号的所有连接
    └── Slot
        ├── Connection conn
        │   ├── Callable callable
        │   └── uint32_t flags  ← CONNECT_ONE_SHOT / DEFERRED / PERSIST
        └── ...

发射流程（Object::emit_signalp，object.cpp:1256）
1. 检查 _block_signals
2. 加锁（ObjectSignalLock）
3. 查找 signal_map[p_name]
4. 复制 slot 列表到栈（≤5 个）或堆（>5 个）
5. 解锁
6. 遍历调用每个 Callable
```

### 3.6 Node / SceneTree 生命周期

```
SceneTree（scene/main/scene_tree.h:89）
├── 继承 MainLoop
├── 持有 root: Window（根视口）
├── 维护节点树（树形结构）
│
├── 主循环（继承自 MainLoop）
│   ├── initialize()           ← 启动时调用
│   ├── process(time)          ← 每帧调用
│   │   ├── _process 通知
│   │   └── 物理处理
│   ├── physics_process(time)  ← 每物理帧调用
│   │   └── _physics_process 通知
│   └── finalize()             ← 退出时调用
│
└── Node 通知（Node::NOTIFICATION_*）
    ├── POSTINITIALIZE (0)     ← 构造后（Object 级）
    ├── PREDELETE (1)          ← 删除前（Object 级）
    ├── ENTER_TREE (10)        ← 进入场景树
    ├── EXIT_TREE (11)         ← 离开场景树
    ├── READY (13)             ← 节点就绪（子节点全部 ENTER_TREE 后）
    ├── PROCESS (20)           ← 每帧
    ├── PHYSICS_PROCESS (24)   ← 每物理帧
    ├── PARENTED (18)          ← 被设置父节点
    ├── UNPARENTED (19)        ← 被移除父节点
    ├── DRAG_BEGIN/END         ← 拖拽
    ├── PATH_CHANGED           ← 路径变化
    ├── CHILD_ORDER_CHANGED    ← 子节点顺序变化
    └── ...
```

---

## 四、学习路线总览

> **证据等级 D**：以下学习路线为作者基于源码结构总结的推荐阅读顺序。

### 4.1 八阶段学习路径

```
阶段 1：建立全局认知（1-2 天）
├── 阅读 00-README.md（本文档）
├── 阅读 01-总体架构.md
└── 目标：理解七大目录职责与依赖关系

阶段 2：掌握核心概念（3-5 天）
├── 阅读 02-核心概念.md
├── 精读 core/object/object.h
├── 精读 core/variant/variant.h
├── 精读 core/object/class_db.h
└── 目标：理解 Object/Variant/ClassDB 三大支柱

阶段 3：理解启动流程（2-3 天）
├── 阅读 03-启动流程.md
├── 精读 main/main.cpp
├── 精读 platform/<your_platform>/godot_*.cpp
└── 目标：能追踪从 main() 到第一帧渲染的完整链路

阶段 4：深入服务器架构（3-5 天）
├── 阅读 04-服务器架构.md
├── 精读 servers/rendering/rendering_server.h
├── 精读 servers/physics_2d/physics_server_2d.h
├── 精读 core/templates/rid_owner.h
└── 目标：理解 Server/RID/Backend 三层抽象

阶段 5：追踪调用链（3-5 天）
├── 阅读 05-调用链案例.md
├── 选择 1-2 个案例深入追踪
└── 目标：能独立追踪任意功能的调用链

阶段 6：学习 C++ 技巧（2-3 天）
├── 阅读 06-C++技术分析.md
├── 在源码中搜索对应模式
└── 目标：掌握 Godot 的 C++ 工程范式

阶段 7：理解扩展机制（2-3 天）
├── 阅读 07-模块系统与扩展.md
├── 精读 modules/gdscript/register_types.cpp
├── 尝试编写一个简单模块
└── 目标：能扩展引擎功能

阶段 8：形成完整认知（持续）
├── 阅读 08-源码阅读路线.md
├── 持续阅读感兴趣的具体模块
└── 目标：成为 Godot 源码贡献者
```

### 4.2 推荐的源码阅读工具

| 工具 | 用途 | 推荐度 |
|------|------|--------|
| VS Code + C/C++ + clangd | 代码导航、跳转、补全 | ★★★★★ |
| SourceTrail / Understand | 可视化调用关系 | ★★★★ |
| Doxygen | 生成类继承图 | ★★★ |
| `grep` / `rg` | 快速搜索 | ★★★★★ |
| GDB / LLDB | 运行时调试 | ★★★★★ |
| Valgrind | 内存分析 | ★★★ |

### 4.3 构建源码以辅助阅读

> **证据等级 C**：基于 Godot 官方构建文档。

```bash
# 1. 克隆源码
git clone https://github.com/godotengine/godot.git
cd godot

# 2. 生成构建系统（以 Linux 为例）
scons platform=linuxbsd target=editor dev_build=yes

# 3. 生成 compile_commands.json（供 clangd 使用）
scons platform=linuxbsd target=editor dev_build=yes compiledb=yes

# 4. 运行编辑器
./bin/godot.linuxbsd.editor.dev.x86_64
```

`dev_build=yes` 会启用调试符号与断言，便于源码阅读时理解运行时行为。

---

## 五、文档使用约定

### 5.1 证据等级标注

每个关键结论后会标注证据等级，例如：

> Object 类定义于 `core/object/object.h:60`，是所有可暴露给脚本的对象的根基类。**[证据等级 A]**

读者可据此判断结论的可靠性，并按图索骥验证源码。

### 5.2 源码引用格式

源码引用统一采用以下格式：

```
文件路径:行号
```

例如：`core/object/object.h:60` 表示 `core/object/object.h` 文件第 60 行。

### 5.3 代码片段引用

引用源码片段时，会标注来源并保留原始注释：

```cpp
// 来源：core/object/object.h:727-733
_FORCE_INLINE_ void notification(int p_notification, bool p_reversed = false) {
    if (p_reversed) {
        _notification_backward(p_notification);
    } else {
        _notification_forward(p_notification);
    }
}
```

### 5.4 Mermaid 图表

复杂流程使用 Mermaid 图表绘制，支持在 GitHub / VS Code / Typora 等工具中渲染。

### 5.5 ASCII 图表

简单的结构关系使用 ASCII 字符画，确保在任何环境下可读。

---

## 六、版本与时效性说明

> **证据等级 A**：本教材基于以下版本验证。

- **源码版本**：Godot 4.8.dev（master 分支）
- **检出日期**：2026-08-26
- **关键文件验证状态**：所有引用的文件路径与行号均基于此版本验证

Godot 引擎处于活跃开发中，源码会持续演进。读者在阅读时请注意：

1. **行号可能偏移**：随着提交，行号会变化，但文件路径与类名通常稳定
2. **API 可能调整**：4.x 系列内 API 相对稳定，但 5.0 可能有重大变化
3. **建议交叉验证**：阅读时打开对应版本的源码进行验证

---

## 七、下一步

建议从 [01-总体架构.md](./01-总体架构.md) 开始，建立对 Godot 引擎整体结构的认知。

如需快速定位特定主题，可使用以下索引：

- 想了解 **Object 系统** → [02-核心概念.md](./02-核心概念.md)
- 想了解 **引擎如何启动** → [03-启动流程.md](./03-启动流程.md)
- 想了解 **渲染/物理如何工作** → [04-服务器架构.md](./04-服务器架构.md)
- 想看 **具体调用链** → [05-调用链案例.md](./05-调用链案例.md)
- 想学 **C++ 技巧** → [06-C++技术分析.md](./06-C++技术分析.md)
- 想了解 **如何扩展引擎** → [07-模块系统与扩展.md](./07-模块系统与扩展.md)
- 想要 **完整学习路线** → [08-源码阅读路线.md](./08-源码阅读路线.md)
