---
title: "emacs lisp"
author: ["Hongbin Qu"]
date: 2026-11-02T00:00:00+08:00
tags: ["linux", "emacs"]
categories: ["linux coffee"]
draft: false
---

## Emacs {#emacs}

一个非常古老的编辑器，年龄甚至比 Linux 内核还要大（Emacs 诞生于 1976 年，而 Linux 内核在 1991 年才发布）。它并不是一个单纯的文本编辑器，而是一个伪装成编辑器的操作系统(¬､¬)。

而让这个“操作系统”拥有无限生命力和可扩展性的基石，正是其内置的扩展语言 —— **Emacs Lisp** (简称 Elisp)。


### lisp {#lisp}

Lisp（LISt Processor）是由图灵奖得主 John McCarthy 在 1958 年设计发明的，是除了 Fortran 之外历史第二悠久的高级编程语言。

在很多现代程序员眼中，Lisp 最显著的特征就是“那数不清的括号”（甚至常被戏称为 _Lots of Irritating Silly Parentheses_ 🤣）。但正是这种由括号包裹的结构，构成了 Lisp 最核心的设计美学：

1.  **S-表达式 (S-expression) 与前缀记法**
    所有运算和函数调用都采用统一的前缀表达式：=(operator arg1 arg2 ...)=。
    ```emacs-lisp
       ;; 基础的加法和减法，操作符在最前
       (+ 1 2 3 4)        ;; 结果为 10
       (- (* 2 5) (/ 8 2)) ;; (2 * 5) - (8 / 2) = 6
    ```

2.  **同像性 (Homoiconicity)：代码即数据，数据即代码**
    在大多数编程语言中，代码的抽象语法树（AST）是隐藏在编译器背后的；但在 Lisp 中，代码本身就是用列表（List）表示的。这赋予了 Lisp 极其强悍的元编程和宏（Macro）能力，你可以像操作普通数组或链表一样去操作、生成和改写代码。

3.  **Emacs 与 Elisp 的关系**
    Emacs 本身是一个非常精简的 C 语言核心（负责实现 Lisp 解释器、垃圾回收、文本渲染与底层 I/O），而我们在日常使用中接触到的光标移动、缓冲区管理、模式切换（Major/Minor Mode）、语法高亮以及海量插件（如 Org-mode、Magit），统统是用 Emacs Lisp 编写的。你在 Emacs 里按下的每一个按键，本质上都在触发一个 Elisp 函数调用。

<div class="tip">

刚接触 Lisp 的人往往会有“括号恐惧症”，担心括号少配对或眼花缭乱。
其实在 Emacs 中写 Elisp，大家几乎不手动去数括号，而是搭配 `smartparens~、~paredit` 或 Emacs 内置的括号高亮功能。这时候括号不再是累赘，反而成了代码结构块的保护伞，配合快捷键可以整块整块地移动、替换或包裹表达式，写起来极具掌控感。

</div>

常用基础语法速记：

```emacs-lisp
;; 1. 变量赋值与定义
(setq user-nick "Hongbin")     ;; 设置变量值
(defvar coffee-count 1 "记录今天喝了几杯咖啡") ;; 定义变量并附带文档说明

;; 2. 引用 (Quote)
;; 加单引号表示符号本身，不当作代码去求值
(setq fruit-list '(apple banana orange))

;; 3. 简单的条件判断
(if (> (length fruit-list) 2)
    (message "水果真多！")
  (message "快去买水果！"))

;; 4. 函数定义 (defun)
(defun square (n)
  "计算数字 n 的平方。"
  (* n n))

(square 5) ;; => 25
```

社区与推荐资源：

