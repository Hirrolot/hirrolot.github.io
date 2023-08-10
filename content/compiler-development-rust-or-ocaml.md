<div class="introduction">

The question of which language suits best for compiler development is a frequent one amongst language enthusiasts (e.g., see discussions [here](https://www.reddit.com/r/ProgrammingLanguages/comments/k3zgjy/which_language_to_write_a_compiler_in/), [here](https://www.reddit.com/r/ProgrammingLanguages/comments/13eztdp/good_languages_for_writing_compilers_in/), and [here](https://www.reddit.com/r/ProgrammingLanguages/comments/15gz8rb/how_good_is_go_for_writing_a_compiler/)). Sadly, most of the commenters either 1) just answer with their language of choice without any explanation, or 2) provide a vague explanation without any specific examples to prove their point of view. Both types of answers serve little to no purpose for the person asking the question. In this essay, I will try to provide a more detailed perspective on the topic by comparing two languages: Rust and OCaml.

</div>

## CPS conversion

Before presenting my actual argument, I will show two analogous implementations of [CPS] conversion [^cps-compiler] for a very simple language, without making any conclusions. The general approach is borrowed from ["Compiling with Continuations"] by Andrew W. Appel. No worries if you are not familiar with the idea; the only thing to focus your attention on is _how_ the idea is implemented in both Rust and OCaml.

[CPS]: https://en.wikipedia.org/wiki/Continuation-passing_style
["Compiling with Continuations"]: https://www.amazon.com/Compiling-Continuations-Andrew-W-Appel/dp/052103311X

Here is CPS conversion written in Rust [^update-rust-arenas]:

```{.rust .numberLines}
use std::cell::RefCell;
use std::ops::Deref;
use std::rc::Rc;

// A variable identifier of the lambda language `Term`.
type Var = String;

// The lambda language; direct style.
type Term = Rc<TermTree>;

enum TermTree {
    Var(Var),
    Fix(Vec<(Var, Vec<Var>, Term)>, Term),
    Appl(Term, Vec<Term>),
    Record(Vec<Term>),
    Select(Term, u32),
}

use TermTree::*;

#[derive(Clone)]
enum CpsVar {
    // Taken from the lambda term during CPS conversion.
    CLamVar(Var),
    // Generated uniquely during CPS conversion.
    CGenVar(u32),
}

use CpsVar::*;

// The resulting CPS term.
enum CpsTerm {
    CFix(Vec<(CpsVar, Vec<CpsVar>, CpsTerm)>, Box<CpsTerm>),
    CAppl(CpsVar, Vec<CpsVar>),
    CRecord(Vec<CpsVar>, Binder),
    CSelect(CpsVar, u32, Binder),
    CHalt(CpsVar),
}

use CpsTerm::*;

// Binds a unique `CpsVar` within `CpsTerm`.
type Binder = (CpsVar, Box<CpsTerm>);

// Generates a unique CPS variable given the current `i`.
fn gensym(i: RefCell<u32>) -> CpsVar {
    let x = CGenVar(i.clone().into_inner());
    i.replace_with(|&mut i| i + 1);
    x
}

// Converts `Term` to `CpsTerm`, applying `finish` to the resulting
// CPS variable.
fn convert(gen: RefCell<u32>, finish: impl FnOnce(CpsVar) -> CpsTerm, term: Term) -> CpsTerm {
    match term.deref() {
        Var(x) => finish(CLamVar(x.to_string())),
        Fix(defs, m) => CFix(
            defs.iter()
                .map(|def| convert_def(gen.clone(), def.clone()))
                .collect(),
            Box::new(convert(gen, finish, m.clone())),
        ),
        Appl(f, args) => {
            let ret_k = gensym(gen.clone());
            let ret_k_x = gensym(gen.clone());
            CFix(
                vec![(ret_k.clone(), vec![ret_k_x.clone()], finish(ret_k_x))],
                Box::new(convert(
                    gen.clone(),
                    |f_cps| {
                        convert_list(
                            gen,
                            |args_cps| {
                                CAppl(f_cps, args_cps.into_iter().chain(vec![ret_k]).collect())
                            },
                            args,
                        )
                    },
                    f.clone(),
                )),
            )
        }
        Record(fields) => convert_list(
            gen.clone(),
            |fields_cps| {
                let x = gensym(gen);
                CRecord(fields_cps, (x.clone(), Box::new(finish(x))))
            },
            fields,
        ),
        Select(m, i) => convert(
            gen.clone(),
            |m_cps| {
                let x = gensym(gen);
                CSelect(m_cps, *i, (x.clone(), Box::new(finish(x))))
            },
            m.clone(),
        ),
    }
}

// Converts `Vec<Term>` to `Vec<CpsVar>` and applies `finish` to it.
fn convert_list(
    gen: RefCell<u32>,
    finish: impl FnOnce(Vec<CpsVar>) -> CpsTerm,
    terms: &[Term],
) -> CpsTerm {
    fn go(
        gen: RefCell<u32>,
        finish: impl FnOnce(Vec<CpsVar>) -> CpsTerm,
        mut acc: Vec<CpsVar>,
        terms: &[Term],
    ) -> CpsTerm {
        match terms.split_first() {
            None => finish(acc),
            Some((x, xs)) => convert(
                gen.clone(),
                |x_cps| {
                    acc.push(x_cps);
                    go(gen, finish, acc, xs)
                },
                x.clone(),
            ),
        }
    }
    let acc = Vec::with_capacity(terms.len());
    go(gen, finish, acc, terms)
}

// Converts a single function definition to its CPS form.
fn convert_def(
    gen: RefCell<u32>,
    (f, params, m): (Var, Vec<Var>, Term),
) -> (CpsVar, Vec<CpsVar>, CpsTerm) {
    let k = gensym(gen.clone());
    (
        CLamVar(f),
        params
            .into_iter()
            .map(CLamVar)
            .chain(std::iter::once(k.clone()))
            .collect(),
        convert(gen, |m_cps| CAppl(k, vec![m_cps]), m),
    )
}
```

The code is 145 lines long, including comments and blank lines.

The same algorithm in idiomatic OCaml [^cps-ocaml]:

```{.ocaml .numberLines}
(* A variable identifier of the lambda language [term]. *)
type var = string

(* The lambda language; direct style. *)
type term =
  | Var of var
  | Fix of (var * var list * term) list * term
  | Appl of term * term list
  | Record of term list
  | Select of term * int

type cps_var =
  (* Taken from the lambda term during CPS conversion. *)
  | CLamVar of var
  (* Generated uniquely during CPS conversion. *)
  | CGenVar of int

(* The resulting CPS term. *)
type cps_term =
  | CFix of (cps_var * cps_var list * cps_term) list * cps_term
  | CAppl of cps_var * cps_var list
  | CRecord of cps_var list * binder
  | CSelect of cps_var * int * binder
  | CHalt of cps_var

(* Binds a unique [cps_var] within [cps_term]. *)
and binder = cps_var * cps_term

(* Generates a unique CPS variable given the current [i]. *)
let gensym i =
  let x = CGenVar !i in
  i := !i + 1;
  x

(* Converts [term] to [cps_term], applying [finish] to the resulting
   CPS variable. *)
let rec convert gen finish = function
  | Var x -> finish (CLamVar x)
  | Fix (defs, m) -> CFix (List.map (convert_def gen) defs, convert gen finish m)
  | Appl (f, args) ->
      let ret_k = gensym gen in
      let ret_k_x = gensym gen in
      CFix
        ( [ (ret_k, [ ret_k_x ], finish ret_k_x) ],
          f
          |> convert gen (fun f_cps ->
                 args
                 |> convert_list gen (fun args_cps ->
                        CAppl (f_cps, args_cps @ [ ret_k ]))) )
  | Record fields ->
      fields
      |> convert_list gen (fun fields_cps ->
             let x = gensym gen in
             CRecord (fields_cps, (x, finish x)))
  | Select (m, i) ->
      m
      |> convert gen (fun m_cps ->
             let x = gensym gen in
             CSelect (m_cps, i, (x, finish x)))

(* Converts [term list] to [cps_var list] and applies [finish] to it. *)
and convert_list gen finish =
  let rec go acc = function
    | [] -> finish (List.rev acc)
    | x :: xs -> x |> convert gen (fun x_cps -> go (x_cps :: acc) xs)
  in
  go []

(* Converts a single function definition to its CPS form. *)
and convert_def gen (f, params, m) =
  let k = gensym gen in
  ( CLamVar f,
    List.map (fun x -> CLamVar x) params @ [ k ],
    m |> convert gen (fun m_cps -> CAppl (k, [ m_cps ])) )
```

The code is 74 lines long, including comments and blank lines. This is ~2.0 times shorter than the Rust version.

## Comparing the two implementations

Compiler development is characterized by:

 1. a lot of recursively defined data structured,
 2. a lot of complex data transformation.

How do Rust and OCaml handle these two aspects? Below is a brief summary:

 1. Recursive data structures:
    1. **OCaml**: recursive data structures are supported natively.
    2. **Rust**: we need to imitate data recursion by wrapping recursive occurences of `TermTree` and `CpsTerm` into `Rc`s [^rc-wrap] and `Box`es.
 2. Complex data transformation:
    1. **OCaml**:
       - Recursion is a common practice. OCaml has tail-call optimization and ["Tail Modulo Constructor (TMC)"] optimization.
       - Pattern matching is made very ergonomic. With `function`, we can pattern-match the "last parameter" [^currying-ocaml] of a function without introducing any extra indentation. (`function` also lets us omit the last parameter with oftentimes a dummy name like `term`; if you think the parameter name is useful, you can write it in the signature.) Lists can be matched as simply as `| [] -> ...` and `| x :: xs -> ...` without further hussle.
       - The majority of standard data structures are immutable. This makes it easy to reason about the code.
    2. **Rust**:
       - Recursion is uncommon. TCO is [not guaranteed] (compare it with OCaml's [`[@tailcall]`] and [`[@tail_mod_cons]`] annotations).
       - Pattern matching requires extra indentation and the need to explicate the matched parameter. There are several ways to "match" vectors, but they all are more verbose than OCaml's built-in syntax.
       - The majority of standard data structures are mutable, which inclines us towards the imperative style instead of the applicative style. Iterators provide us with a hatch to write code in the pipelined fashion, but first we need to `.iter()`/`.iter_mut()`/`.into_iter()` the data structure, perform the work, and then `.collect()`.

["Tail Modulo Constructor (TMC)"]: https://v2.ocaml.org/manual/tail_mod_cons.html
[not guaranteed]: https://stackoverflow.com/questions/59257543/when-is-tail-recursion-guaranteed-in-rust
[`[@tailcall]`]: https://stackoverflow.com/questions/23186717/verify-that-an-ocaml-function-is-tail-recursive
[`[@tail_mod_cons]`]: https://v2.ocaml.org/manual/tail_mod_cons.html

In addition to being syntactically more verbose than OCaml, Rust is a language without garbage collection. This forces us to make certain explicit choices about memory management: you can observe the plentitude of plumbing with boxes, references (both `&` and `Rc`), cloning, etc. Although it provides us with a greater sense of _how_ the code is executing, it brings very little value to the algorithm itself.

Even mutation can be more challenging in Rust:

```{.rust .numberLines}
fn gensym(i: RefCell<u32>) -> CpsVar {
    let x = CGenVar(i.clone().into_inner());
    i.replace_with(|&mut i| i + 1);
    x
}
```

In OCaml, it is just:

```{.ocaml .numberLines}
let gensym i =
  let x = CGenVar !i in
  i := !i + 1;
  x
```

Why `RefCell<u32>` instead of `&mut u32`? Because Rust requires us to have a single mutable reference to a value at any given time. This is a very reasonable requirement in multithreaded code, but we do not use more than one thread in our algorithm. We need `RefCell` just to circumvent this superfluous limitation [^update-refcell].

The last thing to note is the implementation of `convert_list` in Rust. Since `fn`s are inherently no more than code pointers, we need to pass `gen` and `finish` explicitly on each call to `go` [^rec-closures]. In turn, this leads us to duplicating the types of these variables in the signature of `go` (in Rust, there is no type inference of function parameters). In contrast, OCaml captures `gen` and `finish` automatically.

While the algorithm presented here is not very complex, it does already demonstrate the convenience of programming in a language from the ML family. However, let us see some more examples concerning type systems of both languages.

## Type safety: GADTs

Resource management aside, OCaml's type system is generally more expressive than that of Rust. For example, OCaml supports [Generalized Algebraic Data Types (GADTs)] to enforce certain invariants on the structure of data. Let us imagine an object language of booleans, integers, and operations upon them:

[Generalized Algebraic Data Types (GADTs)]: https://v2.ocaml.org/manual/gadts-tutorial.html

```{.rust .numberLines}
enum Term {
    Bool(bool),
    Not(Box<Term>),
    And(Box<Term>, Box<Term>),
    Int(i32),
    Neg(Box<Term>),
    Add(Box<Term>, Box<Term>),
}

enum Value {
    Bool(bool),
    Int(i32),
}
```

How do we write an evaluator for it? Here is a possible solution:

```{.rust .numberLines}
fn eval(term: &Term) -> Value {
    use Term::*;

    match term {
        Bool(b) => Value::Bool(*b),
        Not(m) => match eval(m) {
            Value::Bool(b) => Value::Bool(!b),
            _ => panic!("`Not` on a non-boolean value"),
        },
        And(m, n) => match (eval(m), eval(n)) {
            (Value::Bool(b1), Value::Bool(b2)) => Value::Bool(b1 && b2),
            _ => panic!("`And` on non-boolean values"),
        },
        Int(i) => Value::Int(*i),
        Neg(m) => match eval(m) {
            Value::Int(i) => Value::Int(-i),
            _ => panic!("`Neg` on a non-integer value"),
        },
        Add(m, n) => match (eval(m), eval(n)) {
            (Value::Int(i1), Value::Int(i2)) => Value::Int(i1 + i2),
            _ => panic!("`Add` on non-integer values"),
        },
    }
}
```

The solution is simple enough; however, it is not very robust. What happens if an input to `eval` is ill-typed? Take the following example:

```{.rust .numberLines}
fn main() {
    use Term::*;
    let term = Not(Box::new(And(Box::new(Bool(true)), Box::new(Int(42)))));
    dbg!(eval(&term));
}
```

The program panics with "`And` on non-boolean values", because the second operand of `And` must necessarily be a boolean, not an integer.

To prevent this kind of errors, we can use GADTs in OCaml:

```{.ocaml .numberLines}
type _ term =
  | Bool : bool -> bool term
  | Not : bool term -> bool term
  | And : bool term * bool term -> bool term
  | Int : int -> int term
  | Neg : int term -> int term
  | Add : int term * int term -> int term

let rec eval : type a. a term -> a = function
  | Bool b -> b
  | Not m -> not (eval m)
  | And (m, n) ->
      let b1, b2 = (eval m, eval n) in
      b1 && b2
  | Int i -> i
  | Neg m -> -eval m
  | Add (m, n) ->
      let i1, i2 = (eval m, eval n) in
      i1 + i2
```

Now what happens if we construct an ill-typed term?:

```{.ocaml .numberLines}
let () =
  let _term = Not (And (Bool true, Int 42)) in
  ()
```

It just will not type-check!:

```{.code .numberLines}
File "bin/main.ml", line 22, characters 35-41:
22 |   let _term = Not (And (Bool true, Int 42)) in
                                        ^^^^^^
Error: This expression has type int term
       but an expression was expected of type bool term
       Type int is not compatible with type bool
```

This is possible because we essentially encoded the object language type system in the definition of `term`. OCaml knows that `And` accepts boolean-typed terms, not integer-typed ones. In a real-world scenario, we can have an unrestricted `term` akin to Rust's `Term`, which is produced by parsing and elaborated further into a proper GADT-style `term`. The latter can be handled by `eval` (or `compile`, whatever).

## Type flexibility: First-class modules

Another neat feature of OCaml not present in Rust is [first-class modules]. Can you imagine a module that is stored in a variable, passed as a parameter, or returned from a regular function? This is what first-class modules are about. Suppose that your object language includes various fixed-size integers, such as `i8`/`u8`, `i16`/`u16`, and so on. With OCaml, you can represent them via (regular) modules:

[first-class modules]: https://dev.realworldocaml.org/first-class-modules.html

<p class="code-annotation">`fixed_ints.mli`</p>

```{.ocaml .numberLines}
(* [u8], [u16], etc. are defined by us. *)

module type S = sig
  type t

  val add : t -> t -> t
  val sub : t -> t -> t
  val mul : t -> t -> t
  val div : t -> t -> t
  val rem : t -> t -> t

  (* Some more operations here. *)
end

module U8 : S with type t = u8
module U16 : S with type t = u16
module U32 : S with type t = u32
module U64 : S with type t = u64
module U128 : S with type t = u128
module I8 : S with type t = i8
module I16 : S with type t = i16
module I32 : S with type t = i32
module I64 : S with type t = i64
module I128 : S with type t = i128
```

In the AST, we can represent integer values as follows:

```{.ocaml .numberLines}
type generic =
  | U8 of u8
  | U16 of u16
  | U32 of u32
  | U64 of u64
  | U128 of u128
  | I8 of i8
  | I16 of i16
  | I32 of i32
  | I64 of i64
  | I128 of i128
```

Having so many possible combinations of arithmetical operators `add`/`sub`/`mul`/`div`/`rem` and variously typed operands, how to implement evaluation sanely?

Here is an idea:

 1. Define a function `pair_exn` that maps two `generic`s into a first-class module `Pair`.
 2. Define a module `Pair` that implements `S` for a given pair of integers.
 3. Define a function `do_int_bin_op` that accepts `Pair` as a parameter and performs an operation `op` on the pair of integers.
 4. Call `do_int_bin_op` from `eval`.

In OCaml:

<p class="code-annotation">`fixed_ints.mli`</p>

```{.ocaml .numberLines}
module type Pair = sig
  type t

  include S with type t := t

  val pair : t * t
end

val pair_exn : generic * generic -> (module Pair)
```

The implementation of `pair` would be:

<p class="code-annotation">`fixed_ints.ml`</p>

```{.ocaml .numberLines}
let pair_exn =
  let make (type a) (module S : S with type t = a) (x, y) =
    (module struct
      include S

      let pair = x, y
    end : Pair)
  in
  function
  | U8 x, U8 y -> make (module U8) (x, y)
  | U16 x, U16 y -> make (module U16) (x, y)
  | U32 x, U32 y -> make (module U32) (x, y)
  | U64 x, U64 y -> make (module U64) (x, y)
  | U128 x, U128 y -> make (module U128) (x, y)
  | I8 x, I8 y -> make (module I8) (x, y)
  | I16 x, I16 y -> make (module I16) (x, y)
  | I32 x, I32 y -> make (module I32) (x, y)
  | I64 x, I64 y -> make (module I64) (x, y)
  | I128 x, I128 y -> make (module I128) (x, y)
  | _, _ -> raise (invalid_arg "Fixed_ints.pair_exn")
;;
```

Now we can write `eval` as follows:

```{.ocaml .numberLines}
(* Somewhere within the definition of [eval]. *)
| IntBinOp (op, ty, m, n) ->
  let x = extract_int_exn (eval m) in
  let y = extract_int_exn (eval n) in
  let (module Pair) = Fixed_ints.pair_exn (x, y) in
  do_int_bin_op op (module Pair)
```

`extract_int_exn` takes a value and extracts an integer `generic`, raising an exception if the value is not an integer.

Finally, `do_int_bin_op` is defined as follows:

```{.ocaml .numberLines}
let do_int_bin_op op (module S : Fixed_ints.Pair) =
  let x, y = S.pair in
  match op with
  | Add -> S.add x y |> S.to_value
  | Sub -> S.sub x y |> S.to_value
  | Mul -> S.mul x y |> S.to_value
  | Div -> S.div x y |> S.to_value
  | Rem -> S.rem x y |> S.to_value
;;
```

`S.to_value` converts a typed integer back to a value holding `generic`.

With the aid of first-class modules, we were able to implement evaluation of fixed-size integers without much boilerplate. The best you could do in Rust is to resort to `macro_rules!`, which are notorious for their hard-to-decipher syntax, shallow integration with the rest of the language, and poor IDE support.

## Final words

While Rust excels at resource management, OCaml turns out to be a more suitable choice for compiler development. We have not covered many other interesting features of it, such as [polymorphic variants], [custom binding operators], and [effect handlers]. Due to its completely static and flexible type system, OCaml has been historically used as a host language for many projects, including the [Frama-C toolchain], the [Coq theorem prover], and early versions of the Rust compiler itself.

[polymorphic variants]: https://v2.ocaml.org/releases/4.14/htmlman/polyvariant.html
[custom binding operators]: https://v2.ocaml.org/manual/bindingops.html
[effect handlers]: https://v2.ocaml.org/manual/effects.html
[Frama-C toolchain]: https://frama-c.com/
[Coq theorem prover]: https://coq.inria.fr/

OCaml is not without its flaws, though. The standard library and the overall ecosystem is clearly inferior to that of Rust. The full set of fixed-size integers found in Rust is not directly available in OCaml, although it can be implemented with a combination of native OCaml integers, the `Int32` and `Int64` modules from the standard library, and C FFI. (Pro tip: do not use [`ocaml-stdint`], it is unmaintained and is very buggy as of Aug 6, 2023. [`ocaml-integers`] is a more robust alternative but it lacks support for `Int8`, `Int16`, and 128-bit integers and has problems with documentation.)

[`ocaml-stdint`]: https://github.com/andrenth/ocaml-stdint
[`ocaml-integers`]: https://github.com/yallop/ocaml-integers

As Rust is gaining more and more popularity, more and more desperate developers from GitHub will start their compiler projects in it. I believe this can be a good decision either if 1) you are trying to learn Rust by writing "too many compilers" in it, or 2) you do really know what you are doing. If your intention is in compiler development itself, OCaml will save you a lot of time and undamaged nerves.

Other alternatives to consider is Haskell and various Lisp dialects. If you have already "tamed" Haskell (my congratulations and condolences), probably learning OCaml just for writing a compiler is not going to be worth it; if you have not, OCaml is a much more approachable language. Lisps can be very flexible, but they usually lack static type safety, opening a wide and horrible door to run-time errors.

## Appendix: Getting started with OCaml

Here is an easy way to get started with OCaml:

 1. [Install OCaml on Linux, macOS, *BSD, or Windows >>](https://ocaml.org/install)
 2. Install the [Dune build system]: `opam install dune`.
 3. Create a new project: `dune init project my_compiler`.

[Dune build system]: https://dune.readthedocs.io/en/stable/overview.html

The directory `my_compiler` will look like this:

```{.code .numberLines}
my_compiler/
├── bin
│   ├── dune
│   └── main.ml
├── _build
│   └── log
├── dune-project
├── lib
│   └── dune
├── my_compiler.opam
└── test
    ├── dune
    └── my_compiler.ml
```

 1. `bin/` is for setup code and CLI.
 2. `lib/` is where most of the code lives.
 3. `test/` is for tests.

I recommend [`alcotest`] for unit tests and [`ppx_deriving`] for the deriving functionality (akin to `#[derive(...)]` from Rust). Install them as follows:

[`alcotest`]: https://github.com/mirage/alcotest
[`ppx_deriving`]: https://github.com/ocaml-ppx/ppx_deriving

```{.code .numberLines}
$ opam install alcotest
$ opam install ppx_deriving
```

Edit `my_compiler/lib/dune` as follows:

```{.code .numberLines}
(library
 (name my_compiler)
 (preprocess
  (pps ppx_deriving.show ppx_deriving.eq)))
```

And `my_compiler/test/dune` as follows:

```{.code .numberLines}
(test
 (name my_compiler)
 (libraries my_compiler alcotest))
```

 1. Type `dune build` to build the project.
 2. Type `dune test` to run the tests.
 3. Type `dune exec my_compiler` to execute the binary.

You can now create a file `foo.ml` with a corresponding `foo.mli` in `my_compiler/lib` and access it as `My_compiler.Foo` from `bin/` and `lib/`.

For test coverage, consider using [`bisect_ppx`].

[`bisect_ppx`]: https://github.com/aantron/bisect_ppx

If you know Rust, you will find OCaml very familiar. I recommend the following resources for learning the language:

 - [OCaml Programming: Correct + Efficient + Beautiful](https://cs3110.github.io/textbook/cover.html)
 - [Real World OCaml](https://dev.realworldocaml.org/)
 - [The official tutorial](https://v2.ocaml.org/manual/coreexamples.html) (can be read in an evening!)

## References

[^cps-compiler]: CPS is the central representation of the compiler Standard ML of New Jersey.

[^update-rust-arenas]: _Update: several people have suggested to use arenas (regions) instead of the approach I have demonstrated. I am well aware of the technique; however, I do not think that arenas would make a significant difference in code clarity and ergonomics. Flattening an AST has many performance benefits, such as spatial locality and cheap allocation and deallocation, but they add a little value to the overall discussion._

[^cps-ocaml]: You can access the code itself and accompanying tests [here](https://gist.github.com/Hirrolot/d16dc5e78639db6e546b5054afefd142).

[^rc-wrap]: `Rc` was chosen to avoid expensive cloning of `TermTree`s in some places.

[^currying-ocaml]: To be precise, all functions are curried in OCaml, so `function` just "defines" a function with a single parameter and pattern-matches on it.

[^update-refcell]: _Update: this is actually not true that this requirement is only needed in multithreaded code. However, I do still think it is superfluous in the code I have suggested._

[^rec-closures]: Unfortunately, closures provide us with no solution here: they cannot be called recursively, at least without prior hoodoo.
