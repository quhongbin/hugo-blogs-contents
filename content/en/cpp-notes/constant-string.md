---
title: "constant-string"
author: ["DESKTOP-LC3M84B"]
tags: ["CPP"]
draft: false
logs: "powered by AI"
---

1.  on variable value
    -   "const" modifies an object or value,which hasn't been changing:RiComputerFill:
    -   "constexpr" decorates value of expression which is known at compile-time:RiTimer2Line:
2.  on arguments of function
    -   "const" function is regarded as constant value at runtime
    -   "constexpre" function can't indicate the arguments of function
3.  constexpr function([be used of in constant expression](<https://learncpp.com.cn/cpp-tutorial/constexpr-variables/>))
    -   it must evaluate the result at compile-time

&gt; they aren't basic types, but class types

in C++,we should avoid using C-style string that use strings around double quotation marks

string is variable-length value, unlike basic types (e.g int,unsigned int) have fixed sizes. so the data that std::string object point to is allocated in the heap, std::string object in the stack

it cast expensive performance overhead that copy a data that std::string object points to, so that we shouldn't directly use "=", nor any operation of value of copying assignment

as-if rule

-   use "const" keyword to decorate variable always
-   the constant value propagation

\`\`\`cpp
\#include &lt;iostream&gt;

int main()
{
	int x { 7 };
	std::cout &lt;&lt; x &lt;&lt; '\n'; //the x can be replaced by 7

	return 0;
}
\`\`\`

-   the constant value folding

\`\`\`cpp
\#include &lt;iostream&gt;

int main()
{
	int x { 7 };
	int y { 3 };
	std::cout &lt;&lt; x+y &lt;&lt; '\n';
	_/ x+y will be replaced by 10
	/_ first: x and y become 7 and 3
	// second: calculate the expression of x+y
	return 0;
}
\`\`\`
