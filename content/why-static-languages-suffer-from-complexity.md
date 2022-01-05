---
title: "Why Static Languages Suffer From Complexity"
author: Hirrolot's Blog
date: Jan 3, 2022
---

<div class="introduction">

![](../media/why-static-languages-suffer-from-complexity/preview.jpg)

People in the programming language design community strive to make their languages more expressive, with a strong type system, mainly to increase ergonomics by avoiding code duplication in final software; however, the more expressive their languages become, the more abruptly duplication penetrates the language itself.

This is what I call **statics-dynamics biformity**: whenever you introduce a new linguistic abstraction to your language, it may reside either on the statics level, on the dynamics level, or on the both levels. In the first two cases, where the abstraction is located only on one particular level, you introduce _inconsistency_ to your language; in the latter case, you inevitably introduce the _feature biformity_.

For our purposes, the **statics level** is where all linguistic machinery is being performed at compile-time. Similarly, the **dynamics level** is where code is being executed at run-time. Thence the typical control operators, such as `if`/`while`/`for`/`return`, data structures, and procedures, are dynamic, whereas static type system features and syntactical macros are static. In essence, the majority of static linguistic abstractions have their correspondence in the dynamic space and vice versa:

| Dynamics | Statics |
|----------|---------|
| Variable | Generic/[Associated type] [^type-variables] |
| `if` | [Trait bound] |
| Loop/recursion | Type-level induction |
| Array | Record type/Tuple/[Heterogeneous list] |
| Tree (data structure) | Sum type/[Coproduct] |
| Pattern matching | Multiple trait implementations |

[Trait bound]: https://doc.rust-lang.org/book/ch10-02-traits.html
[Associated type]: https://doc.rust-lang.org/rust-by-example/generics/assoc_items/types.html
[Heterogeneous list]: https://beachape.com/frunk/frunk/hlist/index.html
[Coproduct]: https://beachape.com/frunk/frunk/coproduct/index.html

In the following sections, before elaborating on the problem further, let me demonstrate to you how to implement logically equivalent programs using the static and dynamic approaches. All the examples are written in Rust, but can be applied to any other general-purpose programming language with enough expressive type system. If you feel busy, feel free to jump right to the [main section] about the problem explanation.

[main section]: #the-unfortunate-consequences-of-being-static

</div>

## Record type -- Array

Consider your everyday manipulation with record types ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=945f3a2f34937369495b3733718598a5)):

<p class="code-annotation">`automobile-static.rs`</p>

```rust
struct Automobile {
    wheels: u8,
    seats: u8,
    manufacturer: String,
}

fn main() {
    let my_car = Automobile {
        wheels: 4,
        seats: 4,
        manufacturer: String::from("X"),
    };

    println!(
        "My car has {} wheels and {} seats, and it was made by {}.",
        my_car.wheels, my_car.seats, my_car.manufacturer
    );
}
```

The same can be done using arrays ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=1dc3af0998b22c401a0042b081f441e1)):

<p class="code-annotation">`automobile-dynamic.rs`</p>

```rust
use std::any::Any;

#[repr(usize)]
enum MyCar {
    Wheels,
    Seats,
    Manufacturer,
}

fn main() {
    let my_car: [Box<dyn Any>; 3] = [Box::new(4), Box::new(4), Box::new("X")];

    println!(
        "My car has {} wheels and {} seats, and it was made by {}.",
        my_car[MyCar::Wheels as usize]
            .downcast_ref::<i32>()
            .unwrap(),
        my_car[MyCar::Seats as usize].downcast_ref::<i32>().unwrap(),
        my_car[MyCar::Manufacturer as usize]
            .downcast_ref::<&'static str>()
            .unwrap()
    );
}
```

Yes, if we specify an incorrect type to `.downcast_ref`, we will get a panic. But the very **logic** of the program remains the same, only we elevate type checking to run-time.

Going further, we can encode static `Automobile` as a heterogenous list:

<p class="code-annotation">`automobile-hlist.rs`</p>

