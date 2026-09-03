---
title: "Virtual"
author: ["Hongbin Qu"]
date: 2026-05-21T00:00:00+08:00
tags: ["Data Type"]
categories: ["CPP"]
draft: false
logs: "power by AI"
---

## virtual {#virtual}

> This word is supplied for c++98 standard, decorate VIRTUAL with OVERRIDE and FINAL is C++11

Declare the function is a virtual function——allow derived class overwrite the virtual function

> derived class own all members which is not private from base class


### Runtime Polymorphism {#runtime-polymorphism}

```cpp
#include <iostream>
#include <string>

// base class
class Animal {
public:
  Animal(){
          std::cout<<"Animal Constructor" <<std::endl;
  }
    // 虚函数：允许派生类重写
    virtual std::string speak() const {

        return "Animal Constructor";
    }

    // 虚析构函数（通常都应写）
    virtual ~Animal(){
      std::cout<<"Animal Destructor"<<std::endl;
    };
};

// 派生类
class Dog : public Animal {
public:
  Dog(){
          std::cout<<"Dog Constructor" <<std::endl;
  }
    std::string speak() const override {

        return "Woof!!!";
    }
    ~Dog(){
      std::cout<< "Dog Destructor"<<std::endl;
    }
};

class Cat : public Dog {
public:
  Cat(){
      std::cout<<"Cat Constructor" <<std::endl;
  }
    std::string speak() const override {

      return "Meow!!!";
    }
    ~Cat(){
      std::cout<<"Cat Destructor" <<std::endl;
    }
};


/// @brief global function accept Animal class or pointer
/// @param a derived class from Animal class
void make_sound(const Animal& a) {
    std::cout << a.speak() << std::endl;
}

int main() {

    Animal* cat = new Cat();
    delete cat;
    std::cout << "Demo con/destructor" << std::endl;
    Dog dog;
    make_sound(dog);   // 输出 "Woof!"（实际调用 Dog::speak）
    return 0;

//    make_sound(cat);   // 输出 "Meow!"（实际调用 Cat::speak）
}
```


### const and override {#const-and-override}

-   const decorate the function which declare can't modify self-object members
-   override declare the function inherit the same function signature from base class, whether thrrow except


### construction and destruction {#construction-and-destruction}

{{< figure src="../Excalidraw/virtual_keyword.png" >}}

-   if base class add virtual keyword for destructor function,
    therefore all derived class for destructor implicitly own virtual attribute


### pure virtual function {#pure-virtual-function}

-   it indicate the derived class must overwrite the pure virtual function from base class

<!--listend-->

```cpp
#include<iostream>
using namespace std;
class Shape {
public:
    virtual void draw() = 0;  // 纯虚函数
};

class Circle : public Shape {
public:
    void draw() override {
        cout << "Draw Circle" << endl;
    }
};

int main() {
    // Shape s; // 错误，抽象类不能实例化
    Shape* s = new Circle();
    s->draw(); // 输出 "Draw Circle"
    delete s;
}
```

```text
Draw Circle
```
