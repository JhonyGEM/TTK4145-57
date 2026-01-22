# Answers

<br>

## Task 2

**What happens and why:**

<br>

When two threads (or goroutines) change the same variable at the same time without any protection, the result is random and often wrong. This is because both can read and write the variable at once, so some updates are lost. This is called a race condition.

<br>

**What does GOMAXPROCS do?**

**GOMAXPROCS** sets how many operating system threads can execute Go code at the same time. If it is set to `1`, only one goroutine runs at a time, so goroutines are interleaved but not truly parallel.

<br>

## Task 3

### C

In Task 3, the shared variable `i` is protected using a mutex. This ensures that only one thread can modify `i` at a time. The threads still run at the same time, but the updates to `i` no longer interfere with each other.

Because of this, no updates are lost, and the final value is always correct.

<br>

### Go

In Go, the shared variable is handled by a separate goroutine that is responsible for updating `i`. Other goroutines do not change `i` directly, but instead send messages requesting increments or decrements.

This avoids race conditions and ensures that all updates are applied correctly, while the goroutines still run concurrently.