```rust
use frunk::{hlist, HList};

struct Wheels(u8);
struct Seats(u8);
struct Manufacturer(String);
type Automobile = HList![Wheels, Seats, Manufacturer];

fn main() {
    let my_car: Automobile = hlist![Wheels(4), Seats(4), Manufacturer(String::from("X"))];

    println!(
        "My car has {} wheels and {} seats, and it was made by {}.",
        my_car.get::<Wheels, _>().0,
        my_car.get::<Seats, _>().0,
        my_car.get::<Manufacturer, _>().0
    );
}
```

This version enforces exactly the same type checks as `automobile-static.rs`, but additionally provides [methods] for manipulating with `type Automobile` as with ordinary collections! E.g., we may want to reverse our automobile:

[methods]: https://docs.rs/frunk/0.4.0/frunk/hlist/struct.HCons.html#implementations

```rust
assert_eq!(
    my_car.into_reverse(),
    hlist![Manufacturer(String::from("X")), Seats(4), Wheels(4)]
);
```

(Need to augment the field types with `#[derive(Debug, PartialEq)]`.)

Or we may want to zip our car with their car:

```rust
let their_car = hlist![Wheels(6), Seats(4), Manufacturer(String::from("Y"))];

assert_eq!(
    my_car.zip(their_car),
    hlist![
        (Wheels(4), Wheels(6)),
        (Seats(4), Seats(4)),
        (Manufacturer(String::from("X")), Manufacturer(String::from("Y")))
    ]
);
```

... And so forth.

## Sum type -- Tree

One may find sum types good to represent an AST node ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=e5031b0c2888fe9ea336789ee1cdf049)):

<p class="code-annotation">`ast-static.rs`</p>

```rust
use std::ops::Deref;

enum Expr {
    Const(i32),
    Add(Box<Expr>, Box<Expr>),
    Sub(Box<Expr>, Box<Expr>),
    Mul(Box<Expr>, Box<Expr>),
    Div(Box<Expr>, Box<Expr>),
}

use Expr::*;

fn eval(expr: &Box<Expr>) -> i32 {
    match expr.deref() {
        Const(x) => *x,
        Add(lhs, rhs) => eval(&lhs) + eval(&rhs),
        Sub(lhs, rhs) => eval(&lhs) - eval(&rhs),
        Mul(lhs, rhs) => eval(&lhs) * eval(&rhs),
        Div(lhs, rhs) => eval(&lhs) / eval(&rhs),
    }
}

fn main() {
    let expr: Expr = Add(
        Const(53).into(),
        Sub(
            Div(Const(155).into(), Const(5).into()).into(),
            Const(113).into(),
        )
        .into(),
    );

    println!("{}", eval(&expr.into()));
}
```

The same can be done using tagged trees ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=6da60ed991ab6e6511c4572549047f62)):

<p class="code-annotation">`ast-dynamic.rs`</p>

```rust
use std::any::Any;

struct Tree {
    tag: i32,
    value: Box<dyn Any>,
    nodes: Vec<Box<Tree>>,
}

const AST_TAG_CONST: i32 = 0;
const AST_TAG_ADD: i32 = 1;
const AST_TAG_SUB: i32 = 2;
const AST_TAG_MUL: i32 = 3;
const AST_TAG_DIV: i32 = 4;

fn eval(expr: &Tree) -> i32 {
    let lhs = expr.nodes.get(0);
    let rhs = expr.nodes.get(1);

    match expr.tag {
        AST_TAG_CONST => *expr.value.downcast_ref::<i32>().unwrap(),
        AST_TAG_ADD => eval(&lhs.unwrap()) + eval(&rhs.unwrap()),
        AST_TAG_SUB => eval(&lhs.unwrap()) - eval(&rhs.unwrap()),
        AST_TAG_MUL => eval(&lhs.unwrap()) * eval(&rhs.unwrap()),
        AST_TAG_DIV => eval(&lhs.unwrap()) / eval(&rhs.unwrap()),
        _ => panic!("Out of range"),
    }
}

fn main() {
    let expr = /* Construction omitted... */;

    println!("{}", eval(&expr));
}
```

Similarly to how we did with `struct Automobile`, we can represent `enum Expr` as [`frunk::Coproduct`]. This is left as an exercise to a reader.

[`frunk::Coproduct`]: https://beachape.com/frunk/frunk/coproduct/enum.Coproduct.html

## Variable -- Associated type

