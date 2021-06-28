---
title: "Expression-Oriented Programming in C: The FMT Macro"
author: <a href="..">Hirrolot's Blog</a>
date: May 14, 2021
---

Have you ever created useless intermediate variables like this?

```c
int fee = 500;

char response[128] = {0};
sprintf(response, "Your fee is %d USD", fee);

reply_to_user(response);
```

If you programmed in C before, the answer should be "yes". This language design drawback arises from the fact that C is a _statement-oriented_ language, meaning that if you want to perform some simple task, big chances that you need to allocate a separate variable and manipulate its pointer here and there.

This blog post explains how to overcome this unpleasantness by designing APIs which facilitate _expression-oriented_ programming. In the end, we will come up with a handy `FMT` string formatting macro that mimics [`std::format!`] of Rust:

```c
int fee = 500;
reply_to_user(FMT((char[128]){0}, "Your fee is %d USD", fee));
```

[`std::format!`]: https://doc.rust-lang.org/std/macro.format.html

## sprintf

Why `sprintf` (and its friends) requires a named variable to store a formatted string? Well, let us envision `sprintf` in an ideal world where every meaningful operation fits in a single line -- how would then the signature look? Something like this:

```c
char *sprintf(const char *restrict format, ...);
```

(Note: [`restrict`](https://en.cppreference.com/w/c/language/restrict) here means that an object referenced by `format` will be accessed only through `format`.)

But the gross truth is that it is completely unviable in C: this `sprintf` then needs to allocate memory by itself, whilst many use cases require caller-allocated memory. Let's fix it:

```c
char *sprintf(char *restrict buffer, const char *restrict format, ...);
```

Now the signature is the same as the standard library counterpart, except that we return the passed buffer instead of how many bytes have been written so far. Good. But no one uses it anyway, so we can freely get rid of it. Let's then define our superior `sprintf` wrapper.

## Superior sprintf

```c
#include <stdarg.h>
#include <stdio.h>

#define FMT(buffer, fmt, ...) fmt_str((buffer), (fmt), __VA_ARGS__)

inline static char *fmt_str(char *restrict buffer, const char *restrict fmt, ...) {
    va_list ap;
    va_start(ap, fmt);
    vsprintf(buffer, fmt, ap);
    va_end(ap);

    return buffer;
}
```

See? What we have done is just paraphrasing the old but not obsolete `sprintf`: now we can obtain the resulting string immediately from an invocation of `FMT`, without auxiliary variables:

```c
char *s = FMT((char[128]){0}, "%s %d %f", "hello world", 123, 89.209);
```

If you are curious about `(char[128]){0}`, it is called a [compound literal]: it represents an lvalue with the automatic storage duration that is needed only once. Here, the type of the compound literal is `char[128]`, a `char` array of 128 elements, all initialised to zero.

[compound literal]: https://en.cppreference.com/w/c/language/compound_literal

## The snprintf counterpart

We can modify our `FMT` and `fmt_str` to use the safe alternative to `sprintf`, `snprintf`:

```c
#include <stddef.h>
#include <stdarg.h>
#include <stdio.h>

#define FMT(buffer, fmt, ...) fmt_str(sizeof(buffer), (buffer), (fmt), __VA_ARGS__)

inline static char *fmt_str(
    size_t len, char buffer[restrict static len], const char fmt[restrict],
    ...) {
    va_list ap;
    va_start(ap, fmt);
    vsnprintf(buffer, len, fmt, ap);
    va_end(ap);

    return buffer;
}
```

Now if the passed buffer is not sufficient to hold the formatted data, we will not write past the end of it. Notice how beautiful `sizeof(buffer)` inside `FMT` works: provided it is an array type, its size will be computed correctly, contrary to just a pointer to the first element of an array.

If you are curious about the `char buffer[restrict static len]` syntax, it defines a `restrict` pointer parameter that points to the first element of some array [whose length is at least `len` bytes long](https://en.cppreference.com/w/c/language/array).

## Conclusion

What can we learn from it? At least, there is a certain kind of functions that write to a memory area, and their invocation naturally expresses the whole result of an operation. In order to be suitable for expression-oriented programming, these must return the passed memory area. `FMT` is a perfect example.

Compound literals facilitate expression-oriented programming too. They represent arbitrary values that need to be accessed only once. I encourage you to use them in all cases where a separate variable is superfluous. (Although you might still prefer to give descriptive names to program entities in certain cases.)

I hope you enjoyed the post. Looking forward to any feedback!

## Links

 - [The original post](https://dev.to/hirrolot/expression-oriented-programming-in-c-the-fmt-macro-43jo)
