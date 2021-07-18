---
title: "Extend Your Language, Don't Alter It"
author: Hirrolot's Blog
date: Jul 6, 2021

references:
  - id: mlib
    title: "Generic type-safe Container Library for C language"
    author: Patrick Pelissier
    URL: "https://github.com/P-p-H-d/mlib"

  - id: STC
    title: "Standard Template Containers for C"
    author: Tyge Løvset
    URL: "https://github.com/tylov/STC"

  - id: C-Macro-Collections
    title: "Easy to use, header only, macro generated, generic and type-safe Data Structures in C"
    author: Leonardo Vencovsky
    URL: "https://github.com/LeoVen/C-Macro-Collections"

  - id: frunk
    title: "Funktional generic type-level programming in Rust: HList, Coproduct, Generic, LabelledGeneric, Validated, Monoid and friends"
    author: Lloyd Chan
    URL: "https://github.com/lloydmeta/frunk"

  - id: typeclass-metaprogramming
    type: article-journal
    title: "An introduction to typeclass metaprogramming"
    author: Alexis King
    URL: "https://lexi-lambda.github.io/blog/2021/03/25/an-introduction-to-typeclass-metaprogramming/"

  - id: gadt-for-dummies
    type: article-journal
    title: "GADTs for dummies"
    author: Haskell Wiki
    URL: "https://wiki.haskell.org/GADTs_for_dummies"

  - id: hoas
    type: article-journal
    title: "Higher-order abstract syntax through GADTs"
    author: Wikipedia
    URL: "https://en.wikipedia.org/wiki/Generalized_algebraic_data_type#Higher-order_abstract_syntax"

  - id: type-level-rust-1
    type: article-journal
    title: "Type-level Programming in Rust"
    author: Will Crichton
    URL: "https://willcrichton.net/notes/type-level-programming/"

  - id: type-level-rust-2
    type: article-journal
    title: "Implementing a Type-safe printf in Rust"
    author: Will Crichton
    URL: "https://willcrichton.net/notes/type-safe-printf/"

  - id: rust-type-system-turing-complete
    type: article-journal
    title: "Rust's Type System is Turing-Complete"
    author: Shea Leffler
    URL: "https://sdleffler.github.io/RustTypeSystemTuringComplete/"

  - id: type-operators-rust
    title: "A macro for defining type operators in Rust."
    author: Shea Leffler
    URL: "https://github.com/sdleffler/type-operators-rs"

  - id: typenum
    title: "Compile time numbers in Rust"
    author:
    - Paho Lurie-Gregg
    - Andre Bogus
    URL: "https://github.com/paholg/typenum"

  - id: preprocessing-tokens
    title: "C99 | 6.4 Lexical elements"
    author: ISO C
    URL: "http://www.open-std.org/JTC1/SC22/WG14/www/docs/n1256.pdf"

  - id: fortraith
    title: "Compile-time compiler that compiles Forth to compile-time trait expressions"
    author: Szymon Mikulicz
    URL: "https://github.com/Ashymad/fortraith"

  - id: compiling-adt-c
    title: "Compiling Algebraic Data Types in Pure C99"
    type: article-journal
    author: Hirrolot
    URL: "https://hirrolot.github.io/posts/compiling-algebraic-data-types-in-pure-c99.html"

  - id: metaprog-structures-c-article
    title: "Metaprogramming custom control structures in C"
    type: article-journal
    author: Simon Tatham
    URL: "https://www.chiark.greenend.org.uk/~sgtatham/mp/"

  - id: idris-book
    title: "Type-Driven Development with Idris"
    type: book
    author: Edwin Brady
    URL: "https://www.manning.com/books/type-driven-development-with-idris"
---

Sometimes your programming language lacks a specific feature that will make your life easier. Perhaps language developers look upon it with a great deal of reluctance and skepticism, or are not going to implement it at all. But you need it, you need this feature right here and right now. What are you going to do then?

Generally, you have two approaches: first, you can continue living an utterly miserable and hopeless life without the feature, and second, you can implement the feature by means of some kind of _meta-abstraction_.

