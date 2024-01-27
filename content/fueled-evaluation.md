<div class="introduction">

<p class="discussions">[r/dependent_types](https://reddit.com/r/dependent_types/comments/1ac9gw0/fueled_evaluation_for_decidable_type_checking/)</p>

Beta normalization in lambda calculus is a procedure that reduces a lambda term to its beta normal form. Here are some examples:

| Term | Beta normal form |
|----------|---------|
| `(\x -> x y) z` | `z y` |
| `\x -> (\y -> x y) z` | `\x -> x z` |
| `x y` | `x y` |

A beta normal form of a term is characterized by the fact that it cannot be "reduced further" -- even if we look under lambdas, as in the second example.

In ordinary functional languages, complete beta normalization is typically not implemented; for example, if you type the following into `ghci`:

```{.code .numberLines}
ghci> (\x -> (\y -> y) 42)

<interactive>:3:1: error:
    • No instance for (Show (p0 -> Integer))
        arising from a use of ‘print’
        (maybe you haven't applied a function to enough arguments?)
    • In a stmt of an interactive GHCi command: print it
```

You will get a complaint from Haskell that it cannot print a function body! Moreover, beta normalization permits _symbolic computation_, which is a very crucial point, as we will see shortly. Let us test our first example in `ghci`:

```{.code .numberLines}
ghci> (\x -> x y) z

<interactive>:4:10: error: Variable not in scope: y

<interactive>:4:13: error: Variable not in scope: z :: t0 -> t
```

The errors happen because Haskell's REPL requires a _concrete value_ for every variable in scope! In contrast, beta normalization is free from such a restriction -- if the value of a variable is unknown at some point, it just uses its _symbolic name_ and proceeds with the computation. The transition from ordinary computation to symbolic computation is akin to the transition from arithmetic to algebra.

Perhaps the most frequent use case of beta normalization lies in dependently typed languages, such as Idris, Agda, Coq, Lean, and others. The reason is that dependent types require a type checker to perform an "equality check" between two types, which may contain arbitrary terms; beta normalization is used to obtain normal forms of the two types and then compare them syntactically [^beta-conv].

Here comes the problem of ensuring termination of type checking. Since the untyped lambda calculus is Turing-complete, evaluation of a lambda term will not terminate, and since dependent type checking may involve evaluation of arbitrary terms, it becomes undecidable. Existing dependently typed languages are designed in such a way that they are _not_ Turing-complete, thus solving the issue. However, what if we deliberately plan to perform _any_ conceivable computation?

Then, we need some way of ensuring termination of evaluation. In this essay, I am to suggest a simple technique called _fueled evaluation_, which amounts to annotating every term with a maximum number of times it can be evaluated.

</div>

## Vanilla Normalization by Evaluation

Let me first lay out the foundation upon which I will demonstrate fueled evaluation. We are going to use an approach of normalizing higher-order terms called _Normalization by Evaluation (NbE)_. Simply speaking, we are going to have two functions, `eval` and `quote`: the former evaluates the input term up to lambdas (i.e., it does look into lambda bodies), whereas the latter function pushes evaluation under lambdas, thereby obtaining the normal form. By combining the two functions, roughly `quote . eval`, we would obtain a full lambda calculus normalizer.

The following is the definition of a lambda term, in OCaml:

```{.ocaml .numberLines}
type term = Lam of term | Var of int | Appl of term * term
[@@deriving show { with_path = false }]
```

We use the first-order representation of lambda abstractions, forcing `Var` to contain a De Bruijn _index_ (starting at 0) following the [well-known scheme].

[well-known scheme]: https://en.wikipedia.org/wiki/De_Bruijn_index

To be able to write a sequence of applications conveniently, we define the following shortcut function:

```{.ocaml .numberLines}
let appl (f, list) = List.fold_left (fun m n -> Appl (m, n)) f list
```

Next, here is the representation of values:

```{.ocaml .numberLines}
type value = VClosure of value list * term | VNt of neutral
and neutral = NVar of int | NAppl of neutral * value
```

Here we use `VClosure` to represent a closure function; it contains a lambda body that _closes_ over the environment of values. We also use `VNt` to represent computations blocked on a value of some unknown variable. This is an essential piece of NbE: while ordinary computation requires variables to hold _concrete_ values at any point, NbE permits variables to be unknown during evaluation.

We define some shortcut functions for convenience:

```{.ocaml .numberLines}
let vvar lvl = VNt (NVar lvl)
let vappl (m, n) = VNt (NAppl (m, n))
```

Now we define the evaluator itself, which throws `term` under `rho:value list` into `value`:

```{.ocaml .numberLines}
let rec eval ~rho = function
  | Lam m -> VClosure (rho, m)
  | Var idx -> List.nth rho idx
  | Appl (m, n) -> (
      let m_val = eval ~rho m in
      let n_val = eval ~rho n in
      match m_val with
      | VClosure (rho, m) -> eval ~rho:(n_val :: rho) m
      | VNt neut -> vappl (neut, n_val))
```

The code is pretty self-explanatory. The only thing to note is that values in the environment `rho` (pronounced as "row" [^rho]) are indexed via De Bruijn indices (see the case of `Var idx`), making access as easy as possible.

The next piece of code is the `quote` function, which throws `value` under `lvl:int` back into `term`:

```{.ocaml .numberLines}
let rec quote ~lvl = function
  | VClosure (rho, m) ->
      let m_nf = normalize_at ~lvl ~rho m in
      Lam m_nf
  | VNt neut -> quote_neut ~lvl neut

and quote_neut ~lvl = function
  | NVar var -> Var (lvl - var - 1)
  | NAppl (neut, n_val) ->
      let m_nf = quote_neut ~lvl neut in
      let n_nf = quote ~lvl n_val in
      Appl (m_nf, n_nf)
```

In the case of `NVar var`, we convert a De Bruijn _level_ `var` into a corresponding De Bruijn _index_ [^de-bruijn-level]; this is what the formula `lvl - var - 1` does. In the case of `VClosure (rho, m)`, we explicitly push evaluation under binders by calling `normalize_at ~lvl ~rho m`. The function `normalize_at` is defined as follows:

```{.ocaml .numberLines}
and normalize ~lvl ~rho term = quote ~lvl (eval ~rho term)

and normalize_at ~lvl ~rho term =
  normalize ~lvl:(lvl + 1) ~rho:(vvar lvl :: rho) term
```

Hooray, we have our vanilla NbE ready! Let us test it with some trivial examples.

Applying `id` to itself:

```{.ocaml .numberLines}
(* (Lam (Var 0)) *)
let id = Lam (Var 0) in
normalize ~lvl:0 ~rho:[] @@ Appl (id, id)
```

Computing a boolean expression `if true then false else true`:

```{.ocaml .numberLines}
(* (Lam (Lam (Var 0))) *)
let t = Lam (Lam (Var 1)) in
let f = Lam (Lam (Var 0)) in
let if_then_else = Lam (Lam (Lam (Appl (Appl (Var 2, Var 1), Var 0)))) in
normalize ~lvl:0 ~rho:[] @@ appl (if_then_else, [ t; f; t ])
```

Computing `SKSK` (from the [SKI calculus]):

[SKI calculus]: https://en.wikipedia.org/wiki/SKI_combinator_calculus

```{.ocaml .numberLines}
(* (Lam (Lam (Var 1))) *)
let k = Lam (Lam (Var 1)) in
let s = Lam (Lam (Lam (Appl (Appl (Var 2, Var 0), Appl (Var 1, Var 0))))) in
normalize ~lvl:0 ~rho:[] @@ appl (s, [ k; s; k ])
```

All the results are correct. Now, let us extend the code to support fueled evaluation.

## Fueled evaluation

Before expounding on the idea, let me show the actual code [^gist].

As per term representation, we need to augment every term (including all of its subterms) with a mutable integer called _fuel_. We could do something like this:

```{.ocaml .numberLines}
type term = term_def * int ref

and term_def = Lam of term | Var of int | Appl of term * term
[@@deriving show { with_path = false }]
```

However, this approach involves a few maintenance problems. First, besides the type checker and the evaluator, there may be more moving parts in the compiler that manipulate terms -- and they do not care a little about fuel! Second, if we already have these moving parts, and we want to extend our evaluation mechanism to support fueled evaluation, we need to scrupulously modify all of them. Third, in a real implementation, we may want to write _a lot_ of unit tests, and manually inserting fuel into every term and subterm is going to be a very tedious task.

The solution is to use ML modules -- they really shine in this case. We begin by defining the following functor [^functor]:

```{.ocaml .numberLines}
module Make_term (S : sig
  type 'a t [@@deriving show]
end) =
struct
  type t = def S.t

  and def = Lam of t | Var of int | Appl of t * t
  [@@deriving show { with_path = false }]

  let _ = S.show
end
```

(The `let _ = S.show` line just suppresses a compiler warning, ignore it.)

The functor `Make_term` takes a module with a polymorphic type component `t` and outputs a new module that contains a (different) type `t` that specializes `S.t` to `def`, which is the term definition itself. Inside `def`, we refer to the previously defined `t` so that not only the upper level of a term is annotated but also all of its subterms.

We can now define two versions of terms, the first being called `T`:

```{.ocaml .numberLines}
module T = Make_term (struct
  type 'a t = 'a [@@deriving show]
end)
```

And the second being called `Nbe_term`:

```{.ocaml .numberLines}
module Nbe_term = Make_term (struct
  type 'a t = 'a * int ref [@@deriving show]
end)
```

The difference between these two should be pretty clear: while `T` is the regular term representation we had in the previous section, `Nbe_term` possesses the additional fuel component. Besides the evaluator, code that used to deal with terms can be left untouched -- save a few lines of auxiliary code.

Moving further, we need a way to turn `T.def` into `Nbe_term.def`. This can be accomplished quite straightforwardly:

```{.ocaml .numberLines}
let limit = 1000

let tank =
  let open Nbe_term in
  let rec go term = (go_def term, ref limit)
  and go_def = function
    | T.Lam m -> Lam (go m)
    | T.Var idx -> Var idx
    | T.Appl (m, n) -> Appl (go m, go n)
  in
  go
```

The `limit` constant is the maximum number of times a certain term can be passed to the evaluator, as we will see shortly. The `tank` function just recurses over the structure of an input term, annotating every subterm with `ref limit`.

The key change lies in the evaluator code:

```{.ocaml .numberLines}
exception Out_of_fuel of Nbe_term.def

let check_limit_exn (term, fuel) =
  if !fuel > 0 then decr fuel else raise (Out_of_fuel term)

let rec eval ~rho (term, fuel) =
  let open Nbe_term in
  check_limit_exn (term, fuel);
  match term with
  | Lam m -> VClosure (rho, m)
  | Var idx -> List.nth rho idx
  | Appl (m, n) -> (
      let m_val = eval ~rho m in
      let n_val = eval ~rho n in
      match m_val with
      | VClosure (rho, m) -> eval ~rho:(n_val :: rho) m
      | VNt neut -> vappl (neut, n_val))
```

The only needed change is to add the extra `check_limit_exn (term, fuel)` line to the beginning of the function. If the fuel variable is zero, we raise an exception called `Out_of_fuel`; otherwise, we merely decrement the fuel and proceed with the evaluation.

The rest of the code is concerned with quotation and full normalization. Unlike `eval`, it need not be changed at all, but let me include it for reference:

```{.ocaml .numberLines}
let rec quote ~lvl = function
  | VClosure (rho, m) ->
      let m_nf = normalize_at ~lvl ~rho m in
      T.Lam m_nf
  | VNt neut -> quote_neut ~lvl neut

and quote_neut ~lvl = function
  | NVar var -> T.Var (lvl - var - 1)
  | NAppl (neut, n_val) ->
      let m_nf = quote_neut ~lvl neut in
      let n_nf = quote ~lvl n_val in
      T.Appl (m_nf, n_nf)

and normalize ~lvl ~rho term = quote ~lvl (eval ~rho term)

and normalize_at ~lvl ~rho term =
  normalize ~lvl:(lvl + 1) ~rho:(vvar lvl :: rho) term
```

One thing to note is that the inferred types are going to be different. Previously, `normalize` and `normalize_at` both had the following function type:

```{.ocaml .numberLines}
lvl:int -> rho:value list -> term -> term
```

But now, the type of `normalize` and `normalize_at` is:

```{.ocaml .numberLines}
lvl:int -> rho:value list -> Nbe_term.t -> T.def
```

As you can observe, the new input term type is `Nbe_term.t` instead of `T.t`. This is quite correct because, in `quote`, we still need to call `normalize_at` with `m : Nbe_term.t` coming from `VClosure (rho, m)`; however, for the sake of convenience of usage, we should be able to _redefine_ `normalize` as follows [^normalize-redef]:

```{.ocaml .numberLines}
let normalize term = normalize ~lvl:0 ~rho:[] (tank term)
```

As expected, the final type of `normalize` is `T.def -> T.def`.

## Testing termination

The new implementation is able to detect potentially non-terminating computation. The following function tests a term:

```{.ocaml .numberLines}
let test term =
  try
    let _nf = normalize term in
    print_endline "Ok."
  with Out_of_fuel term ->
    Printf.printf "The computational limit is reached for %s.\n"
      (Nbe_term.show_def term)
```

Let us define some [Church numerals] for testing:

[Church numerals]: https://en.wikipedia.org/wiki/Church_encoding#Church_numerals

```{.ocaml .numberLines}
let zero = T.(Lam (Lam (Var 0)))
let succ = T.(Lam (Lam (Lam (Appl (Var 1, Appl (Appl (Var 2, Var 1), Var 0))))))

let mul =
  T.(Lam (Lam (Lam (Lam (Appl (Appl (Var 3, Appl (Var 2, Var 1)), Var 0))))))

let appl (f, list) = List.fold_left (fun m n -> T.Appl (m, n)) f list
let one = appl (succ, [ zero ])
let two = appl (succ, [ one ])
let three = appl (succ, [ two ])
let four = appl (succ, [ three ])
let five = appl (succ, [ four ])
let ten = appl (mul, [ five; two ])
let hundred = appl (mul, [ ten; ten ])
let thousand = appl (mul, [ hundred; ten ])
```

If we apply `id` to itself, the console output will be `Ok.`:

```{.ocaml .numberLines}
let id_appl = T.(Appl (Lam (Var 0), Lam (Var 0))) in
test id_appl
```

If we try to compute 5000 using Church numerals, the output will be `Ok.` as well:

```{.ocaml .numberLines}
let n5k = appl (mul, [ thousand; five ]) in
test n5k
```

However, normalization of the `omega` combinator yields a failure:

```{.ocaml .numberLines}
let self_appl = T.(Lam (Appl (Var 0, Var 0))) in
let omega = T.Appl (self_appl, self_appl) in
test omega
```

The output is `The computational limit is reached for (Appl (((Var 0), ref (0)), ((Var 0), ref (0))))`, as expected.

## Informal reasoning

Why is fueled evaluation implemented exactly as it is?

The informal reasoning behind this approach is as follows. Usually, time complexity of an algorithm depends on the size of input (the so-called "Big O notation"); in this case, the bigger the input, the more computation steps the algorithm will perform. For example, if the time complexity is O(n^2), the number of steps for some input of size 5 is 25, whereas if the input size is 10, the number of steps increases to 100.

Of course, for a language to be Turing-complete, it must permit terms whose time complexity cannot be described as a function of an input size. Therefore, we cannot determine the complexity of `eval`, but we can instead approximate the number of invocations of `eval` by augmenting each input term with some suitable integer: the bigger the term, the more times `eval` can be (recursively) invoked [^halting-problem].

Termination of `eval ~rho term` for any `rho` and `term : Nbe_term.t` follows from the following observations:

 - All terms passed to `eval` are drawn from the finite set of all subterms of `term`.
 - The number of times a certain term can be passed to `eval` is limited.

Since `quote`, `normalize`, and `normalize_at` only call `eval` and reduce their arguments structurally, they terminate as well.

However, this does not _guarantee_ termination (i.e., decidability) of type checking. For example, if you do some sophisticated machinery during type checking, such as certain forms of unification, you may still end up with an undecidable type system. Thus, the only thing that the aforestated property guarantees is that the type checker will not be undecidable _due to_ the evaluation mechanism.

There are several advantages of fueled evaluation over more traditional approaches:

 - In contrast to the [Calculus of Inductive Constructions (CIC)], which only allows arguments to be reduced structurally to ensure termination, the advantage of our approach is that it does not exclude any programming techniques -- it is always possible to adjust the fuel constant for specific needs.

 - In contrast to termination checking with homeomorphic embedding [^MetacompBySupercomp] [^ConvergenceSorensen], which requires accumulating a history of computation and performing a structural check for every previous term in the history, our checking conditions possess very little performance penalty. This is very crucial for any dependently typed language. In addition, the homeomorphic embedding relation requires a form of small-step operational semantics in order to have a traceable history of computation, whereas our NbE algorithm more resembles big-step semantics.

 - We can extend the language with more constructions without worrying about introducing potential non-termination. This is because fueled evaluation considers only the size metric of terms, not their actual contents.

[Calculus of Inductive Constructions (CIC)]: https://coq.inria.fr/doc/v8.9/refman/language/cic.html

Limiting potentially infinite computations with a configurable constant is widely used in practical compiler implementations:

 - GCC and Clang have the `-ftemplate-depth` option (resp. [GCC] and [Clang]) to "set the maximum instantiation depth for template classes".

 - Rust has the [`recursion_limit` attribute] to "set the maximum depth for potentially infinitely-recursive compile-time operations like macro expansion or auto-dereference".

 - Scala 3 has the [`-Xmax-inlines` option] described as the "maximal number of successive inlines".

 - Lean 4 has the [`maxRecDepth` option] described as the "maximum recursion depth" (for macros and regular computation).

[GCC]: https://gcc.gnu.org/onlinedocs/gcc/C_002b_002b-Dialect-Options.html#index-ftemplate-depth
[Clang]: https://clang.llvm.org/docs/ClangCommandLineReference.html#cmdoption-clang-ftemplate-depth
[`recursion_limit` attribute]: https://doc.rust-lang.org/reference/attributes/limits.html#the-recursion_limit-attribute
[`-Xmax-inlines` option]: https://docs.scala-lang.org/scala3/guides/migration/options-new.html#advanced-settings
[`maxRecDepth` option]: https://github.com/leanprover/lean4/blob/ec39de8caed5f86604cec2b4b788d917aaebbe34/src/Init/Prelude.lean#L4365

This provides empirical evidence that the suggested approach should work well in practice. Note that we could just as well fix _globally_ the limit for `eval`, but such a solution would be strictly less flexible. The key idea behind fueled evaluation is to expand and shrink the freedom of evaluation according to the given input -- the bigger the term, the more freedom is granted to `eval` [^eval-freedom].

## The type checker sketch

Although the type checking algorithm is out of the scope of the current writing, I briefly sketch how it could be adapted to fueled evaluation.

It is reasonable to give the end user the right to configure the `limit` constant at will. Therefore, the `Typing` module should contain the `Make` functor that accepts `limit` as a run-time parameter. This situation can roughly look as follows [^bidirectional-typing]:

```{.ocaml .numberLines}
module type Opts = sig
  val fuel : int
end

module Make (_ : Opts) : sig
  val infer
    :  rho:Value.t list
    -> gamma:Gamma.t
    -> Raw_term.t
    -> (Term.t * Value.t, Report.t) result

  val check
    :  rho:Value.t list
    -> gamma:Gamma.t
    -> Raw_term.t * Value.t
    -> (Term.t, Report.t) result
end
```

The implementation of `Typing` would then look as:

```{.ocaml .numberLines}
module type Opts = sig
  val fuel : int
end

module Make (Opts : Opts) = struct
  let eval ~rho term = Nbe.eval ~rho (Value.Term'.tank ~fuel:Opts.fuel term)

  let rec infer ~(rho : Value.t list) ~(gamma : Gamma.t) = function (* ... *)
  and check ~(rho : Value.t list) ~(gamma : Gamma.t) = function (* ... *)
end
```

Whenever we elaborate a new term, we can submit it to the inner `eval` function (from within `Make`) to obtain a value. The `Term'` submodule of the module `Value` is a functor application of the following form:

```{.ocaml .numberLines}
module Term' = struct
  include Term.Make (struct
      type 'a t = 'a * int ref [@@deriving show]
    end)

  let tank ~fuel = (* ... *)
end
```

With the following signature:

```{.ocaml .numberLines}
module Term' : sig
  include module type of Term.Make (struct
      type 'a t = 'a * int ref [@@deriving show]
    end)

  val tank : fuel:int -> Term.t -> t
end
```

In turn, the `Term` module signature must include the `Make` functor that defines the term:

```{.ocaml .numberLines}
module Make (S : sig
    type 'a t [@@deriving show]
  end) : sig
  type t = def S.t

  and def = (* ... *)
  [@@deriving show]
end
```

And apply it as follows (in the same signature):

```{.ocaml .numberLines}
include module type of Make (struct
    type 'a t = 'a [@@deriving show]
  end)
```

The implementation of `Term` would be:

```{.ocaml .numberLines}
module Make (S : sig
    type 'a t [@@deriving show]
  end) =
struct
  type t = def S.t

  and def = (* ... *)
  [@@deriving show]

  let _ = S.show
end

include Make (struct
    type 'a t = 'a [@@deriving show]
  end)
```

To use the `Typing` module, all we need is to instantiate the functor `Typing.Make` with the constant `cli_fuel` coming as a run-time CLI parameter [^cli-fuel]:

```{.ocaml .numberLines}
let module Typing =
  Typing.Make (struct
    let fuel = cli_fuel
  end)
in
(* ... *)
```

Then, writing `Typing.infer ~rho ~gamma prog` (same with `check`) would launch the type checking process for `prog : Raw_term.t`.

Finally, it would make practical sense to do these two things:

 - Exception handling. Do not forget to handle the `Out_of_fuel` exception and properly signal the error to the end user.
 - Source code tracking. Add the `SrcPos` variant to `Term.Make.def`. Include a term and a source segment for it (can be a pair of `Lexing.position`). If `Out_of_fuel` is caught up, highlight the source text that raised the exception.

## Final words

Although the demonstrated approach is concerned with pure untyped lambda calculus, the same reasoning applies to various typed lambda calculi, which can be non-terminating for many reasons, starting from unrestricted recursion using a built-in fixed-point combinator and ending with [Girard's paradox].

[Girard's paradox]: https://en.wikipedia.org/wiki/System_U#Girard's_paradox

There is a paper from Neil D. Jones called _"Call-by-value Termination in the Untyped lambda-calculus"_ [^cbv-termination]. The paper suggests an _offline_ approach to termination checking, i.e., checking termination before actually executing the program. In contrast, fueled evaluation is an _online_ checking algorithm -- it checks termination _during_ execution. The approach from the paper rests on the fact from pure lambda calculus that the set of subexpressions of the evaluated term is a subset of subexpressions of the input program (see lemma 3.6, definition 3.7, and lemma 3.8). Unfortunately, this surprising property breaks after adding built-ins to the language: consider an expression `!true` resulting in `false`, which is not a subexpression of `!true` (same for integers, strings, etc.) Whether the approach demonstrated in the paper can be generalized to a more practical language than pure lambda calculus is a matter of further research.

What is left to do is to apply the technique to a real-world implementation of a dependently typed programming language. I am currently working on one, which is not made public yet. As far as things go, the technique works well enough -- it is sufficiently _simple, stupid_ to reason about, test, and maintain.

If I do not forget, I will link an open-sourced repository someday here.

## References

[^beta-conv]: Note that it is typically not implemented in such a brutal way. A more practical approach is to write a [beta conversion checker] that _gradually_ evaluates a pair of terms until either there is no more left to evaluate or equality fails. However, full normalization is still used when [inferring dependent function types] or [printing terms].

[beta conversion checker]: https://github.com/AndrasKovacs/elaboration-zoo/blob/2cbde286207f3b4bf24631b40656aa63d717ce10/02-typecheck-closures-debruijn/Main.hs#L138
[inferring dependent function types]: https://github.com/AndrasKovacs/elaboration-zoo/blob/2cbde286207f3b4bf24631b40656aa63d717ce10/03-holes/Main.hs#L662
[printing terms]: https://github.com/AndrasKovacs/elaboration-zoo/blob/2cbde286207f3b4bf24631b40656aa63d717ce10/02-typecheck-closures-debruijn/Main.hs#L189

[^rho]: The naming is borrowed from Thierry Coquand's paper _"An algorithm for type-checking dependent types"_ (1996).

[^de-bruijn-level]: Contrary to a De Bruijn index, a De Bruijn level is a natural number indicating the number of binders between the variable's binder and the term's _root_. For example, the term `\x -> \y -> x` is represented as `\_ -> \_ -> 0`, where `0` is the De Bruijn level of `x`. The De Bruijn level of `y` would be `1` if we used it instead of `x`.

[^gist]: The complete code for fueled evaluation can be accessed as a [GitHub gist].

[GitHub gist]: https://gist.github.com/Hirrolot/0cbf1d44fab5a265ac3fd891d20fc1c4

[^functor]: You can think of a functor as a module-level function: a function that maps a module expression to another module expression. A more thorough explanation can be found in the [corresponding chapter](https://dev.realworldocaml.org/functors.html) of _"Real World OCaml"_.

[^normalize-redef]: The redefinition of `normalize` is a good example of name shadowing. We could name the first version of `normalize` as `normalize_nbe`, but 1) the code would need to be changed, 2) the longer name would be less convenient to use, and 3) it would pollute the whole namespace, causing confusion between the two versions.

[^halting-problem]: According to the [halting problem], it is generally impossible to tell if an arbitrary program will finish running -- so we are doomed to use one or another approximation in any case. However, it is not that bad as it may seem to be: type systems are _already_ approximations of operationally sound programs.

[halting problem]: https://en.wikipedia.org/wiki/Halting_problem

[^MetacompBySupercomp]: Robert Glück and Morten Heine Sørensen. 1996. A Roadmap to Metacomputation by Supercompilation. In Selected Papers from the International Seminar on Partial Evaluation. Springer-Verlag, Berlin, Heidelberg, 137–160.

[^ConvergenceSorensen]: 
Sørensen, M.H.B. (1998). Convergence of program transformers in the metric space of trees. In: Jeuring, J. (eds) Mathematics of Program Construction. MPC 1998. Lecture Notes in Computer Science, vol 1422. Springer, Berlin, Heidelberg. https://doi.org/10.1007/BFb0054297

[^eval-freedom]: As I have mentioned earlier, this is only an approximation. For example, `omega` from the previous section is pretty small, but it requires an infinite number of steps to normalize. On the other hand, a huge term can be quickly normalizable if most of its subterms are never executed (i.e., it contains many "dead paths").

[^bidirectional-typing]: The `infer` and `check` functions manifest a [bidirectional typing] algorithm -- a _de facto_ standard for dependent type checking.

[bidirectional typing]: https://davidchristiansen.dk/tutorials/bidirectional.pdf

[^cli-fuel]: The instantiation of `Typing.Make` on `cli_fuel` provides a good example of _run-time configuration of ML modules_. Rudimentary module systems of most programming languages make this pattern impossible. The best we could do in Haskell, Rust, Java, or C# is to have a sort of record or class that accepts fuel as a parameter, together with the methods `infer` and `check`. With ML modules, we did not need to alter the structure of our code -- the transition from static modules to configurable functors happened _naturally_.

[^cbv-termination]: Jones, N., & Bohr, N. (2008). Call-by-value Termination in the Untyped lambda-calculus. Logical Methods in Computer Science, Volume 4, Issue 1.
