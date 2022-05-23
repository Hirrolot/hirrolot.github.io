---
title: "Pretty-Printable Enumerations in Pure C"
author: Hirrolot's Blog
date: May 23, 2022

references:
  - id: pike-style
    title: "Notes on Programming in C"
    author: Rob Pike
    URL: "http://www.literateprogramming.com/pikestyle.pdf"
---

<div class="introduction">

It is a notorious fact that the `enum` keyword in C is just another way to give integers names: by defining an `enum`, you perform a work similar to `#define`-ing integer macros or declaring `const` variables. However, sometimes we may want to give our enumerations a bit more high-level semantics, as in the following scenario:

```c
#include <stdio.h>

typedef enum {
    Red,
    Green,
    Blue,
    Orange,
    White,
    Black,
    Pink,
    Yellow,
} Colour;

int main(void) {
    printf("Got '%s'!\n", Colour_print(Yellow));
}
```

We expect having `Got 'Yellow'!` printed to `stdout`. The tricky bit is the `Colour_print` function, which converts a `Colour` object to its corresponding textual representation. The reason behind this function is that we cannot just pass a `Colour` object to `printf` -- it would then print a number like `7` (if a specifier like `%d` is used).

In this post, I am to outline several alternatives that I saw or used by myself.

</div>

## The naive solution

... would be just writing this damn function:

```c
const char *Colour_print(Colour c) {
    switch (c) {
    case Red: return "Red";
    case Green: return "Green";
    case Blue: return "Blue";
    case Orange: return "Orange";
    case White: return "White";
    case Black: return "Black";
    case Pink: return "Pink";
    case Yellow: return "Yellow";
    default: return "Unknown";
    }
}
```

Damn nice.

But what if it changes?...

Look: every time you change your enumeration, this function must be updated accordingly. For example, if you change `Yellow` to `Foo`, you must update `return "Yellow";` to `return "Foo";`. To make matters worse, you may have several such functions scattered among many different places across your codebase, which is totally not fine. Besides the hassle associated with updating N different points instead of one (`O(N)` vs. `O(1)`), you are now in the risk of forgetting to update some line of code, thereby introducing a software bug that is ready to crash your working application.

## X-Macro

Some old folks may respond to this problem with a technique known as [X-Macro]:

[X-Macro]: https://en.wikipedia.org/wiki/X_Macro

```c
#define COLOURS \
    X(Red)      \
    X(Green)    \
    X(Blue)     \
    X(Orange)   \
    X(White)    \
    X(Black)    \
    X(Pink)     \
    X(Yellow)

#define X(c) c,
typedef enum { COLOURS } Colour;
#undef X

const char *Colour_print(Colour c) {
    switch (c) {
#define X(c) case c: return #c;
        COLOURS;
#undef X
    default: return "Unknown";
    }
}
```

This is a neat technique because it solves the aforementioned problem with updating N places at once: now you only need to update the `COLOURS` macro and you can expect all the other places to be updated automatically by the preprocessor. Going further, if we want to define several such enumerations without unnecessary boilerplate, we can generalise our X-Macro: instead of using the `X` convention inside `COLOURS`, we can parameterise `COLOURS` with the `f` parameter, thereby expressing a _higher-order macro_:

<p class="code-annotation">`enum-printable.h`</p>

```c
#define ENUM_PRINTABLE(name, list) \
    typedef enum { list(DEF_ENUM_VARIANT) } name; \
 \
    const char *name##_print(name val) { \
        switch (val) { \
            list(CASE_ENUM_VARIANT) \
        default: return "Unknown"; \
        } \
    }

#define DEF_ENUM_VARIANT(c)  c,
#define CASE_ENUM_VARIANT(c) case c: return #c;
```

<p class="code-annotation">`colour.c`</p>

```c
#include "enum-printable.h"

#define COLOURS(f) \
    f(Red)         \
    f(Green)       \
    f(Blue)        \
    f(Orange)      \
    f(White)       \
    f(Black)       \
    f(Pink)        \
    f(Yellow)

ENUM_PRINTABLE(Colour, COLOURS)
```

<p class="code-annotation">`apple.c`</p>

