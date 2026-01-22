Exercise 1 - Theory questions
-----------------------------

### Concepts

What is the difference between *concurrency* and *parallelism*?
> Concurrency is about dealing with many tasks at once (possibly interleaved), while parallelism is about doing many tasks at the same time (simultaneously on multiple processors).

What is the difference between a *race condition* and a *data race*? 
> A race condition happens when the outcome depends on the timing of events. A data race is a specific type of race condition where two threads access the same variable at the same time, and at least one is writing.

*Very* roughly - what does a *scheduler* do, and how does it do it?
> A scheduler decides which task or thread runs next. It does it by following certain rules or algorithms to share CPU time among tasks.


### Engineering

Why would we use multiple threads? What kinds of problems do threads solve?
> Multiple threads can make programs faster by doing work in parallel, or help keep programs responsive.

Some languages support "fibers" (sometimes called "green threads") or "coroutines"? What are they, and why would we rather use them over threads?
> Fibers/coroutines are lightweight units of execution managed by the language, not the OS. They use less memory and can be easier to manage than threads for certain tasks, like handling many network connections.

Does creating concurrent programs make the programmer's life easier? Harder? Maybe both?
> Both. Concurrency can make programs faster or more responsive, but it also makes code harder to write and debug.

What do you think is best - *shared variables* or *message passing*?
> Message passing seems to be safer and easier to reason about.
