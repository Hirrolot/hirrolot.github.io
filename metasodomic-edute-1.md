# Metasodomic Etude №1: SKI Combinator Calculus

This article opens a series of so-called "metasodomic" etudes, written in a functional language of mine, [Metalang99]. The goal is to have a fun contemplating rather cumbersome, sophisticated code using only the C/C++ macro system. Today we are about to implement a syntactically reduced version of the untyped lambda calculus, the [SKI combinator calculus].

[Metalang99]: https://github.com/Hirrolot/metalang99
[SKI combinator calculus]: https://en.wikipedia.org/wiki/SKI_combinator_calculus

## The Core Calculus

Believe or not, we can achieve functional programming even without lambda abstractions as a reified software building block; our approach is to define a couple of built-in combinators and then express all possible computations in terms of their composition -- this is what the SKI calculus about. Being a [Turing tarpit], the SKI calculus is enough powerful to express all computations which can be expressed by the Turing machine or the untyped lambda calculus.

[Turing tarpit]: https://en.wikipedia.org/wiki/Turing_tarpit

So the first combinator is called `K`. It merely reduces to its first argument, ignoring the second one:

```c
#define K_IMPL(x, y)    v(x)
#define K_ARITY 2
```

The second combinator is `S`, also referred to as the "substitution combinator". It applies its third argument and the application of the second and the third ones to its first argument, or more formally:

```c
#define S_IMPL(x, y, z) M_appl(M_appl(v(x), v(z)), M_appl(v(y), v(z)))
#define S_ARITY 3
```

Finally, the third combinator is the identity combinator -- it just evaluates to its single argument. In fact, it can be expressed in terms of the two aforementioned combinators like this:

```c
#define I M_appl2(v(S), v(K), v(K))
```

Thus, `I` can be perceived as a syntactic sugar.

Having our tiny calculus completed, let us look at the example evaluation of `SKSK`:

```c
// SKSK -> (by the S-rule) KK(SK) -> (by the K-rule) K.
M_eval(M_appl3(v(S), v(K), v(S), v(K)))
```

## Self-application and Recursion

## Boolean Logic
