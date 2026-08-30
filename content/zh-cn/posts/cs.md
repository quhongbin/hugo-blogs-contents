---
title: operating-system operating-system
---

# Process

- A basic resources unit which is assigned with memory in OS - Has isolated memory address
  > arly System

# Thread

- A basic unit is called by CPU
- rely on and within [*Process*]
- use the virtual address from the Process, but the Threads must be in same process

# coroutine

- ***

| **dimension**                      | **Process**                            | **thread**                               | **coroutine**                             |
| ---------------------------------- | -------------------------------------- | ---------------------------------------- | ----------------------------------------- |
| **caller**                         | os\'s kernel                           | os\'s kernel                             | user program                              |
| **memory space**                   | Individual(no shared)                  | share with Memory of Process             | share with Memory of Proces               |
| **switching performance overhead** | very high (us-ms)                      | middle(us)                               | low(ns)                                   |
| **parallel**                       | hundreds of concurrent cons            | Several thousand up to tens of thousands | hundreds of thousands to millions         |
| **concurrent ability**             | multicore                              | multicore                                | concurrent within singal thread           |
| **usage**                          | sandbox, dense scenario of CPU-process | Web service, dense scenario I/O          | high-concurrent gateway,async web crawler |

---

[file:memory_alignment.excalidraw](memory_alignment.excalidraw)

Memory Alignment means:

- the memory address where is stored by values is multiple of its
  value\'s bytes

# stack and heap

![](clipboard-20260607T141249.png)

# Virtualization(Three Easy Pieces)

## The Process

### A Process

- consititutes of a process
  1.  memory
  2.  address space
  3.  registers

### Process API

- Every OS must be included these ideas above **Create**: The OS is
  invoked to create a new process to run program you have indicated
  **Destroy**: A interface to halt runaway process is quite useful
  **Wait**: To wait for a process to stop running **Miscellaneous
  Control**: there are sometimes other controls that are possible. For
  example, some kind of method to suspend a process **Status**:
  Interfaces to get some status information about a Process

UNIX present two interfaces to create new process, both are fork() and
exec()

- the fork() system call

```C
#include<stdio.h>
#include<stdlib.h>
#include<unistd.h>

int main(int argc, char *argv[])
{
  printf("hello world (pid:%d)\n", (int) getpid());
  int rc = fork();
  if (rc < 0) { // fork failed; exit
    fprintf(stderr, "fork failed\n");
    exit(1);
  } else if (rc == 0) { // child (new process)
    printf("hello, I am child (pid:%d)\n", (int) getpid());
  } else { // parent goes down this path (main)
    printf("hello, I am parent of %d (pid:%d)\n",
       rc, (int) getpid());
  }
return 0;
}
```

exact copy of the calling process. To the OS, there are two same
programs is running.

- the exec() system call run different program(e.g. call _wc_ command
  in exec())
- the wait() system call

### Process Creation: A Little More Detail

![](clipboard-20260607T155546.png)

How does the OS get a program up and running?

The first thing that OS must load its code and any static data(e.g.
initialized variables) into memory, into the address space of the
process.

Once the code and data are loaded into memory, there are a few other
things the OS needs to do before running the process.

- allocated run-time stack
- allocated heap for data explicitly request dynamically-allocated
- related to I/O setup

In early OSes, they load process is done eagerly. Now modern OSes
perform the process lazily i.e., by loading pieces of code or data only
as they needed during program

To truly understand how lazy loading of pieces of code and data, we have
to understand more about the **pagin** and \*swapping\*(virtualization
of memory)

Summary,before running anything, the OS clearly must do some work to get
the important program bits from disk into memory

### Process States

- Running
- Ready
- Blocked

![](clipboard-20260607T163538.png)

### Data Structures

The OS need to track all of running program information using Data
Structures(e.g. Link Table, HashMap, Queue)

## Mechanism: Limited Direct Execution

There are a few challenges

- The first is _performance_
- The second is _control_

### Solution

1.  Restricted Operations

> Direct execution has obvious advantage of being fast; the program run
> natively on hardware CPU. But there are a few problems of privilege if
> directly execute the program e.g. access any resources which is CPU or
> MEMORY,RESIGTER and so forth

What approach OS has taken to isolate users and hardware

- user mode when running in the user mode, the process can\'t issue
  I/O request or allocate memory so forth. If doing so, the OS would
  then likely kill the process
- kernel mode(OS or kernel run in) In this mode, code that run can do
  what it likes, e.g. issue I/O requests and execute all types of
  restricted instructions, any operations needed privilege so forth