We may want to negate a boolean value using the standard operator `!` ([playground](https://play.rust-lang.org/?version=nightly&mode=debug&edition=2021&gist=0dea07f96037bce0e82a2c93c77898b0)):

<p class="code-annotation">`negate-dynamic.rs`</p>

```rust
fn main() {
    assert_eq!(!true, false);
    assert_eq!(!false, true);
}
```

The same can be done through associated types ([playground](https://play.rust-lang.org/?version=nightly&mode=debug&edition=2021&gist=e101a1a384390a1d502aa514b21f9954)):

<p class="code-annotation">`negate-static.rs`</p>

```rust
use std::marker::PhantomData;

trait Bool {
    type Value;
}

struct True;
struct False;

impl Bool for True { type Value = True; }
impl Bool for False { type Value = False; }

struct Negate<Cond>(PhantomData<Cond>);

impl Bool for Negate<True> {
    type Value = False;
}

impl Bool for Negate<False> {
    type Value = True;
}

const ThisIsFalse: <Negate<True> as Bool>::Value = False;
const ThisIsTrue: <Negate<False> as Bool>::Value = True;
```

(We could even generalise these two implementations of `Negate` over a generic value `Cond`, but this is impossible due to a [known bug in the Rust's type system](https://github.com/rust-lang/rust/issues/20400).)

## If -- Trait bound

If-then-else is much like trait bounds ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=4ce6bda628caf7147a46df9f97864043)):

<p class="code-annotation">`kosher-dynamic.rs`</p>

```rust
const FOO: i32 = 0;
const BAR: i32 = 1;

fn is_kosher(x: i32) -> bool {
    match x {
        FOO => true,
        _ => false,
    }
}

fn main() {
    assert!(is_kosher(FOO));

    // ERROR:
    assert!(is_kosher(BAR));
}
```

This time, let us make a predicate out of a trait and define `Foo` and `Bar` as custom types ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=d8953b584f3c885c4b90051a18e79e35)):

<p class="code-annotation">`kosher-static.rs`</p>

```rust
trait Kosher {}

struct Foo;
struct Bar;

impl Kosher for Foo {}

fn accept_kosher<T: Kosher>() {}

fn main() {
    accept_kosher::<Foo>();

    // ERROR:
    accept_kosher::<Bar>();
}
```

With [negative trait bounds], we could even handle the case when `T` does _not_ implement `Kosher`, thereby expressing the "else" thing.

## Recursion -- Type-level induction

Let me show you one more example. But hold on tight this time ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=4b13a54fa1a41d928508546ef741700e))!

<p class="code-annotation">`peano-dynamic.rs`</p>

```rust
use std::ops::Deref;

#[derive(Clone, Debug, PartialEq)]
enum Nat {
    Z,
    S(Box<Nat>),
}

fn add(lhs: &Box<Nat>, rhs: &Box<Nat>) -> Nat {
    match lhs.deref() {
        Nat::Z => rhs.deref().clone(),
        Nat::S(next) => Nat::S(Box::new(add(next, rhs))),
    }
}

fn main() {
    let one = Nat::S(Nat::Z.into());
    let two = Nat::S(one.clone().into());
    let three = Nat::S(two.clone().into());

    assert_eq!(add(&one.into(), &two.into()), three);
}
```

This is the [Peano encoding] of a natural number. In the `add` function, we use recursion to compute a sum and pattern matching to find out where to stop.

[Peano encoding]: https://en.wikipedia.org/wiki/Peano_axioms

