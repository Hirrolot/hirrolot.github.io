---
title: "Compiling Algebraic Data Types in Pure C99"
author: Hirrolot's Blog
date: May 25, 2021
---

<div class="introduction">

Lots of people have been constantly asking me how [Datatype99] works:

```c
datatype(
    BinaryTree,
    (Leaf, int),
    (Node, BinaryTree *, int, BinaryTree *)
);

int sum(const BinaryTree *tree) {
    match(*tree) {
        of(Leaf, x) return *x;
        of(Node, lhs, x, rhs) return sum(*lhs) + *x + sum(*rhs);
    }
}
```

This library implements [algebraic data types (ADT)] for pure C99. This allows you to design convenient, type-safe interfaces while sticking to plain old C with zero runtime dependencies. Today I would like to explain what the hell it does under the hood!

If you are not acquainted with the concept of algebraic data types, please read [_Unleashing Sum Types in Pure C99_](https://medium.com/@hirrolot/unleashing-sum-types-in-pure-c99-31544302d2ba) first.

To study the formal system on a concrete example, it is helpful to read this post along with the [generated output](https://godbolt.org/z/zda5Ysr6W) of [`examples/binary_tree.c`](https://github.com/Hirrolot/datatype99/blob/master/examples/binary_tree.c).

</div>

## Data layout

First of all, we want to generate a data layout for a sum type. This is achieved by a tagged union: a union of possible variants data paired with a tag (discriminant). The latter is an integer designating which particular variant is active now.

Provided that multiple parameters can be specified in a single variant definition, we must gather them all into a separate C structure:

```
typedef struct <datatype-name><variant-name> {
    <type>0 _0;
    ...
    <type>N _N;
} <datatype-name><variant-name>;
```

Metavariables:

 - `<datatype-name>` is a sum type name.
 - `<variant-name>` is a variant name.
 - `<type>I` is a type of Ith variant parameter.

So far so good. Now, in order to collect all variant data structures into a single entity, we must put them into a union:

```
typedef union <datatype-name>Variants {
    char dummy;

    <datatype-name><variant-name>0 <variant-name>0;
    ...
    <datatype-name><variant-name>N <variant-name>N;
} <datatype-name>Variants;
```

`char dummy;` is very important: consider a situation where a `datatype` definition is analogous to a plain `enum` in C. In this case, `<datatype-name>Variants` would contain only `char dummy;`. If it was not present, the union would be empty, thus violating the C standard.

In order to allow exhaustive pattern matching (more on this later), a tag is represented as `enum`:

```
typedef enum <datatype-name>Tag {
    <variant-name>0Tag, ..., <variant-name>NTag
} <datatype-name>Tag;
```

At least one variant must be specified in `datatype`, so this enumeration will never be empty.

Finalising a tagged union definition:

```
struct <datatype-name> {
    <datatype-name>Tag tag;
    <datatype-name>Variants data;
};
```

Now it can be used as any other manually written tagged union.

## Value constructors

A value constructor is an `inline static` function which constructs a variant instance. Each variant has its own value constructor.

```
inline static <datatype-name> <variant-name>(...) {
    <datatype-name> result;
    result.tag = <variant-name>Tag;
    {
        result.data.<variant-name>._0 = _0;
        ...
        result.data.<variant-name>._N = _N;
    }
    return result;
}
```

Contrary to manual initialisation (a.k.a. `{.tag = A, .data.A = { ... }}`), a value constructor is

 - More convenient to use.
 - Safer: it automatically injects a corresponding tag, protecting a user from invalid combinations of tag + data.

## Pattern matching

This is the most interesting part. Pattern matching is described by the following form:

```
match(val) {
    of(<variant-name>0, ...) <stmt>
    ...
    of(<variant-name>N, ...) <stmt>
}
```

How this can be implemented? The key idea is to use _statement prefixes_. They were first introduced by Simon Tatham in his [Metaprogramming Custom Control Structures in C].

So what is actually a statement prefix? It is any C language construction that must precede a C statement. Here are some examples: `if (...)`, `for (...)`, `while (...)`, `if (...) <stmt> else `, a label, or a combination thereof. Moreover, a statement prefix together with a statement afterwards results in a single C statement. From my experience, `for (...)` is the most flexible one so we are going to use it to implement `match` & `of`.

Let's start with `match(val)`.

[Metaprogramming Custom Control Structures in C]: https://www.chiark.greenend.org.uk/~sgtatham/mp/

### `match`

All `of` clauses must be able to access the provided value, therefore we must store it somewhere beforehand. We cannot just define it as usual: `void *datatype99_priv_matched_val = (void *)&(val);` because

 - The whole `match` construction needs to be a single C statement, according to Datatype99's grammar.
 - If two `match` constructions in the same lexical scope output `datatype99_priv_matched_val`, compilation will fail.

How can we overcome this? The following pattern will help:

```c
for (void *datatype99_priv_matched_val = ((void *)&(val));
         datatype99_priv_matched_val != (void *)0;
         datatype99_priv_matched_val = (void *)0)
```

(`datatype99_priv_matched_val` is named in such a way to respect [macro hygiene], i.e. not to conflict with user-defined variables.)

This single-iteration loop is a statement prefix. In order to avoid name clashes, it opens a new lexical scope and introduces `datatype99_priv_matched_val` into it. After it, we put `switch`:

```
switch ((val).tag)
```

After this `switch`, a user will specify a number of braced `of` clauses, thus completing the initial statement prefix `for (...)`.

`match(val)` is done. Let's move on to `of`.

[macro hygiene]: https://en.wikipedia.org/wiki/Hygienic_macro

### `of`

Logically, one `of` branch must boil down to one `case` branch, but think about `break`: it cannot be put _after_ a user-provided statement `<stmt>` following `of` because our macro just has not got access to this piece of code. Instead, we put it _before_ `<stmt>`! Thus, instead of

```
switch(...) {
    case A: <stmt> break;
    case B: <stmt> break;
    case C: <stmt> break;
}
```

we generate

```
switch(...) {
    break; case A: <stmt>
    break; case B: <stmt>
    break; case C: <stmt>
}
```

 - The first `break` does nothing: the control flow will never reach it.
 - The last `break` is absent: the control flow will nonetheless fall into the next instruction after `switch(...)`.

Now the bindings need to be generated somehow (i.e. variable names provided to `of`). All the information we have in `of` is

 - the variant name `<variant-name>`,
 - a list of bindings,
 - the matched value `datatype99_priv_matched_val`.

Each binding must

 1. Boil down to a variable definition.
 2. Be initialised to a corresponding variant argument taken from `datatype99_priv_matched_val`.

For this to happen, we need to

 1. Deduce the type of each binding.
 2. Deduce the previously erased type of `datatype99_priv_matched_val`.

### Type deduction

Hopefully, all the types are already known to us at the stage of `datatype` generation. The trick is to `typedef` them appropriately:

 - To deduce the type of the Ith binding, we can take the provided `<variant-name>` and concatenate it with `_I`, thereby obtaining a type alias to the corresponding parameter type.

 - To deduce the type of `datatype99_priv_matched_val`, we can take the provided `<variant-name>` and concatenate it with `SumT`, thereby obtaining a type alias to the outer sum type.

To make it formal, for each variant the following type definitions are generated:

```
typedef struct <datatype-name> <variant-name>SumT;

typedef <type>0 <variant-name>_0;
...
typedef <type>N <variant-name>_N;
```

Finally, inside `of`, this is how each binding is generated:

```c
for (
tag_##_##i *x = &((tag_##SumT *)datatype99_priv_matched_val)->data.tag_._##i, *ml99_priv_break = (void *)0;
ml99_priv_break != (void *)1;
ml99_priv_break = (void *)1)
```

 - `x` stands for the binding name.
 - `tag_##_##i` stands for the type alias `<variant-name>_I`. This is the type of the binding.
 - `tag_##SumT` stands for the type alias `<variant-name>SumT`. This is the type of `datatype99_priv_matched_val`.

As you might have already noticed, this is a single-iteration loop too. But it has quite different structure than the previous one: instead of using `x` to terminate the cycle, we employ the second variable `ml99_priv_break` (originated from [Metalang99], the underlying metaprogramming library). Why? Because otherwise, if `x` is leaved unused by a user, a compiler will not emit a warning, which is undesirable in our case.

Why we generate the long chains of statement prefix loops instead of plain variable definitions? Because all `case` branches in `switch` share a single lexical scope, meaning that two variables with the same name cannot be defined in two different `case` branches. In contrast to this, `for (...)` opens a new lexical scope, thus allowing for the same binding names in two distinct `of` branches.

So what we have in summary? Consider this code snippet:

```c
datatype(
    BinaryTree,
    (Leaf, int),
    (Node, BinaryTree *, int, BinaryTree *)
);

int sum(const BinaryTree *tree) {
    match(*tree) {
        of(Leaf, x) return *x;
        of(Node, lhs, x, rhs) return sum(*lhs) + *x + sum(*rhs);
    }
}
```

Here, `of(Leaf, x)` expands to

```c
for (Leaf_0 *x = &((LeafSumT *)datatype99_priv_matched_val)->data.Leaf._0,
            *ml99_priv_break = (void *)0;
     ml99_priv_break != (void *)1; ml99_priv_break = (void *)1)
```

, while `of(Node, lhs, x, rhs)` expands to

```c
for (Node_0 *lhs = &((NodeSumT *)datatype99_priv_matched_val)->data.Node._0,
            *ml99_priv_break = (void *)0;
     ml99_priv_break != (void *)1; ml99_priv_break = (void *)1)
    for (Node_1 *x = &((NodeSumT *)datatype99_priv_matched_val)->data.Node._1,
                *ml99_priv_break = (void *)0;
         ml99_priv_break != (void *)1; ml99_priv_break = (void *)1)
        for (Node_2 *rhs = &((NodeSumT *)datatype99_priv_matched_val)->data.Node._2,
                    *ml99_priv_break = (void *)0;
             ml99_priv_break != (void *)1; ml99_priv_break = (void *)1)
```

After both of them, a `return` statement follows, thus completing the statement prefixes chain.

## Exhaustive pattern matching

Datatype99 has yet another neat feature: a sane compiler will emit a warning if not all variants are handled in `match`:

```c
match(*tree) {
    of(Leaf, x) return *x;
    // of(Node, lhs, x, rhs) return sum(*lhs) + *x + sum(*rhs);
}
```

```
playground.c: In function ‘sum’:
playground.c:6:5: warning: enumeration value ‘NodeTag’ not handled in switch [-Wswitch]
    6 |     match(*tree) {
      |     ^~~~~
```

This is called exhaustive pattern matching. In fact, Datatype99 does nothing to achieve it -- provided that we desugar `match` into a proper `switch`, a C compiler can do case analysis for us! For example, the following code compiles with a warning as well:

[[godbolt](https://godbolt.org/z/MeY3bbY1a)]
```c
typedef enum {
    Foo, Bar
} MyEnum;

const MyEnum foo = Foo;

switch (foo) {
    case Foo:
        break;
}
```

```
<source>:8:5: warning: enumeration value 'Bar' not handled in switch [-Wswitch]
    8 |     switch (foo) {
      |     ^~~~~~
```

## Final words

We have not considered some utility macros like `ifLet` and `matches` since nothing special lies in them -- they are very similar to what we have already seen. For more details, feel free to investigate the [source code](https://github.com/Hirrolot/datatype99/blob/master/datatype99.h) of Datatype99. Thanks to the rich functionality of [Metalang99], it fits in just ~400 LoC single header file.

It took me almost a year to figure out how to implement Datatype99 properly. Now it is a well-tested and stable v1.x.y project, so you can use it without fear that something will be broken in the next release.

Happy hacking!

Links:

 - [Datatype99 installation instructions](https://github.com/Hirrolot/datatype99#installation)
 - [The original post](https://dev.to/hirrolot/compiling-algebraic-data-types-in-pure-c99-5225)

[Datatype99]: https://github.com/Hirrolot/datatype99
[algebraic data types (ADT)]: https://en.wikipedia.org/wiki/Algebraic_data_type
[Metalang99]: https://github.com/Hirrolot/metalang