How to execute privileged operations by users mode

- Trap Instruction To execute a system call, the program must execute
  **trap** instruction. it simultaneously jumps to into the kernel and
  raises privilege level to kernel mode. Then the process can run any
  privilege operation was needed. while running end, the OS calls
  return-from-trap instruction, returns into the calling user program
  while simultaneously reduces the privilege level back to user mode

> why system calls look like procedure calls? In C Programming,the C
> standard libraries have hidden many details for system calls(e.g. like
> open() or any of system calls provided).

But we need a method to let the kernel jump into specify address(e.g.
the process address of kernel stack), otherwise, anywhere of kernel the
code jump into is a bad idea arbitrary

- Trap Table It is initialized by kernel at boot time. the table is
  likely a search table when the program execute system call.

> There are other situations for Trap Table.
>
> - **Exceptions** were triggered by CPU hardware when programs
>   crash(e.g. Page Fault, invalid instructions)
> - **Interrupt** were triggered external hardwares(e.g. mouse moving,
>   receive data by net card, timer trigger)

if only one of the situations above is triggerd, the trap handler
program will handle it

![](clipboard-20260615T173537.png) **_timeline for trap table set up_**

1.  Switcing Between Context The OS need to decide which process to
    start and stop another when it suffer from trap. If process runs
    exclusively, the OS can\'t switch to run other processes.

- a cooperative approach: wait for system calls
- a non-cooperative approach: the OS takes control
- saving and restoring context

## scheduling

1.  Workload Assumptions

a\. Each job runs for the same amount of time. b. All jobs arrive at the
same time. c. Once started, each job runs to completion. e. The run-time
of each job is known. d. All jobs only use the CPU (i.e., they perform
no I/O)

1.  Scheduling Metrics

$$T_{turnaround} = T_{complete} - T_{arrival}$$

- the $T\_{arrival} $ indicate the time process arrival in the system
- the $T_{complete}$ is time process have done on CPU

> assume all jobs arrived at same time, hence $T\_{arrival} = 0 $

ppthere are two metrics for sheduling, one of turnaround time is
**performance**, the other one is **fairness**

1.  First In, First Out(or First Come, First Served) Just like the Queue
    of datastruct, consequently the job arrived first and running on CPU
    first

- average turnaround time assuming there are three jobs---A,B and C,
  each of jobs runs 10 seconds

\begin{equation}
\label{eq:2}
\frac{10+20+30}{3} = 20
\end{equation}

But if the job in front of scheduling queue is heavyweight resource
consumer, the other jobs of relatively-short resource get queued behind
it. This cause convey effect[^1] Just like, assuming Job A runs for the
full 100 seconds before B and C

---

Job Turnaround priority
A 100s 1
B 110s 2
C 120s 3

---

- average turnaround time (aforementioned $T\_{arrival} = 0 $)

  $$\frac{100+110+120}{3} = 110 $$

---

before, we assumed the $T_{arrival}$ equal 0. Now illustrating the other
example. This time, Assume A arrives at $t = 0 $ and needs to run for
100 seconds, whereas B and C arrive at $t = 10$ and each need to run for
10 seconds. With pure SJF, it seen in

````
```C
![](images/2026-06-17_14-38-39_screenshot.png) In particular, the Job B
and C are forced to wait until Job A has completed. This looks like
non-preemptive scheduler

-   average turnaround time

      ----- ------------ ----------
      Job   Turnaround   priority
      A     100-0        1
      B     110-10       2
      C     120-10       3
      ----- ------------ ----------

    $$\frac{100+100+110}{3} = 103.33 $$

1.  Shortest Time-to-Completion First To address this concern, it can
    **preemt** job A with aforementioned acknowledge about **context
    switching** and **timer interrupts**. Just add preemption to SJF as
    the **Shortest Time-to-Completion First** or **Preemptive Shortest
    Job First** scheduler. With STCS in previous conditions, it seems in

```C
````

![](images/2026-06-17_15-18-38_screenshot.png)

---

Job Turnaround priority
A 120-0 1 » 3
B 20-10 2 » 1
C 30-10 3 » 2

---

- average turnaround time

$$\frac{(120-0)+(20-10)+(30-10)}{3} = 50 $$

1.  New Metric: Response Time For users who interact with OS usually, a
    new metrix was born: **Response Time** ---defined as the time from
    when the job arrives in a system to the first time it is scheduled

$T*{response} = T*{firstrun} - T\_{arrival} $

> The gap-time is preempted by interactive process