As recursion corresponds to type induction and pattern matching corresponds to multiple implementations, the same can be done at compile-time ([playground](https://play.rust-lang.org/?version=stable&mode=debug&edition=2021&gist=d4c34d5ca2d4ea81c704aeb22a443e0f)):

<p class="code-annotation">`peano-static.rs`</p>

```rust
use std::marker::PhantomData;

struct Z;
struct S<Next>(PhantomData<Next>);

trait Add<Rhs> {
    type Result;
}

impl<Rhs> Add<Rhs> for Z {
    type Result = Rhs;
}

impl<Lhs: Add<Rhs>, Rhs> Add<Rhs> for S<Lhs> {
    type Result = S<<Lhs as Add<Rhs>>::Result>;
}

type One = S<Z>;
type Two = S<One>;
type Three = S<Two>;

const THREE: <One as Add<Two>>::Result = S(PhantomData);
```

Here, `impl ... for Z` is the base case, and `impl ... for S<Lhs>` is the induction step. As in the first example, the induction works by reducing the first argument to `Z`.

You can clearly see the logical resemblance of the both examples -- because the **logic** part remains the same, no matter how you call it: be it statics or dynamics.

## The unfortunate consequences of being static

> Are you quite sure that all those bells and whistles, all those wonderful facilities of your so called powerful programming languages, belong to the solution set rather than the problem set?

<p class="quote-author">[Edsger Dijkstra]</p>

[Edsger Dijkstra]: https://en.wikipedia.org/wiki/Edsger_W._Dijkstra

Programming languages nowadays do not focus on the logic. They focus on the mechanisms inferior to logic; they call boolean negation the most simple operator that must exist from the very beginning but [negative trait bounds] are considered a debatable concept with "a lot of issues". The majority of mainstream PLs support the tree data structure in their standard libraries, but sum types [stay unimplemented for decades]. I cannot imagine a single language without the `if` operator, but only a few PLs accommodate full-fledged trait bounds, not to mention pattern matching. This is **inconsistency** -- it compels software enginners design low-quality APIs that either go dynamic and expose a very few compile-time checks or go static and try to circumvent the fundamental limitations of a host language, thereby making their usage more and more abstruse. Combining statics and dynamics in a single working solution is also complicated since you cannot invoke dynamics in a static context. In terms of [function colors], dynamics is coloured red, whereas statics is blue.

[negative trait bounds]: https://github.com/rust-lang/rfcs/issues/1834
[stay unimplemented for decades]: https://bitbashing.io/std-visit.html
[function colors]: https://journal.stuffwithstuff.com/2015/02/01/what-color-is-your-function/

In addition to this inconsistency, we have the feature **biformity**. In such languages as C++, Haskell, and Rust, this biformity amounts to the most perverse forms; you can think of any so-called "expressive" programming language as of two or more smaller languages put together: C++ the language and C++ templates/macros, Rust the language and type-level Rust + declarative macros, etc. With this approach, each time you write something at a meta-level, you cannot reuse it in the host language and vice versa, thus violating the [DRY principle] (as we shall see in a minute). Additionally, biformity increases the learning curve, hardens language evolution, and finally ends up in such a feature bloat that only the initiated can figure out what is happening in the code. Take a look at any production code in Haskell and you will immediately see those numerous GHC `#LANGUAGE` clauses, each of which signifies a separate language extension:

[DRY principle]: https://en.wikipedia.org/wiki/Don%27t_repeat_yourself

<p class="code-annotation">`feature-bloat.hs`</p>

```haskell
{-# LANGUAGE BangPatterns               #-}
{-# LANGUAGE CPP                        #-}
{-# LANGUAGE ConstraintKinds            #-}
{-# LANGUAGE DefaultSignatures          #-}
{-# LANGUAGE DeriveAnyClass             #-}
{-# LANGUAGE DeriveGeneric              #-}
{-# LANGUAGE DerivingStrategies         #-}
{-# LANGUAGE FlexibleContexts           #-}
{-# LANGUAGE FlexibleInstances          #-}
{-# LANGUAGE GADTs                      #-}
{-# LANGUAGE GeneralizedNewtypeDeriving #-}
{-# LANGUAGE NamedFieldPuns             #-}
{-# LANGUAGE OverloadedStrings          #-}
{-# LANGUAGE PolyKinds                  #-}
{-# LANGUAGE RecordWildCards            #-}
{-# LANGUAGE ScopedTypeVariables        #-}
{-# LANGUAGE TypeFamilies               #-}
{-# LANGUAGE UndecidableInstances       #-}
{-# LANGUAGE ViewPatterns               #-}
```

<p class="adapted-from">Adapted from [haskell/haskell-language-server].</p>

[haskell/haskell-language-server]: https://github.com/haskell/haskell-language-server/blob/ee0a0cc78352c961f641443eea89a26b9e1d3974/hls-plugin-api/src/Ide/Types.hs

When a host language does not provide enough static capabilities needed for convenient development, some programmers go especially insane and create whole new compile-time metalanguages and eDSLs atop of existing ones. Thus, inconsistency has the treacherous property of transforming into biformity:

<ul>
<li>

[**C++**] We have [template metaprogramming] libraries, such as [Boost/Hana] and [Boost/MPL], which copy the functionality of C++ to be used at a meta-level:


[template metaprogramming]: https://en.wikipedia.org/wiki/Template_metaprogramming
[Boost/Hana]: https://github.com/boostorg/hana
[Boost/MPL]: https://github.com/boostorg/mpl

<p class="code-annotation">`take_while.cpp`</p>

```cpp
BOOST_HANA_CONSTANT_CHECK(
    hana::take_while(hana::tuple_c<int, 0, 1, 2, 3>, hana::less.than(2_c))
    ==
    hana::tuple_c<int, 0, 1>
);
```

<p class="adapted-from">Adapted from [hana/example/take_while.cpp].</p>

[hana/example/take_while.cpp]: https://github.com/boostorg/hana/blob/998033e9dba8c82e3c9496c274a3ad1acf4a2f36/example/take_while.cpp

<p class="code-annotation">`filter.cpp`</p>

```cpp
constexpr auto is_integral =
    hana::compose(hana::trait<std::is_integral>, hana::typeid_);

static_assert(
    hana::filter(hana::make_tuple(1, 2.0, 3, 4.0), is_integral)
    == hana::make_tuple(1, 3), "");
static_assert(
    hana::filter(hana::just(3), is_integral)
    == hana::just(3), "");
BOOST_HANA_CONSTANT_CHECK(
    hana::filter(hana::just(3.0), is_integral) == hana::nothing);
```

<p class="adapted-from">Adapted from [hana/example/filter.cpp].</p>

[hana/example/filter.cpp]: https://github.com/boostorg/hana/blob/998033e9dba8c82e3c9496c274a3ad1acf4a2f36/example/filter.cpp

<p class="code-annotation">`iter_fold.cpp`</p>

```cpp
typedef vector_c<int, 5, -1, 0, 7, 2, 0, -5, 4> numbers;
typedef iter_fold<
    numbers,
    begin<numbers>::type,
    if_<less<deref<_1>, deref<_2>>, _2, _1>
>::type max_element_iter;

BOOST_MPL_ASSERT_RELATION(
    deref<max_element_iter>::type::value, ==, 7);
```

<p class="adapted-from">Adapted from [the docs of MPL].</p>

[the docs of MPL]: https://www.boost.org/doc/libs/1_78_0/libs/mpl/doc/refmanual/iter-fold.html

</li>
<li>

[**C**] My own compile-time metaprogramming framework [Metalang99] does the same by (ab)using the C preprocessor. It came to such extend that I was forced to [re-implement recursion] through the combination of a Lisp-like trampoline and [Continuation-Passing Style (CPS)]. In the end, I had a considerable number of list functions in the standard library, such as [`ML99_listMap`], [`ML99_listIntersperse`], and [`ML99_listFoldr`], which arguably makes Metalang99, as a pure data transformation language, more expressive than C itself.

[Metalang99]: https://github.com/Hirrolot/metalang99
[re-implement recursion]: https://github.com/Hirrolot/metalang99/blob/master/include/metalang99/eval/rec.h
[Continuation-Passing Style (CPS)]: https://en.wikipedia.org/wiki/Continuation-passing_style
[`ML99_listMap`]: https://metalang99.readthedocs.io/en/latest/list.html#c.ML99_listMap
[`ML99_listIntersperse`]: https://metalang99.readthedocs.io/en/latest/list.html#c.ML99_listIntersperse
[`ML99_listFoldr`]: https://metalang99.readthedocs.io/en/latest/list.html#c.ML99_listFoldr

</li>
<li>

[**Rust**] In the first example of inconsistency (`Automobile`), we used a heterogenous list from the [Frunk] library. It is of no difficulty to see that Frunk [duplicates](https://github.com/lloydmeta/frunk/blob/master/core/src/hlist.rs) some of the functionality of collections and iterators just to elevate them to type-level. It could be cool to apply `Iterator::map` or `Iterator::intersperse` to heterogenous lists, but we cannot. Even worse, if we nevertheless want to perform declarative type-level data transformations, we have to maintain the 1-to-1 correspondence between the iterator adaptors and those of type-level; each time a
 new utility is implemented for iterators, we have one utility missing in `hlist`.

[Frunk]: https://docs.rs/frunk/latest/frunk/index.html

</li>
<li>

[**Rust**] [Typenum] is yet another popular type-level library for Rust: it allows to perform integral calculations at compile-time, by encoding signed and unsigned integers as generics [^recall-generics-correspondence]. By doing this, the part of the language responsible for integers finds its counterpart in the statics, thereby introducing even more biformity [^const-generics]. We cannot just parameterise some type with `(2 + 2) * 5`, we have to write something like `<<P2 as Add<P2>>::Output as Mul<P5>>::Output`! The best thing you could do is to write a macro that does the dirty job for you, but it would only be syntax sugar -- you would anyway see hordes of compile-time errors with the aforementioned traits.

[Typenum]: https://docs.rs/typenum/latest/typenum/

</li>
</ul>

Sometimes, software engineers find their languages too primitive to express their ideas even in dynamic code. But they do not give up:

 - [**Golang**] [Kubernetes], one of the largest codebases in Golang, has its own [object-oriented type system] implemented in the [`runtime` package].

[Kubernetes]: https://github.com/kubernetes/kubernetes
[object-oriented type system]: https://medium.com/@arschles/go-experience-report-generics-in-kubernetes-25da87430301
[`runtime` package]: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime

 - [**C**] The [VLC media player] has a macro-based [plugin API] used to represent media codecs. Here is how [Opus is defined].

[VLC media player]: https://github.com/videolan/vlc
[plugin API]: https://github.com/videolan/vlc/blob/271d3552b7ad097d796bc431e946931abbe15658/include/vlc_plugin.h
[Opus is defined]: https://github.com/videolan/vlc/blob/271d3552b7ad097d796bc431e946931abbe15658/modules/codec/opus.c#L57

 - [**C**] The [QEMU machine emulator] is built upon their custom [object model](https://github.com/qemu/qemu/tree/master/include/qapi/qmp): `QObject`, `QNum`, `QNull`, `QList`, `QString`, `QDict`, `QBool`, etc.

[QEMU machine emulator]: https://github.com/qemu/qemu

Recalling the famous [Greenspun's tenth rule], such hand-made metalanguages are typically "ad-hoc, informally-specified, bug-ridden, and slow", with quite vague semantics and awful documentation. The concept of a metalinguistic abstraction simply does not work, albeit the rationale of creating declarative, small domain-specific languages sounds so cool at first sight. Have you ever tried to use a sophisticated type or macro API? If yes, then you should be perfectly acquainted with inscrutable compiler diagnostics, which can be summarised in the following screenshot [^teloxide-error-message]:

[Greenspun's tenth rule]: https://en.wikipedia.org/wiki/Greenspun%27s_tenth_rule

![](../media/whats-the-point-of-the-c-preprocessor-actually/2.jpg)

This is woefully to say, but it seems that an "expressive" PL nowadays means "Hey there, I have seriously messed up with the number of features, but that is fine!"

Finally, a word has to be said about metaprogramming in a host language. With such systems as [Template Haskell] and [Rust's procedural macros], we can manipulate an AST of a host language using the same language, which is good in terms of biformity but unpleasant in terms of inconsistency. Macros are not functions: we cannot partially apply a macro and obtain a partially applied function, since they are just different concepts. Personally, I do think that procedural macros in Rust are a giant design mistake that is comparable to `#define` macros in plain C: aside from pure syntax, the macro system simply has no idea about the language being manipulated. E.g., imagine there is an enumeration called `Either`, whose definition is as follows:

[Template Haskell]: https://wiki.haskell.org/A_practical_Template_Haskell_Tutorial
[Rust's procedural macros]: https://doc.rust-lang.org/reference/procedural-macros.html

<p class="code-annotation">`either.rs`</p>

```rust
pub enum Either<L, R> {
    Left(L),
    Right(R),
}
```

<p class="adapted-from">Adapted from [either::Either].</p>

[either::Either]: https://docs.rs/either/latest/either/enum.Either.html

Now imagine we have an arbitrary trait `Foo`, and we are willing to implement this trait for `Either<L, R>`, where `L` and `R` both implement `Foo`. It turns out that we cannot apply a derive macro to `Either` that implements this trait, even if the name is known because, in order to do this, this macro must know all the signatures of `Foo`. To make the situation even worse, `Foo` may be defined in a separate library, meaning that we cannot augment its definition with extra meta-information needed for the derivation for `Either<L, R>`. While it may seem as a rare scanario, in fact it is not; I highly encourage you to look at [tokio-util]'s [`Either`](https://docs.rs/tokio-util/latest/tokio_util/either/enum.Either.html), which is **exactly** the same enumeration but it implements Tokio-specific traits, such as `AsyncRead`, `AsyncWrite`, `AsyncSeek`, etc [^my-tokio-either]. Now imagine you have five different `Either`s in your project that came from different libraries -- that would be a true integration headache! While type introspection may be a compromise, it would nonetheless make the language even more complex than it already is.

[tokio-util]: https://docs.rs/tokio-util/latest/tokio_util/

## Is there a way out?...

Let us think a little bit about how to workaround the issue. If we make our languages fully dynamic, we will win biformity and inconsistency [^terra], but will imminently lose the pleasure of compile-time validation and will end up debugging our programs at mid-nights. The misery of dynamic type systems is widely known.

The only way to approach the problem is to make a language whose features are both static and dynamic and not to split the same feature into two parts [^dep-types]. Thus, the ideal linguistic abstraction is both static and dynamic; however, it is still a single concept and not two logically similar concepts but with different interfaces [^concept-disorder]. A perfect example is [CTFE], colloquially known as `constexpr`: same code can be executed at compile-time under a static context and at run-time under a dynamic context (e.g., when requesting a user input from `stdin`.); thus, we do not have to write different code for compile-time (statics) and run-time (dynamics), instead we use the same representation.

[CTFE]: https://en.wikipedia.org/wiki/Compile-time_function_execution

Static languages enforce compile-time checks; this is good. But they suffer from feature biformity and inconsistency -- this is bad. Dynamic languages, on the other hand, suffer from these drawbacks to a lesser extent, but they lack compile-time checks. A hypothetical solution should take the best from the both worlds.

Programming languages ought to be rethought.

[^type-variables]: In such systems as [Calculus of Constructions], polymorphic functions accept generics as ordinary function parameters of the type *, or "the type of all types".

[Calculus of Constructions]: https://en.wikipedia.org/wiki/Calculus_of_constructions

[^recall-generics-correspondence]: Do you remember our variables-generics correspondence?

[^const-generics]: Some time ago, minimal working [const generics] were stabilised. In perspective, they could replace Typenum by using the same integer representation as in ordinary code.

[const generics]: https://github.com/rust-lang/rust/issues/44580

[^teloxide-error-message]: Got it while working on [teloxide], IIRC.

[teloxide]: https://github.com/teloxide/teloxide

[^my-tokio-either]: It is even more of comedy that initially, I wrote a third-party crate called [tokio-either], which just contained that `Either` with several trait implementations. Only later, the Tokio maintainers [decided](https://github.com/tokio-rs/tokio/pull/2821) to move it to tokio-util.<br>As of Jan. 4 2022, tokio-either has 5,394 downloads total.

[tokio-either]: https://github.com/Hirrolot/tokio-either

[^terra]: Terra is a perfect example of a simple dynamic language. In the ["Simplicity" section](https://terralang.org/#simplicity), they show how features of static PLs can be implemented as libraries in dynamic languages.

[^dep-types]: To achieve so, we could use [dependent types]. For example, when requesting a field value from a hash map, as in one of the examples above, the `get` function will request a _proof_ that the field actually exists in that hash map; this way, the check will be performed at compile-time, using the same hash map that we use at run-time. Dependent types, however, are still too low-level to me, and I hardly believe that they will find their production use in their current form. With such powerful typing facilities, you typically twiddle with your type system instead of focusing on business logic.

[dependent types]: https://www.idris-lang.org/

[^concept-disorder]: Multiple personality disorder? 🤨