The first approach would introduce additional [technical debt] to your architecture. Given that sufficiently large codebases written in such languages as C already have a considerable amount of debt in them <span class="note">"Any sufficiently complicated C or Fortran program contains an ad hoc, informally-specified, bug-ridden, slow implementation of half of Common Lisp."_ -- [Greenspun's tenth rule], an amusing quote with which I agree a lot.</span>, this would make the situation even worse. On the other hand, the second approach, when you roll up your sleeves and implement a new linguistic abstraction by yourself, relieves much more potential: now you do not need to wait for a new language release with the desired functionality (which might never happen), and your codebase then going to suffer less because you are becoming able to express something important to sustain code elegance.

[technical debt]: https://en.wikipedia.org/wiki/Technical_debt
[Greenspun's tenth rule]: https://en.wikipedia.org/wiki/Greenspun%27s_tenth_rule

However, when you find yourself engineering a custom linguistic abstraction, you are in the position of a language designer. What it practically means is that the affairs can go especially tricky because your feature ought to fit well with all the other features your host language already has. In particular, the desired ability must look _natural_: it is when you feel like you continue programming in that general-purpose PL, but with the new feature added; it should not feel like an alien spacecraft fallen to Earth. In this post, I am to elaborate on the example of three PLs supporting user extension, C, Rust, and Common Lisp <span class="note">I wish I had more experience with limitless Lisp-style user extension, as in [Racket] or Common Lisp. Maybe my post would have been more profound then.</span>. I will show you how to _extend_ the language, not to _alter_ it.

[Racket]: https://racket-lang.org/

## Establishing the terminology

What do I mean by that gorgeous word, linguistic abstraction, to which I have referred two times in the introduction part? Basically, it is any language "feature": a variable, an interface, a function. And guess what a _metalinguistic abstraction_ means? Recall that the prefix "meta" simply means that the thing is used to deal with something of a similar nature: a metaprogram manipulates other programs, metadata describe other data, a [metagrammar] specifies other grammars, and so on. From this point we conclude that a metalinguistic abstraction is a linguistic abstraction used to deal with other linguistic abstractions. I am only aware of two types of them: code generation (or macros, or metaprogramming, whichever term you prefer <span class="note">There is also runtime reflection, though I am not sure whether it is a special kind of metaprogramming or not. Maybe JIT macros could outperform Java-style runtime reflection? Oh my god, that is insane...</span>) and a type system.

[metagrammar]: https://en.wikipedia.org/wiki/Metasyntax

**Why macros are "meta"?** Well, macros can do pretty much anything with the code: they can accept it, they can transform it, they can emit the code... remember how Rust allows you to execute arbitrary instructions inside [procedural macros], remember the [incremental TT muncher pattern], or how you can brutally imitate type polymorphism through the C preprocessor [@mlib; @STC; @C-Macro-Collections]. Whilst macros can manipulate other code, other code cannot manipulate macros <span class="note">Unless macros generate other macros! C/C++ macros cannot do that, while Rusty and Lispy macros can.</span> -- this is the reason why macros are "meta".

[procedural macros]: https://doc.rust-lang.org/reference/procedural-macros.html
[incremental TT muncher pattern]: https://danielkeep.github.io/tlborm/book/pat-incremental-tt-munchers.html

**Why types are "meta"?** Consider this: what you usually accomplish with metaprogramming, you can leverage to enough expressive types <span class="note">It might not be formally correct... if your metaprogramming system is Turing-complete, how would you leverage everything to a logically consistent type system, as of [Idris]? Surely, this is out of the scope of this blog post.</span>. Returning back to our poor C preprocessor, in Rust you can simply use generics instead of instantiating type-specific code by hand. Or you can go insane and play with heterogenous lists [@frunk] instead of (ab)using compile-time macro collections of [Boost/Preprocessor]. So types are capable of metaprogramming to some extent [@typeclass-metaprogramming; @hoas; @gadt-for-dummies; @type-level-rust-1; @type-level-rust-2; @rust-type-system-turing-complete; @type-operators-rust; @typenum; @fortraith; @idris-book] -- this is why they are "meta".

[Idris]: https://www.idris-lang.org/
[Boost/Preprocessor]: https://github.com/boostorg/preprocessor

The presence of either of these tools in our language makes us able to extend it with custom concepts in an embedded manner, i.e., without intervention of third-party utilities. However, today I will discuss only the syntax business -- macros.

Having the terminology established, let us dive into the pragmatics!

## Syntactical consistency

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

What is this? This is how good old C looks like with [Datatype99], a library that provides us with the full support for [algebraic data types]. Please pay your attention to the pattern matching syntax. Does it feel alright? Does it feel natural, like it has always been there? Absolutely. Now gaze upon this imaginary piece of code:

[Datatype99]: https://github.com/Hirrolot/datatype99
[algebraic data types]: https://en.wikipedia.org/wiki/Algebraic_data_type

```c
int sum(const BinaryTree *tree) {
    match(
        *tree,
        {
            of(Leaf, (x), return *x),
            of(Node, (lhs, x, rhs), return sum(*lhs) + *x + sum(*rhs)),
        });
}
```

I ask you the same question: does it feel alright? Does it feel natural, like it has always been there? Absolutely **NOT**. While it might look fine in another language, it looks utterly weird in C. But actually, what is the essential difference between these two code snippets, the difference that makes the former look properly, well-formedly, whereas the latter one look like a disformed creature? The **syntactical consistency**.

By syntactical consistency, I understand the degree by which the grammar of a particular meta-abstraction (in our case, the macros `match` & `of`) conforms to/coincides with the grammar of a host language. Recall that in C-like languages, we quite often see syntactical constructions of the form `<keyword> (...) <compound-statement>` <span class="note">A [compound statement] is a sequence of statements put into curly braces.</span>:

[compound statement]: https://en.cppreference.com/w/c/language/statements

 - `for (int i = 0; i < 10; i++) { printf("%d\n", i); }`
 - `while (i < 10) { printf("%d\n", i); i++; }`
 - `if (5 < 10) { printf("true"); }`
 - and more...

But we do not see

 - `for ((int i = 0; i < 10; i++), { printf("%d\n", i); });`
 - `while (i < 10, { printf("%d\n", i); i++; });`
 - `if (5 < 10, { printf("true"); });`
 - etc.

Got the pattern? The proper syntax of `match` _coincides_ with the syntax of the host language, C in our case, whereas the latter one does not. Another example:

```c
#define State_INTERFACE               \
    iFn(int, get, void *self);        \
    iFn(void, set, void *self, int x);

interface(State);

typedef struct {
    int x;
} Num;

int Num_State_get(void *self) {
    return ((Num *)self)->x;
}

void Num_State_set(void *self, int x) {
    ((Num *)self)->x = x;
}

impl(State, Num);
```

This time you see pure ISO C99 augmented with [Interface99], a library that provides the software interface pattern. Notice that the function definition syntax remains the same (albeit `iFn` is somewhat less common), and `impl` just deduces these definitions (`Num_State_get` & `Num_State_set`) from the context. Now consider this:

[Interface99]: https://github.com/Hirrolot/interface99

```c
impl(
    (State) for (Num),

    (int)(get)(void *self)({
        return ((Num *)self)->x;
    }),

    (void)(set)(void *self, int x)({
        ((Num *)self)->x = x;
    }),
);
```

This macro `impl` does not follow the syntax of C. This is why it looks so odd.

Both alternatives have the same semantics and the same functionality. The difference is only in the syntax part. Always try to mimic to the syntax of your host language, and you should be fine. Do not try to alter the common syntactical forms like a function/variable definition. This is what I call syntactical consistency <span class="note">[C99-Lambda] is yet another terrifying example of abusing the preprocessor. It attempts to _alter_ the native function definition syntax, and therefore it looks so odd.</span>.

[C99-Lambda]: https://github.com/Leushenko/C99-Lambda

## The bliss of Rust: Syntax-aware macros

While C/C++ macros work with preprocessing tokens [@preprocessing-tokens], Rusty macros work with [concrete syntax trees], and sometimes with language tokens. This is cool because they let you to _imitate_ the syntax of Rust: you can parse native function definitions, structures, enumerations, or pretty much anything! Consider [`tokio::select!`]:

[concrete syntax trees]: https://en.wikipedia.org/wiki/Parse_tree
[`tokio::select!`]: https://docs.rs/tokio/latest/tokio/macro.select.html

```rust
tokio::select! {
    v1 = (&mut rx1), if a.is_none() => a = Some(v1.unwrap()),
    v2 = (&mut rx2), if b.is_none() => b = Some(v2.unwrap()),
}
```

See? The `<something> => <something>` syntax is much like native Rusty pattern matching. Because of it, this syntax looks very familiar, and even if you are not yet acquainted with the macro, you can already roughly understand what is happening. Another example is derive macros of [serde-json]:

[serde-json]: https://github.com/serde-rs/json

```rust
#[derive(Serialize, Deserialize)]
struct Person {
    name: String,
    age: u8,
    phones: Vec<String>,
}
```

Here, `Serialize` & `Deserialize` are indeed macros. They parse the contents of `struct Person` and derive the corresponding traits for it. You do not need to adjust the definition of the structure because the syntax is shared, and this is awesome. If I was designing a new language of mine and there was a need in macros, I would definitely endeavour to make them work nicely with the ordinary syntax.

## The bliss of Lisp: Why S-expressions are so hot

> "Are you quite sure that all those bells and whistles, all those wonderful facilities of your so-called powerful programming languages, belong to the solution set rather than the problem set?"

<p class="quote-author">&mdash; [Edsger Dijkstra]</p>

[Edsger Dijkstra]: https://en.wikipedia.org/wiki/Edsger_W._Dijkstra

The Rust's syntax is not simple. As quite often happens in software engineering, a programming language grammar is a trade-off:

 - **Complicated syntax** allows the code to be more concise, however, it drastically reduces the amount of people able to produce reliable macros.

 - **Simple syntax** can sometimes be a bit wordy or superfluous, but enables ordinary developers to write reliable macros. With simple syntax, the chances to [mess up with syntax pecularities] are much lesser.

[mess up with syntax pecularities]: https://github.com/teloxide/teloxide-macros/pull/12

Citing [David Tolnay] (adapted from [dtolnay/reflect]):

[David Tolnay]: https://github.com/dtolnay
[dtolnay/reflect]: https://github.com/dtolnay/reflect

> "The macro author is responsible for the placement of every single angle bracket, lifetime, type parameter, trait bound, and phantom data. There is a large amount of domain knowledge involved and very few people can reliably produce robust macros with this approach."

<p class="quote-author">&mdash; [David Tolnay]</p>

As opposed to Rust, we have a solution in a completely different direction -- [s-expressions]. Instead of oversophisticating the language grammar by each subsequent release, some people decide to keep the grammar always trivial. This approach has a bunch of far-reaching implications, including simplified IDE support and language analysis in general. Metaprogramming becomes more malleable too, because you only need to handle a single homogenous structure (a list); you do not need to deal with an intimidating variety of syntactic forms your host language accomodates.

[s-expressions]: https://en.wikipedia.org/wiki/S-expression

To come back to our muttons, s-expressions facilitate syntactical consistency too. Consider this: everything is composed of s-expressions, and your macros perfectly deal with them; therefore, you can easily imitate any language construction. Everything will look the same. Even with so-called powerful Rusty macros, we cannot do this:

```rust
delegate!(self.inner) {
    pub fn is_empty(&self) -> bool;
    pub fn push(&mut self, value: T);
    pub fn pop(&mut self) -> Option<T>;
    pub fn clear(&mut self);
}
```

The only way is to write like this:

```rust
delegate! {
    to self.inner {
        pub fn is_empty(&self) -> bool;
        pub fn push(&mut self, value: T);
        pub fn pop(&mut self) -> Option<T>;
        pub fn clear(&mut self);
    }
}
```

(Adapted from [Kobzol/rust-delegate], a library for automatic method delegation in Rust.)

[Kobzol/rust-delegate]: https://github.com/Kobzol/rust-delegate

Clearly less nifty. The `match` control flow operator can do that, why your "powerful" macros cannot? Look:

```rust
let x = Some(5);
let y = 10;

// match!(x) { ...} ? Hmm...
match x {
    Some(50) => println!("Got 50"),
    Some(y) => println!("Matched, y = {:?}", y),
    _ => println!("Default case, x = {:?}", x),
}
```

(Adapted from [the chapter of TRPL about pattern matching](https://doc.rust-lang.org/book/ch18-03-pattern-syntax.html).)

Even if it could be fixed, this example still demonstrates the white holes of communication of the main syntax and user-defined macros in Rust: sometimes, due to its multifaceted grammar, it just does not allow us to express things naturally. One possible solution is an [adaptive grammar]:

[adaptive grammar]: https://en.wikipedia.org/wiki/Adaptive_grammar

> "An adaptive grammar is a formal grammar that explicitly provides mechanisms within the formalism to allow its own production rules to be manipulated."

<p class="quote-author">&mdash; Wikipedia</p>

Basically what it means is that you can specify your own syntactic forms (like `match` or `if`) right inside a source file, and a built-in parser will do the trick. [Idris] supports the feature called [syntax extensions], which is, to the best of my understanding, is pretty much like an adaptive grammar; believe or not, the `if ... then ...  else` syntax is not built into the Idris compiler, but is rather defined via the `ifThenElse` function:

[syntax extensions]: http://docs.idris-lang.org/en/latest/tutorial/syntax.html

```idris
ifThenElse : (x:Bool) -> Lazy a -> Lazy a -> a;
ifThenElse True  t e = t;
ifThenElse False t e = e;
```

Which is invoked by the following syntactic rule:

```idris
syntax if [test] then [t] else [e] = ifThenElse test t e;
```

Similar syntactical constructions can be defined in the same way. No need to wait for a couple of years till language designers decide to ship a new release, do it right here and right now. Yes, you will be right if you say that Rust is extensible, but the thing is that its extensibility is still very limited <span class="note">On the other hand, limitless extensibility tied up with a complicated syntax would make mean developers mess up with reliable macros again. What a disappointment! Returning back to s-expressions...</span>, sometimes unpleasant <span class="note">For example, the syntax of `match` is gradually evolving over time. Not so long time ago the core team has announced ["or" patterns]. With an adaptive grammar, this feature could be implemented in libraries.</span>.

["or" patterns]: https://blog.rust-lang.org/2021/06/17/Rust-1.53.0.html#or-patterns


## Extending good old C

This is all exciting and fun, but how to apply this knowledge in practice? I have an answer. Rather a long answer, full of peculiar details and techniques.

I suggest you to start with a popular article of Simon Tatham [@metaprog-structures-c-article] about metaprogramming custom control structures in C <span class="note">To the best of my knowledge, Simon Tatham was first to formulate the term _statement prefix_. He described precisely how to build custom control flow operators via regular `#define` macros, and make them look natural.</span>. If you are only interested in a working solution, consider [Metalang99] <span class="note">[Metalang99] is an advanced metaprogramming system for C99. It is implemented as a purely functional programming language, with partial applications, recursion, algebraic data types, cons-lists, and all the stuff.</span> with its [statement chaining macros]. Seeing how pattern matching works in [Datatype99] [@compiling-adt-c] can also give you some insight.

[Metalang99]: https://github.com/Hirrolot/metalang99
[statement chaining macros]: https://metalang99.readthedocs.io/en/latest/gen.html

## Final words

Some languages are more malleable to user extension than the others. Some employ adaptive grammars ([Idris]), some employ syntax-aware macros (Rust), some employ Lisp-style s-expressions. Surely, there are a lot of alternatives in the design space, each has its own benefits and downsides.

The intent of this blog post was to advocate the principle of syntactical consistency.

I encourage you to mimic the syntax of a host language when writing macros, to make your code look more eye-pleasing, less like a malevolent beast.

I encourage you to **extend, not to alter**.

## Links

 - Reddit discussions:
   - [r/programming](https://www.reddit.com/r/programming/comments/oexpgf/extend_your_language_dont_alter_it/)
   - [r/rust](https://www.reddit.com/r/rust/comments/oexogi/extend_your_language_dont_alter_it/)
   - [r/ProgrammingLanguages](https://www.reddit.com/r/ProgrammingLanguages/comments/oexnl6/extend_your_language_dont_alter_it/)

## References