imagine sitting at a terminal, typing, and having to wait 10 seconds to
see a response from the system just because some other job got scheduled
in front of yours: not too pleasant.

```C
#+DOWNLOADED: screenshot @ 2026-06-22 17:16:20
```

![](images/2026-06-22_17-16-20_screenshot.png)

Thus, we are left with another problem: how can we build a scheduler
that is sensitive to response time? []{#response_time}

1.  Round Robin Scheduling To solve [this problem](#response_time), a
    new scheduling algorithm will be learned, classically referred to as
    Round Robin algorithm. The basic idea is simple: runs job for a
    **Time Slice** (sometimes called a **scheduling quantum**) and then
    switches to the next job in the run queue. It reapeatlly does so
    until the jobs are finished. For this reason, RR called also
    **time-slicing**.

```C
#+DOWNLOADED: screenshot @ 2026-06-22 17:15:52
```

![](images/2026-06-22_17-15-52_screenshot.png)

> Note that, the length of time slice must be multiple of
> timer-interrupt period

the length of time slice is critical for RR.The shorter it is , the
better the performance of RR under the response-time metrix. But if
making it long enough to amortize the cost of switching context without
making it so long that system is no longer responsive

> context switching does not arise solely from the OS actions of
> restoring and saving a few registers. Swithing to another job causes
> the state which are stored branch predictors, TLBs, in CPU caches is
> flushed

More generally, any policy(such as RR ) that is **fair**. This is an
inherent trade-off: if you are willing to be unfair, you can run shorter
jobs to completion, but at the time of response time; if you want
turnaround time is faster or better than it, the response time will be
lowered. This type of **trade-off** is common in systems

1.  Incorporating I/O If the jobs have operation of I/O, it will run
    exclusively on CPU without scheduling algorithm; just imagine two
    jobs A and B, both spend 10ms every time slice. But A is broken into
    10ms excess sub-jobs (such as operation of I/O,responsive network)

```C
#+DOWNLOADED: screenshot @ 2026-06-23 15:06:04
```

![](images/2026-06-23_15-06-04_screenshot.png)

A coommon approach is to treat echo 10-ms sub-job of A as an independent
job,With STCF, the choice is clear: choose the shorter one, in this case
A. Then, when the first sub-job of A has completed, only B is left, and
it begins running. Then a new sub-job of A is submitted, and it preempts
B and runs for 10 ms. Doing so allows for overlap, with the CPU being
used by one process while waiting for the I/O of another process to
complete; the system is thus better utilized

```C
#+DOWNLOADED: screenshot @ 2026-06-23 15:37:38
```

![](images/2026-06-23_15-37-38_screenshot.png)

> In fact, in a general-purpose OS,the OS usually knows very little
> about the length of echo job. So we need a approach without _priority_
> knowleges and like SJF/SCTF

## The Multiple-Level Feedback Queue

[^1]:
    **convey effect**: the process of CPU-bound or heavyweight
    resource to consume runs exclusively on CPU. otherwise, there are a
    pile of processes need light resource to consume behind the heavy
    process

    1.  Shortest Job First It turn out a approach solve this problem,
        then the Queue will be

    ---

    Job Turnaround priority
    B 10s 1
    C 20s 2
    A 120s 3

    ---

    - average turnaroud time(aforementioned $T\_{arrival} = 0 $)

    $$\frac{10+20+120}{3} = 50 $$

    > In the old days of batch computing, a number of non-preemptive
    > schedulers were developed. Virtually all modern scheders are
    > preemptive, stop one process from running in order to run another;
    > in particular, the scheduler can perform **context switch**,
    > stopping temporarily process is running and resuming(or starting)
    > another

## Address Spaces

### three goals for virtualize memory

- transparency
  - it means opposite: that he illusion was provided by OS
- efficiency
  - OS make the virtulization as **efficiency** as possible, both time and space(just each of processes can't run much more time or slowly; can't cost more memory)
- protection
  - isolate processes, make sure that each of processes has private address spaces

### Address Translation

OS with hardware's help turns the ugly physical mechanism into something that is useful, powerful, and easy to use abstraction

#### Base and Bounds

- it need two registers within CPU; one is called **Base**, the other is called **Bound**. base-and-bound pair allow we to place anywhere in physical address

> the technique is also referred to as **dynamic relocation**. because base-and-bound pair translate address occur in runtime, so we can move address spaces after the process has started running

physical address = virtual address + base; if physical address less than bounds, the address is legal. And CPU will raise exception, if physical address out of bounds

#### segmentation

#### MMU

Memory management unit is a part of processor to help address translation