```c
#include "enum-printable.h"

#define APPLES(f) \
    f(GodenDel)   \
    f(Winesap)    \
    f(Jonathan)   \
    f(Cortland)

ENUM_PRINTABLE(Apple, APPLES)
```

Much better now! However, we can accomplish the same even without one macro per a user enumeration. Let me show you how.

## Macro iteration

```c
ENUM_PRINTABLE(Colour,
    Red, Green, Blue, Orange, White, Black, Pink, Yellow
)
```

Excellent, but how this kind of `ENUM_PRINTABLE` is implemented? If you try to implement it by yourself, you will definitely be in trouble.

However, in this post, I will not borther you with all the details. Here is the definition:

```c
#include <metalang99.h>

#define ENUM_PRINTABLE(name, ...) \
    typedef enum { __VA_ARGS__ } name; \
 \
    const char *name##_print(name val) { \
        switch (val) { \
            ML99_EVAL(ML99_variadicsForEach(ML99_reify(v(CASE_ENUM_VARIANT)), \
                                            v(__VA_ARGS__))) \
        default: return "Unknown"; \
        } \
    }

#define CASE_ENUM_VARIANT(c) case c: return #c;
```

[Metalang99] is a macro metaprogramming library for pure C99 (and C++11) that allows you to perform macro recursion and iteration. In this code snippet, we iterate on variadic arguments using [`ML99_variadicsForEach`] in order to generate a pretty-printing function for a particular enumeration. I developed Metalang99 in response to a [number of use-cases] that require macro recursion and iteration; deriving pretty-printers is only one example of such a majestic superpower.

[Metalang99]: https://github.com/Hirrolot/metalang99
[`ML99_variadicsForEach`]: https://metalang99.readthedocs.io/en/latest/variadics.html#c.ML99_variadicsForEach
[number of use-cases]: c

Why this approach is better than the previous one? It is more natural to C, easier to write, and causes less confusion at a caller site ("What is this mysterious `f`???"). E.g., you can just copy the comma-separated enumeration variants and paste them somewhere else, but with X-Macro you would need to remove the `f` invocations and intersperse commas between the variants in a proper manner.

What are the disadvantages? Now you _cannot_ derive some stuff for an `enum` in a different source file. This is what I call the _locality of definition_: all necessary stuff must be defined in the same place as our enumeration. This might or might not be a dissapointment for you, but generally speaking, locality of definition reduces flexibility of code. For example, in Rust, I cannot put my own derive macro onto some type defined in a third-party library, which sometimes causes frustration and ass pain.

> A program is a sort of publication. It’s meant to be read by the programmer, another programmer (perhaps yourself a few days, weeks or years later), and lastly a machine.

<p class="quote-author">[Rob Pike] [@pike-style]</p>

[Rob Pike]: https://en.wikipedia.org/wiki/Rob_Pike

## A word about third-party codegen

One may want to suggest using code generators like [M4] instead of Metalang99, owing to the huge amount of [macro machinery] involved in this library and/or incomprehensible compile-time errors (a false statement, we will come back to this later). Sure you can, but think about the consequences; you would then need to either:

[M4]: https://en.wikipedia.org/wiki/M4_(computer_language)
[macro machinery]: https://github.com/Hirrolot/metalang99/tree/master/include/metalang99

 1. Separate codegen files from C files, or
 2. Embed special syntax to C files and fuck up IDE support, or
 3. Write code in comments and fuck up IDE support, or
 4. Do something else and fuck IDE support.

Several projects adopted some of the aforementioned solutions (especially 3 and 4). The unfortunate consequences may not seem so severe if you invoke codegen only in a small amount of places, but declaring types, obviously, does not fall into this category of things. Therefore, we need a solution that does not require external machinery, and the only thing in pure C that can do the trick is its macro system.

Now about compilation errors: they are just fine, really. For example, if we accidentally make a mistake somewhere in the middle of `ENUM_PRINTABLE`:

<p class="code-annotation">`test.c`</p>

```c
#define ENUM_PRINTABLE(name, ...) \
    typedef enum { __VA_ARGS__ } name; \
 \
    const char *name##_print(name val) { \
        switch (val) { \
            ML99_EVAL(ML99_variadicsForEach_BLAH( \
                ML99_reify(v(CASE_ENUM_VARIANT)), v(__VA_ARGS__))) \
        default: return "Unknown"; \
        } \
    }

// More code here...
```