-   [emacs中国社区论坛](https://emacs-china.org)
-   [GNU Emacs Lisp Reference Manual](https://www.gnu.org/software/emacs/manual/elisp.html)


### 怎么即时学习elisp {#怎么即时学习elisp}

学习其他语言（比如 C++、Rust 甚至 Python），通常需要你搭建环境、新建文件、调用编译器或运行外部解释器。而学习 Elisp 最让人着迷的地方，就在于 **即时反馈 (Immediate Feedback)** 与 \*自省能力 (Self-documentation)\*。

Emacs 是一个活生生运行着的 Lisp 虚拟机，你不需要重启编辑器，就能在毫秒级别执行新写的代码并观察结果。


#### 1. 即时求值的三大利器 {#1-dot-即时求值的三大利器}

-   ****`*scratch*` 缓冲区****
    启动 Emacs 时默认展示的草稿缓冲区。它的主模式是 ~lisp-interaction-mode~。
    把光标放在任意一个表达式闭括号后：
    -   按 `C-x C-e` (`eval-last-sexp`)：求值当前表达式，结果会直接显示在最底部的 Minibuffer（回显区域）。
    -   按 `C-j` (`eval-print-last-sexp`)：求值并将结果直接打印在缓冲区的下一行。
-   ****`M-:` (`eval-expression`)****
    不管你在编辑什么文件，随时按下 `M-:` 快捷键，就可以在底部的 Minibuffer 中直接输入一行 Elisp 表达式求值，随用随扔，非常适合验证小逻辑。
-   ****`M-x ielm` (Interactive Emacs Lisp Mode)****
    进入一个像 Python REPL 那样的交互式命令行窗口，你可以一步一步和 Emacs 对话，查看各种变量的即时状态。
-   ****`C-M-x` (`eval-defun`)****
    当你在函数定义体内修改了实现，不需要选中整段代码，只要在函数内部按下 ~C-M-x~，这个函数就会被立刻重新加载到当前的 Emacs 会话中。


#### 2. 无敌的内建“自省”系统 {#2-dot-无敌的内建-自省-系统}

Emacs 被称为“自文档化 (Self-documenting)”的编辑器，这意味着遇到任何不懂的函数、变量或按键，你都不需要打开搜索引擎，Emacs 自身就是最权威的百科全书：

| 快捷键  | 对应命令            | 功能作用                         |
|------|-----------------|------------------------------|
| `C-h f` | `describe-function` | 查看函数的文档注释、参数签名及所在源文件 |
| `C-h v` | `describe-variable` | 查看变量的当前值、默认值及详细说明文档 |
| `C-h k` | `describe-key`      | 按下任意按键序列，查看它绑定了哪个 Elisp 函数 |
| `C-h o` | `describe-symbol`   | 模糊查找任意符号（函数、变量或 Face 样式） |
| `C-h i` | `info`              | 打开内置的 Info 手册库（包含完整的 Elisp 官方教程） |

更绝的是：在 `describe-function` 或 `describe-variable` 的帮助页面里，函数名旁边会有一个链接，点击（或者按回车）可以直接跳转到该函数在 Elisp（甚至 C 语言）中的 \*具体源码实现\*！这种完全透明的代码可见性，是其他现代编辑器无法比拟的。

<div class="tip">

在编写与调试 Elisp 时，有两个极具幸福感的小技巧：

1.  `(message "调试信息: %s" some-var)` 会把日志输出到底部和 `*Messages*` 缓冲区，是最好用的“打印调试法”。
2.  执行 ~M-x toggle-debug-on-error~，开启错误堆栈跟踪。一旦配置报错，Emacs 会自动弹出 Backtrace 窗口，清晰列出是哪一行代码、什么原因导致了错误，定位 Bug 极其迅速。

</div>


### 动手实战：从零编写一个交互式小命令 {#动手实战-从零编写一个交互式小命令}

掌握了即时求值后，我们就可以尝试为自己写一个真正可用的小功能了。

在 Elisp 中，普通函数通过 `(defun name () ...)= 定义，但如果想让用户能通过 ~M-x` 搜索执行，或者绑定快捷键给用户交互使用，就必须在函数体开头声明 `(interactive)` ：

```emacs-lisp
(defun coffee/insert-timestamp ()
  "在当前光标所在位置插入格式化的时间戳（形如：[2026-11-09 16:45]）。"
  (interactive)
  (let ((time-str (format-time-string "[%Y-%m-%d %H:%M]")))
    (insert time-str)
    (message "成功插入时间戳: %s" time-str)))

;; 为该命令绑定一个全局快捷键：C-c t
(global-set-key (kbd "C-c t") #'coffee/insert-timestamp)
```

现在，你可以用刚才学到的技巧亲自体验一番：

1.  把光标放在 `coffee/insert-timestamp` 函数末尾的括号后，按下 ~C-x C-e~；
2.  再把光标放到 `global-set-key` 末尾，按下 ~C-x C-e~；
3.  接着在缓冲区任意空白处，按下快捷键 `C-c t~（或者按 ~M-x` 输入 `coffee/insert-timestamp` 回车）。

瞬间，当前的时间字符串就被插入进来了！这就是配置和扩展 Emacs 的基本闭环：随时随地编写函数、热加载执行、绑定按键，随心所欲地打造属于自己的工作流。
