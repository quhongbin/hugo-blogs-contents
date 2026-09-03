---
title: "control_flow"
author: ["DESKTOP-LC3M84B"]
tags: ["Grammer"]
draft: false
logs: "power by AI"
---

## control workflow {#control-workflow}

> normally, a program execute sequentially


### Constexpr if statement(C++17) {#constexpr-if-statement--c-plus-plus-17}

```cpp
#include <iostream>
int main()
{
	constexpr double gravity{ 9.8 };

	if constexpr (gravity == 9.8) // now using constexpr if
		std::cout << "Gravity is normal.\n";
	else
		std::cout << "We are not on Earth.\n";

	return 0;
}
```


### Switch statement {#switch-statement}

```cpp
#include <iostream>
void printDigitName(int x)
{
    switch (x)
    {
    case 1:
        std::cout << "One";
        return;
    case 2:
        std::cout << "Two";
        return;
    case 3:
        std::cout << "Three";
        return;
    default:
        std::cout << "Unknown";
        return;
    }
}

int main()
{
    printDigitName(2);
    std::cout << '\n';

    return 0;
}
```

-   evaluation statement type must be a int or enum class
    -   only types of int or enum class

-