We would see the following compilation error:

<p class="code-annotation">`/bin/sh`</p>

```
$ gcc test.c -Imetalang99/include -ftrack-macro-expansion=0 
test.c: In function ‘Colour_print’:
test.c:20:1: error: static assertion failed: "invalid term `ML99_variadicsForEach_BLAH( (0args, ML99_reify, (0v, CASE_ENUM_VARIANT)), (0v, Red, Green, Blue, Orange, White, Black, Pink, Yellow))`"
   20 | ENUM_PRINTABLE(Colour, Red, Green, Blue, Orange, White, Black, Pink, Yellow)
      | ^~~~~~~~~~~~~~
```

From which we easily conclude that `ML99_variadicsForEach_BLAH` is not really a macro and we need to change it to the proper `ML99_variadicsForEach`, after which the compilation succeeds. In fact, Metalang99 is built with developer experience in mind: it is equipped with a built-in syntax checker and panic invocation facilities, which makes macro development a much less painful process. For a more elaborated discussion on side-effects of macros and the nature of the C preprocessor, please see my blog post [_"What's the Point of the C Preprocessor, Actually?"_](whats-the-point-of-the-c-preprocessor-actually.html)

## Final words

So here are all three solutions that I saw throughout my experience as a C programmer. You can choose an approach to your own liking. Maybe you have learnt something new. Note that the above discussion can be applied not only to pretty-printing, but generally to any piece of code that needs to generate some other code depending on a symbolic representation of enumeration's values. Cheers!

Links:

 - [Installation instructions for Metalang99](https://github.com/Hirrolot/metalang99#getting-started).
 - [Q: Why use C instead of Rust/Zig/whatever else?](https://github.com/Hirrolot/datatype99#q-why-use-c-instead-of-rustzigwhatever-else)
 - [Q: Why not third-party code generators?](https://hirrolot.github.io/posts/whats-the-point-of-the-c-preprocessor-actually.html)

For more information on Metalang99 and derived projects, see ["_Macros on Steroids, Or: How Can Pure C Benefit From Metaprogramming_"](macros-on-steroids-or-how-can-pure-c-benefit-from-metaprogramming.html).

## Appendix: Deriving pretty-printers via Datatype99

[Datatype99] is a project derived from Metalang99, which allows you to define enumerations with payloads (a.k.a. [algebraic data types (ADTs)]). It does also provide a functionality similar to Rust's derive macros. Let us leverage Datatype99 and see how to achieve pretty-printing through compile-time type introspection:

[Datatype99]: https://github.com/Hirrolot/datatype99
[algebraic data types (ADTs)]: https://en.wikipedia.org/wiki/Algebraic_data_type

```c
#include <datatype99.h>
#include <stdio.h>

#define DATATYPE99_DERIVE_Print_IMPL(name, variants) \
    ML99_call( \
        derivePrint, \
        v(name), \
        ML99_listMapInPlace(ML99_compose(v(caseEnumVariant), v(ML99_untuple)), v(variants)))

#define derivePrint_IMPL(name, ...) \
    v(const char *name##_print(name val) { \
        match(val) { \
            __VA_ARGS__ \
        } \
        return "Unknown"; \
    })

#define caseEnumVariant_IMPL(tag, _sig) ML99_prefixedBlock(DATATYPE99_of(v(tag)), v(return #tag;))
#define caseEnumVariant_ARITY           1

datatype(
    derive(Print),
    Colour,
    (Red),
    (Green),
    (Blue),
    (Orange),
    (White),
    (Black),
    (Pink),
    (Yellow)
);

int main(void) {
    printf("Got '%s'!\n", Colour_print(Yellow()));
}
```

The neat thing is that you can list other derivers together with `Print` like this: `derive(Print, Foo, Bar)`, which adds some extensibility to the code. Note that this implementation does not account variant parameters (payload); [`examples/derive/print.c`] shows how to handle them. For more information on deriver macros, see my blog post [_"Compile-Time Introspection of Sum Types in Pure C99"_](compile-time-introspection-of-sum-types-in-pure-c99.html).

[`examples/derive/print.c`]: https://github.com/Hirrolot/datatype99/blob/master/examples/derive/print.c

## References
