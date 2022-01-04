---
title: "Compile-Time Introspection of Sum Types in Pure C99"
author: Hirrolot's Blog
date: Apr 25, 2021
---

<div class="introduction">

Recently I’ve published a [blog post](https://hirrolot.medium.com/unleashing-sum-types-in-pure-c99-31544302d2ba) about [Datatype99], a library implementing sum types in pure C99 with preprocessor macros only. Today I’m going to present its new metaprogramming ability: introspection of sum types at compilation time, also with preprocessor macros only.

[Datatype99]: https://github.com/Hirrolot/datatype99

</div>

First of all, what is type introspection? For our purposes, type introspection means the retrieval and manipulation of a type representation: imagine for a second that you could gather all variants of a sum type and automatically implement some interface for it! Sounds seditiously? Let me show you how you can achieve it.

Type the following:

```c
datatype(
    MyType,
    (Foo, const char *),
    (Bar, int, int)
);
```

This code defines a sum type `MyType` with two variants: `Foo` and `Bar`. So far so good. Now our goal is to generate a function called `MyType_say_hello` which prints `"hello"` to `stdout`. This can be achieved via a _deriver macro_, a macro which accepts the representation of `MyType` and outputs something global for it, like a function definition:

```c
#define DATATYPE99_DERIVE_SayHello_IMPL(name, variants) \
    v(inline static void name##_say_hello(void) { puts("hello"); })
```

And prepend `derive(SayHello)` to our `datatype`:

```c
datatype(
    derive(SayHello),
    MyType,
    (Foo, const char *),
    (Bar, int, int)
);
```

Finally, test `MyType_say_hello`:

```c
int main(void) {
    MyType_say_hello();
}
```

This outputs `hello`, as expected. The `DATATYPE99_DERIVE_SayHello_IMPL` macro is written in [Metalang99], an underlying metaprogramming framework upon which Datatype99 works. According to Metalang99, a Metalang99-compliant macro has the `_IMPL` postfix and results in one or more language expressions; here, the only expression is `v(...)`, which evaluates to `...`. The parameters `name, variants` stand for the name of a sum type and a list of variants, respectively.

[Metalang99]: https://github.com/Hirrolot/metalang99

So how to manipulate this list of variants? The answer is: use Metalang99’s [list manipulation metafunctions]. Let’s do something more involved, for example, generating a pretty-printer:

[list manipulation metafunctions]: https://metalang99.readthedocs.io/en/latest/list.html

[[`example/print.c`](https://github.com/Hirrolot/datatype99/blob/efd7831929140377b6c3a22040b636d01c3839cc/examples/derive/print.c)]
```c
#define DATATYPE99_DERIVE_Print_IMPL(name, variants) \
    ML99_prefixedBlock( \
        v(inline static void name##_print(name self, FILE *stream)), \
        ML99_prefixedBlock( \
            v(match(self)), \
            ML99_listMapInPlace(ML99_compose(v(GEN_ARM), v(ML99_untuple)), v(variants))))

#define GEN_ARM_IMPL(tag, sig) \
    ML99_TERMS( \
        DATATYPE99_assertAttrIsPresent(v(tag##_Print_fmt)), \
        ML99_prefixedBlock( \
            DATATYPE99_of(v(tag), ML99_indexedArgs(ML99_listLen(v(sig)))), \
            ML99_invokeStmt(v(fprintf), v(stream), DATATYPE99_attrValue(v(tag##_Print_fmt)))))

#define GEN_ARM_ARITY 1
```

Looks scary, ain’t it? 😳🙊😱😱🤭

Don’t panic, I’ll explain everything to you.

The `ML99_prefixedBlock` macro evaluates to `prefix { your code... }`, the `ML99_invokeStmt` macro evaluates to `f(args...);`, `DATATYPE99_assertAttrIsPresent` and `DATATYPE99_attrValue` are a means to deal with attributes (named arguments to a deriver): as you might have already guessed, the former simply asserts the presence of an attribute, and the latter extracts its value, respectively.

The heart of our deriver is `ML99_listMapInPlace`, which walks through all variants and calls `GEN_ARM` for each one. Notice that each variant is represented as a [tuple], so in order to access its fields, one must untuple it; this is achieved by `ML99_compose` and `ML99_untuple`, a sort of functional programming!

[tuple]: https://metalang99.readthedocs.io/en/latest/tuple.html

As usual, define a sum type and test the new deriver:

```c
#define Foo_Print_fmt attr("Foo(\"%s\")", *_0)
#define Bar_Print_fmt attr("Bar(%d, %d)", *_0, *_1)

datatype(
    derive(Print),
    MyType,
    (Foo, const char *),
    (Bar, int, int)
);

// `#undef`s omitted...

int main(void) {
    MyType_print(Foo("hello world"), stdout);
    puts("");
    MyType_print(Bar(3, 5), stdout);
    puts("");
}
```

<details>
  <summary>Output</summary>

```
Foo("hello world")
Bar(3, 5)
```

</details>

Works as expected either. Moving towards more sophisticated derivers, you can generate a command menu printer:

[[`examples/command_menu.c`](https://github.com/Hirrolot/datatype99/blob/efd7831929140377b6c3a22040b636d01c3839cc/examples/derive/command_menu.c)]
```c
#define SendMessage_Menu_description        attr("Send a private message to someone")
#define SubscribeToChannel_Menu_description attr("Subscribe to channel")
#define DeleteAccount_Menu_description      attr("Delete my account")
#define DeleteAccount_Menu_note             attr("DANGEROUS")

datatype(
    derive(Menu),
    UserCommand,
    (SendMessage, MessageContent, UserId),
    (SubscribeToChannel, ChannelId),
    (DeleteAccount)
);

// `#undef`s omitted...

int main(void) {
    UserCommand_print_menu();
}
```

<details>
  <summary>Output</summary>

```
SendMessage: Send a private message to someone.
SubscribeToChannel: Subscribe to channel.
(DANGEROUS) DeleteAccount: Delete my account.
```

</details>

Or even reify the representation of variants into metadata variables:

[[`examples/metadata.c`](https://github.com/Hirrolot/datatype99/blob/efd7831929140377b6c3a22040b636d01c3839cc/examples/derive/metadata.c)]
```c
datatype(
    derive(Metadata),
    Num,
    (Char, char),
    (Int, int),
    (Double, double)
);
```

<details>
  <summary>The generated metadata</summary>

```c
static const VariantMetadata Num_variants_metadata[] = {
    {.name = "Char", .arity = 1, .size = sizeof(NumChar)},
    {.name = "Int", .arity = 1, .size = sizeof(NumInt)},
    {.name = "Double", .arity = 1, .size = sizeof(NumDouble)},
};

static const DatatypeMetadata Num_metadata = {
    .name = "Num",
    .variants = (const VariantMetadata *)&Num_variants_metadata,
    .variants_count = 3,
};
```

</details>

If you’re acquainted with Rust, you probably already know a plenty of use cases. Possible practical applications include strongly typed JSON (as in [serde-json]), strongly typed command-line arguments (as in [CLAP]), and even finite-state machine management (as in [teloxide]).

[serde-json]: https://github.com/serde-rs/json
[CLAP]: https://github.com/clap-rs/clap
[teloxide]: https://github.com/teloxide/teloxide/tree/8d8041ad6d73efd00a15943093413912704ecd14#dialogues-management

Of course, not only sum types can be introspected but also product types, more commonly known as _record types_, but Datatype99 doesn’t provide them at the time of publishing this post.

I hope you enjoyed this post and will give Datatype99 a try!

Links:

 - [Datatype99 installation instructions](https://github.com/Hirrolot/datatype99#installation)
 - [The original post](https://hirrolot.medium.com/compile-time-introspection-of-sum-types-in-pure-c99-ffa523b60385)
