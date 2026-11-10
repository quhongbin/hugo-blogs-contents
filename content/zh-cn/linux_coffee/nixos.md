---
title: "nixos"
author: ["Hongbin Qu"]
date: 2026-10-28T00:00:00+08:00
tags: ["linux", "nixos"]
categories: ["linux coffee"]
draft: false
---

## NixOS {#nixos}

A Declarative-configuration Operation System nix-configured


### Nix {#nix}

一个函数式编程语言，和Haskell类似，不过Haskell是一个通用领域编程语言。著名的pandoc转换程序就是用Haskell写的。有趣的是Haskell通过Nix来进行管理，而nix又集成了Haskell

<div class="tip">

在使用NixOS之前，我一直在使用Arch Linux发行版。高中时期就接触了Linux，当时觉得Linux就是一个纯命令行的 **操作系统** ，天真的我以为Linux就是一个和windows相提并论的操作系统。
我记得高三哪年，因为我们参加的是职教高考需要考写网页，和计算机基础知识（全是专升本需要考的）。学校给我们安排了一个专门的小教室，里面的电脑都有单独的主机，学校平常的机房都是云主机😅，在哪时我就经常装Linux系统，记得当时喜欢装Kali Linux,一台电脑给装好几遍，直接给引导分区干坏了🤣。
到了大学就有专门的Linux课程，当时大一的时候我还买了《鸟哥的Linux私房菜第四版》在晚自习的时候看，慢慢地我觉得老师课堂上讲的内容非常基础，一个知识点讲几遍的那种。后来慢慢地学习，对linux有了更多的认识，也了解到了很多发行版本，人生中接触到第二个Linux发行版便是Ubuntu,真是一个开箱即用的Linux发行版，是我这样的新手初入Linux世界的大门。

</div>

[NixOS官方论坛](https://discourse.nixos.org)


### 系统引导 {#系统引导}

NixOS的系统引导，就像上面说的哪样通过Nix声明式来进行配置

```Nix
{ pkgs, ... }:
n{
  boot.loader = {
    grub = {
      enable = true;
      device = "nodev"; #为什么这里是“nodev”？
      efiSupport = true;
    };
    efi.canTouchEfiVariable = true;
  };
}
```

NixOS还有一个引导配置

```Nix
{ pkgs, ... }:
{
  boot.loader = {
  systemd-boot.enable =true;
    };
  };
```

systemd-boot是systemd项目提供的通用UEFI引导管理器，systemd真是要重塑整个Linux服务管理啊😦!

我今天碰到的问题就是nixos引导器的配置，第一次systemd-boot,第二次grub引导启动，一看efi启动顺序为001 systemd-boot,002 grub引导。一看就不对劲，因为我记得很清楚第一次的configuration.nix是配置的systemd-boot，所有efi分区一直都是读的这个引导文件，即使后面使用grub配置依然不是首选启动项
