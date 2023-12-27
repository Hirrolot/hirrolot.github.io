<div class="introduction">

<p class="discussions">[HN](https://news.ycombinator.com/item?id=36645356) · [r/ProgrammingLanguages](https://www.reddit.com/r/ProgrammingLanguages/comments/14u5o1n/how_to_keep_lambda_calculus_simple/)</p>

Once upon a time, my curiosity of dependent types transitioned into an endeavour to learn how they work under the bonnet. Then I found the paper called ["A Tutorial Implementation of a Dependently Typed Lambda Calculus"] or just "Simply Easy" -- and began reading it. However, the affairs happening there were so complex and hard that I gave up eventually and postponed it for later.

["A Tutorial Implementation of a Dependently Typed Lambda Calculus"]: https://www.andres-loeh.de/LambdaPi/

Some months went by. Meanwhile, I conditioned myself into reading some more obscure papers, but finally understood how dependent types operate. Then I returned to the "Simply Easy" paper and read it from cover to cover. Everything began to be crystal clear. The particular pain points that confused me during the first reading I could then discern and evaluate objectively.

Now I am writing this post for those who are in the same situation as I was. I will try to explain the implementation from the paper, and demonstrate two ways to simplify it so that it becomes far more readable than the original. For those who have not read the paper yet, I highly recommend to forget about it and this post altogether -- you have so many nicer things to do in your splendid life.

</div>

## The original implementation

Let us start with the AST definitions. Terms are represented as follows [^sources]:

```{.haskell .numberLines}
data ITerm
  = Ann CTerm Type
  | Bound Int
  | Free Name
  | ITerm :@: CTerm
  deriving (Show, Eq)

data CTerm
  = Inf ITerm
  | Lam CTerm
  deriving (Show, Eq)

data Name
  = Global String
  | Local Int
  | Quote Int
  deriving (Show, Eq)

data Type
  = TFree Name
  | Fun Type Type
  deriving (Show, Eq)
```

So we have `ITerm` and `CTerm` denoting _inferrable_ and _checkable_ terms, respectively. This distinction stems from the type checking approach taken by the paper, which is called _bidirectional type checking_. More about it later. For now, just focus on the data constructors, which are pretty self-explanatory; the only curious cases are `Ann` and `Inf`, but let us ignore them as well for a moment.

We distinguish two kinds of variables: `Bound` variables and `Free` ones. The first kind of variables is called [_De Bruijn indices_]: each bound variable is a natural number (starting from zero) that designates the number of binders between itself and a corresponding binder. For example, the term `\x -> \y -> x` (the "K combinator") is represented as `\_ -> \_ -> 1`. (If we were returning `y` instead of `x`, we would write `0` instead.)

[_De Bruijn indices_]: https://en.wikipedia.org/wiki/De_Bruijn_index

The second kind of variables can divide into three subkinds: `Global`, `Local` and `Quote`. Global variables are represented as strings and should not possess any surprises; the two other kinds will become clearer later.

Now, terms must evaluate to something. Evaluated terms are called _values_:

```{.haskell .numberLines}
data Value
  = VLam (Value -> Value)
  | VNeutral Neutral

data Neutral
  = NFree Name
  | NApp Neutral Value

vfree :: Name -> Value
vfree n = VNeutral (NFree n)
```

Values are either fully evaluated lambda abstractions `VLam` or neutral terms `VNeutral`. Lambda abstractions are represented as Haskell functions; you can say that you are using _Higher-Order Abstract Syntax_ as a _semantic domain_ here to assert self-dominance. Neutral terms are computations blocked by a variable whose value is unknown: for example, if we apply some value `brrr` to a variable `x`, we will obtain `NApp (NFree $ Global "x") brrr`. This is what people call _open evaluation_, that is, evaluation that can account for free variables.

The evaluation algorithm itself:

```{.haskell .numberLines}
type Env = [Value]

type NameEnv v = [(Name, v)]

iEval :: ITerm -> (NameEnv Value, Env) -> Value
iEval (Ann e _) d = cEval e d
iEval (Free x) d = case lookup x (fst d) of Nothing -> (vfree x); Just v -> v
iEval (Bound ii) d = (snd d) !! ii
iEval (e1 :@: e2) d = vapp (iEval e1 d) (cEval e2 d)

vapp :: Value -> Value -> Value
vapp (VLam f) v = f v
vapp (VNeutral n) v = VNeutral (NApp n v)

cEval :: CTerm -> (NameEnv Value, Env) -> Value
cEval (Inf ii) d = iEval ii d
cEval (Lam e) d = VLam (\x -> cEval e (((\(e, d) -> (e, (x : d))) d)))
```

Since we have two kinds of terms, the evaluation function is divided into two kinds as well. Here is a case-by-case explanation of the algorithm:

 1. `iEval (Ann e _) d`: this rule just pushes evaluation to the checkable term `e`.
 2. `iEval (Free x) d`: if we can find the free variable `x` in the environment of free variables `fst d`, we return its value; otherwise, we return a neutral value `vfree x`.
 3. `iEval (Bound ii) d`: we return the value of the bound variable `ii` from the environment of bound variables `snd d`. The nice trick here is that the _i_-th position within `Env` corresponds to the value of the variable _i_ (i.e., the environment is De Bruijn-_indexed_), so the ridiculously ugly operator `!!` is enough.
 4. `iEval (e1 :@: e2) d`: we evaluate the two terms and hand them to `vapp`. In `vapp`, if the first value is `VLam`, we just apply the second value to it; otherwise, we return a neutral value `VNeutral (NApp n v)`.
 5. `cEval (Inf ii) d`: as with `Ann`, we just push evaluation to the inferrable term `ii`.
 6. `cEval (Lam e) d`: we return a Haskell function as an evaluated lambda abstraction. In it, we accept `x` as an evaluated argument. Our job within the function is to evaluate the body `e` in an environment extended with `x`. Since `Env` indices are De Bruijn indices, and the De Bruijn index of `x` is `0`, we extend the environment of bound variables with `x` by prepending it. Once we are done with that, we evaluate the body `e` in the new environment of bound variables.

To be able to see the result of evaluation, we need a mechanism for printing values. Since Haskell cannot derive `Show` for `VLam (Value -> Value)`, we need to implement the mechanism on our own. The approach taken in the paper is to define a conversion function `quote` that takes a value back to a (printable) term:

```{.haskell .numberLines}
quote0 :: Value -> CTerm
quote0 = quote 0

quote :: Int -> Value -> CTerm
quote ii (VLam f) = Lam (quote (ii + 1) (f (vfree (Quote ii))))
quote ii (VNeutral n) = Inf (neutralQuote ii n)

neutralQuote :: Int -> Neutral -> ITerm
neutralQuote ii (NFree x) = boundfree ii x
neutralQuote ii (NApp n v) = neutralQuote ii n :@: quote ii v

boundfree :: Int -> Name -> ITerm
boundfree ii (Quote k) = Bound (ii - k - 1)
boundfree ii x = Free x
```

`quote` takes 1) a natural number (starting from zero) of binders that have been passed so far, and 2) a value to be converted to a term. Again, here is a case-by-case explanation:

 1. `quote ii (VLam f)`: this is where the `Quote` constructor is used: we apply `f` to `vfree (Quote ii)`, recursively `quote` the result of the substitution on a new level `ii + 1`, and wrap it in `Lam`. `Quote ii` is effectively a De Bruijn _level_ [^de-bruijn-level]: if it occurs somewhere in `f (vfree (Quote ii))`, we will convert it to a proper bound variable.
 2. `quote ii (VNeutral n)`: we push the conversion to `neutralQuote`:
    1. `neutralQuote ii (NFree x)`: in `boundfree`, 1) if `x` is a `Quote k` variable (that has been applied to `f` while converting `VLam f`), then we can convert it to a `Bound` variable using the formula `ii - k - 1`. By doing so, we essentially convert a De Bruijn _level_ `k` to a proper De Bruijn _index_. 2) If `x` is any other kind of a variable, we just return it without changes.
    2. `neutralQuote ii (NApp n v)`: nothing interesting here; we just push the conversion to `neutralQuote` and `quote` recursively.

To see how it works, take our lovely K combinator as an example:

 1. `quote 0 (VLam (\x -> VLam (\y -> x)))`
 2. `Lam (quote 1 (VLam (\y -> vfree (Quote 0))))`
 3. `Lam (Lam (quote 2 (vfree (Quote 0))))`
 4. `Lam (Lam (neutralQuote 2 (NFree (Quote 0))))`
 5. `Lam (Lam (Bound 1))`

Finally, let us move on to type checking. The type and data definitions are:

```{.haskell .numberLines}
type Context = [(Name, Info)]

data Info
  = HasKind Kind
  | HasType Type
  deriving (Show)

data Kind = Star
  deriving (Show)

type Result a = Either String a
```

`Context` is a list containing typing information `Info` about every free variable in a term that we want to type-check:

 - `HasKind Star` means that a variable is a type variable.
 - `HasType ty` means that a variable is a term variable of the type `ty`.

`Result` is just the `Either` monad with `String` used as the error type.

The type checking algorithm is defined as follows [^monad-except]:

```{.haskell .numberLines}
cKind :: Context -> Type -> Kind -> Result ()
cKind g (TFree x) Star =
  case lookup x g of
    Just (HasKind Star) -> return ()
    Nothing -> throwError "unknown identifier"
cKind g (Fun kk kk') Star =
  do
    cKind g kk Star
    cKind g kk' Star

iType0 :: Context -> ITerm -> Result Type
iType0 = iType 0

iType :: Int -> Context -> ITerm -> Result Type
iType ii g (Ann e ty) =
  do
    cKind g ty Star
    cType ii g e ty
    return ty
iType ii g (Free x) =
  case lookup x g of
    Just (HasType ty) -> return ty
    Nothing -> throwError "unknown identifier"
iType ii g (e1 :@: e2) =
  do
    si <- iType ii g e1
    case si of
      Fun ty ty' -> do
        cType ii g e2 ty
        return ty'
      _ -> throwError "illegal application"

cType :: Int -> Context -> CTerm -> Type -> Result ()
cType ii g (Inf e) ty =
  do
    ty' <- iType ii g e
    unless (ty == ty') (throwError "type mismatch")
cType ii g (Lam e) (Fun ty ty') =
  cType
    (ii + 1)
    ((Local ii, HasType ty) : g)
    (cSubst 0 (Free (Local ii)) e)
    ty'
cType ii g _ _ =
  throwError "type mismatch"
```

`cKind` checks that a given type is well-formed in a given context. In `cKind g (TFree x) Star`, if we can find the variable `x` in the context `g`, and it is a type variable, then we return `()`. If there is no such variable in the context, we throw an error. The algorithm, for some reason, does not consider the case when the variable exists but is not a type variable, so this function is partial. The second case is uninteresting.

Then two functions follow, `iType` and `cType`. They are the reason why we separated the representation of terms into inferrable and checkable: while certain terms can be inferred (variables, applications, and type annotations), some can only be checked (lambda abstractions and `Inf`) [^bidirectional-tc].

To type-check such a language, we employ _bidirectional type checking_. In detail, we have the function `iType` that _infers_ a type of a term and `cType` that checks a term against a given type. These functions are mutually recursive: whenever `iType` needs to check a term, it calls `cType`, and vice versa.

The cases of `iType` just encode the corresponding rules of a simply typed lambda calculus. This function is partial as well, now for two reasons: 1) the case `iType ii g (Free x)` does not consider the scenario when the variable is present but is not a term variable, and 2) `iType` does not handle `Bound` variables at all.

The only interesting case of `cType` is `cType ii g (Lam e) (Fun ty ty')`. To check a lambda abstraction, we manually substitute all references to the binder with `(Free (Local ii))` -- again, this is a De Bruijn _level_; then we check the body `e` in the context `g` extended with `(Local ii, HasType ty)`. If the binder is referred somewhere in the body `e`, we will encounter it in the case `iType ii g (Free x)`.

Here are the definitions of `iSubst` and `cSubst`:

```{.haskell .numberLines}
iSubst :: Int -> ITerm -> ITerm -> ITerm
iSubst ii r (Ann e ty) = Ann (cSubst ii r e) ty
iSubst ii r (Bound j) = if ii == j then r else Bound j
iSubst ii r (Free y) = Free y
iSubst ii r (e1 :@: e2) = iSubst ii r e1 :@: cSubst ii r e2

cSubst :: Int -> ITerm -> CTerm -> CTerm
cSubst ii r (Inf e) = Inf (iSubst ii r e)
cSubst ii r (Lam e) = Lam (cSubst (ii + 1) r e)
```

The function is pretty boring: all it does is just substituting a selected `Bound` variable for `ITerm`. In fact, we have already seen substitution happening in `iEval`; the difference is that now we do not have the opportunity to mimick substitution with a native Haskell function application, because the syntax of terms is first-order (it does not represent lambda abstractions in terms of Haskell functions).

Let us complete the section with some examples:

```{.haskell .numberLines}
id' = Lam (Inf (Bound 0))
const' = Lam (Lam (Inf (Bound 1)))

tfree a = TFree (Global a)
free x = Inf (Free (Global x))

term1 = Ann id' (Fun (tfree "a") (tfree "a")) :@: free "y"
term2 =
  Ann
    const'
    ( Fun
        (Fun (tfree "b") (tfree "b"))
        ( Fun
            (tfree "a")
            (Fun (tfree "b") (tfree "b"))
        )
    )
    :@: id'
    :@: free "y"

env1 =
  [ (Global "y", HasType (tfree "a")),
    (Global "a", HasKind Star)
  ]
env2 = [(Global "b", HasKind Star)] ++ env1

-- Inf (Free (Global "y"))
test_eval1 = quote0 (iEval term1 ([], []))

-- Lam (Inf (Bound 0))
test_eval2 = quote0 (iEval term2 ([], []))

-- Right (TFree (Global "a"))
test_type1 = iType0 env1 term1

-- Right (Fun (TFree (Global "b")) (TFree (Global "b")))
test_type2 = iType0 env2 term2
```

## The first alternative, higher-order style

"We pick an implementation that allows us to follow the type system closely, and that reduces the amount of technical overhead to a relative minimum, so that we can concentrate on the essence of the algorithms involved." -- I simply cannot agree with this statement from the paper.

Let us enumerate all the naming schemes used in the original implementation:

 - De Bruijn indices `Bound`;
 - De Bruijn levels `Local` for type checking;
 - De Bruijn levels `Quote` for quoting;
 - Named variables `Global`;
 - Higher-Order Abstract Syntax `VLam`.

And yet, with all this machinery at our disposal, we still need to implement substitution manually. This is somewhat sad.

So let us think about how to simplify the implementation. `iSubst` and `cSubst` give us a useful hint: we basically implement substitution _algorithmically_, whereas in the evaluation and quoting code, we can just apply a Haskell function `f` to a (value) argument to achieve similar behaviour. An obvious desire would be to employ the same technique for terms, thereby making them [_higher-order_].

[_higher-order_]: https://cstheory.stackexchange.com/a/20075

Our decision leads to several consequences:

 - Since terms and values will now use the same representation of functions, there is no need to distinguish between them [^term-value].
 - `Bound` variables can be removed from the representation, since Haskell is now in charge of substitution.
 - `quote` can be removed, since we now only have terms.
 - `Quote` variables can be removed, since there is no quoting now.
 - Terms will no longer derive `Show` and `Eq` automatically, since Haskell cannot print functional values. We will have to implement printing manually.
 - `eval` will only take an environment of `Free` variables, since Haskell is now in charge of substitution.
 - `iSubst` and `cSubst` can be removed, since terms are now higher-order.

Also, let us erase the distinction between checkable and inferrable terms. The choice to separate them avoids one `throwError` in the type checker at the expense of requiring us to separate every single function acting on terms into two functions: `iEval` and `cEval`, `iSubst` and `cSubst`, and etc., although these functions do not really care about the distinction. Let us be pragmatic instead of being smart.

The new data definitions are:

```{.haskell .numberLines}
data Term
  = Ann Term Type
  | Free Name
  | Term :@: Term
  | Lam (Term -> Term)

data Name
  = Global String
  | Local Int
  deriving (Eq)

data Type
  = TFree Name
  | Fun Type Type
  deriving (Eq)
```

The key change is that `Lam` is now a Haskell function `Term -> Term`. The `eval` algorithm is now even simpler:

```{.haskell .numberLines}
eval :: Term -> NameEnv Term -> Term
eval (Ann e _) env = eval e env
eval (Free x) env = case lookup x env of Nothing -> Free x; Just v -> v
eval (e1 :@: e2) env = vapp (eval e1 env) (eval e2 env)
eval (Lam f) env = Lam (\x -> eval (f x) env)

vapp :: Term -> Term -> Term
vapp (Lam f) v = f v
vapp e1 e2 = e1 :@: e2
```

Notice how we evaluate `Lam f`: we return a new `Lam` that accepts a value `x` as an argument, invokes native Haskell substitution `f x`, and evaluates the expansion in the same environment of global variables.

The data definitions for type checking are the same. The type checker is now:

```{.haskell .numberLines}
cKind :: Context -> Type -> Kind -> Result ()
cKind g (TFree x) Star =
  case lookup x g of
    Just (HasKind Star) -> return ()
    Nothing -> throwError "unknown identifier"
cKind g (Fun kk kk') Star =
  do
    cKind g kk Star
    cKind g kk' Star

iType0 :: Context -> Term -> Result Type
iType0 = iType 0

iType :: Int -> Context -> Term -> Result Type
iType ii g (Ann e ty) =
  do
    cKind g ty Star
    cType ii g e ty
    return ty
iType _ g (Free x) =
  case lookup x g of
    Just (HasType ty) -> return ty
    Nothing -> throwError "unknown identifier"
iType ii g (e1 :@: e2) =
  do
    si <- iType ii g e1
    case si of
      Fun ty ty' -> do
        cType ii g e2 ty
        return ty'
      _ -> throwError "illegal application"
iType _ _ (Lam _) = throwError "not inferrable"

cType :: Int -> Context -> Term -> Type -> Result ()
cType ii g (Lam f) (Fun ty ty') =
  cType
    (ii + 1)
    ((Local ii, HasType ty) : g)
    (f (Free (Local ii)))
    ty'
cType _ _ (Lam _) _ =
  throwError "expected a function type"
cType ii g e ty =
  do
    ty' <- iType ii g e
    unless (ty == ty') (throwError "type mismatch")
```

The key change here is that instead of calling `cSubst` in `cType`, we just apply `f` to `Free (Local ii)`. This is possible because `f` is now a Haskell function. Also, we can throw the error `"not inferrable"` in `iType` if we are asked to infer `Lam`.

Finally, we need a way to print terms. "As we mentioned earlier, the use of higher-order abstract syntax requires us to define a _quote_ function that takes a `Value` back to a term." -- this is not true. Of course, quoting to a printable term is a way to go, but it is not the only option. Instead, we can derive `Show` for `Term` ourselves:

```{.haskell .numberLines}
instance Show Term where
  show = go 0
    where
      go ii (Ann e ty) = "(" ++ go ii e ++ " : " ++ show ty ++ ")"
      go _ (Free x) = show x
      go ii (e1 :@: e2) = "(" ++ go ii e1 ++ " " ++ go ii e2 ++ ")"
      go ii (Lam f) = "(λ. " ++ go (ii + 1) (f (Free (Local ii))) ++ ")"

instance Show Type where
  show (TFree x) = show x
  show (Fun ty ty') = "(" ++ show ty ++ " -> " ++ show ty' ++ ")"

instance Show Name where
  show (Global x) = x
  show (Local ii) = show ii
```

Let us test our new implementation:

```{.haskell .numberLines}
id' = Lam (\x -> x)
const' = Lam (\x -> Lam (\_ -> x))

tfree a = TFree (Global a)
free x = Free (Global x)

term1 = Ann id' (Fun (tfree "a") (tfree "a")) :@: free "y"
term2 =
  Ann
    const'
    ( Fun
        (Fun (tfree "b") (tfree "b"))
        ( Fun
            (tfree "a")
            (Fun (tfree "b") (tfree "b"))
        )
    )
    :@: id'
    :@: free "y"

env1 =
  [ (Global "y", HasType (tfree "a")),
    (Global "a", HasKind Star)
  ]
env2 = [(Global "b", HasKind Star)] ++ env1

-- (((λ. 0) : (a -> a)) y)
test_id_show = show term1

-- ((((λ. (λ. 0)) : ((b -> b) -> (a -> (b -> b)))) (λ. 0)) y)
test_const_show = show term2

-- y
test_eval1 = show (eval term1 [])

-- (λ. 0)
test_eval2 = show (eval term2 [])

-- Right a
test_type1 = show (iType0 env1 term1)

-- Right (b -> b)
test_type2 = show (iType0 env2 term2)
```

All in all, our new implementation uses only three naming schemes: higher-order `Lam`, `Global` variables, and De Bruijn levels `Local`. We could also remove global variables and use `Local` instead, but this would make the examples less readable.

## The second alternative, first-order style

Perhaps surprisingly, we can use a first-order [^first-order] encoding for both terms and values without implementing substitution by term traversal. The trick is to use De Bruijn _indices_ for terms and De Bruijn _levels_ for values:

```{.haskell .numberLines}
data Term
  = Ann Term Type
  | Bound Int
  | Free Name
  | Term :@: Term
  | Lam Term
  deriving (Show, Eq)

newtype Name = Global String
  deriving (Show, Eq)

data Type
  = TFree Name
  | Fun Type Type
  deriving (Show, Eq)

data Value
  = VLam Env Term
  | VNeutral Neutral
  deriving (Show, Eq)

data Neutral
  = NVar Int
  | NFree Name
  | NApp Neutral Value
  deriving (Show, Eq)

type Env = [Value]

vvar :: Int -> Value
vvar n = VNeutral (NVar n)

vfree :: Name -> Value
vfree n = VNeutral (NFree n)
```

The `Bound` constructor is a De Bruijn index, whereas `NVar` is a De Bruijn level. The `VLam` constructor is called a _closure_: a lambda body `Term` and an environment `Env` bundled together. The `Env` contains values for all unbound variables in `Term`, except for the variable `Bound 0`, which is the lambda binder. Later, when we should apply `VLam` to some argument `v`, we will extend the environment with `v` by prepending it [^closure-env].

Here is the evaluation algorithm:

```{.haskell .numberLines}
type NameEnv v = [(Name, v)]

eval :: Term -> (NameEnv Value, Env) -> Value
eval (Ann e _) d = eval e d
eval (Bound ii) d = snd d !! ii
eval (Free x) d = case lookup x (fst d) of Nothing -> vfree x; Just v -> v
eval (e1 :@: e2) d = vapp (fst d) (eval e1 d) (eval e2 d)
eval (Lam m) d = VLam (snd d) m

vapp :: NameEnv Value -> Value -> Value -> Value
vapp e (VLam d m) v = eval m (e, v : d)
vapp _ (VNeutral n) v = VNeutral (NApp n v)
```

There are two interesting cases:

 1. `eval (Lam m) d`: we do not perform any evaluation here; instead, we just construct a lambda value `VLam` with unevaluated `m`. This is in contrast with the two previous implementations, where we called `eval` under a `VLam` binder.
 2. `vapp e (VLam d m) v`: we prepend `d` with `v` and continue evaluating `m`; since `d` must contain values for all free variables of `m` except `Bound 0`, `v : d` will contain values for _all_ free variables in `m`, including `Bound 0`.

Since we have the term-value separation again, we can implement `quote` [^quote-first-order]:

```{.haskell .numberLines}
quote0 :: Value -> Term
quote0 = quote [] 0

quote :: NameEnv Value -> Int -> Value -> Term
quote e ii (VLam d m) = Lam (quote e (ii + 1) (eval m (e, vvar ii : d)))
quote e ii (VNeutral n) = neutralQuote e ii n

neutralQuote :: NameEnv Value -> Int -> Neutral -> Term
neutralQuote _ ii (NVar i) = Bound (ii - i - 1)
neutralQuote _ _ (NFree x) = Free x
neutralQuote e ii (NApp n v) = neutralQuote e ii n :@: quote e ii v
```

To quote `VLam d m`, we first evaluate `m` in the environment `d` extended with `vvar ii`, which is a neutral variable corresponding to the current De Bruijn level `ii`. Here, `vvar ii` stands as an argument whose value is unknown. If, during the evaluation of `m`, we should see that it refers to `Bound 0`, we will replace `Bound 0` with `vvar ii` we have just added to the environment. We then continue with quoting the evaluated `m` under the next level `ii + 1`. Finally, we wrap the result in `Lam`.

The data definitions for type checking are:

```{.haskell .numberLines}
type GlobalContext = [(Name, Info)]

type Context = [Type]

data Info
  = HasKind Kind
  | HasType Type
  deriving (Show)

data Kind = Star
  deriving (Show)

type Result a = Either String a
```

In addition to `Info`, `Kind`, and `Result` we have already seen, we introduce `GlobalContext` and `Context`. `GlobalContext` is a list associating names of global variables to their `Info`, whereas `Context` is a De Bruijn-indexed list of types of bound variables. `Context` can only associate _term_ variables to their types; it cannot contain any information about _type_ variables, since we are only dealing with [simple types].

[simple types]: https://en.wikipedia.org/wiki/Simply_typed_lambda_calculus

The type checking algorithm is:

```{.haskell .numberLines}
cKind :: (GlobalContext, Context) -> Type -> Kind -> Result ()
cKind g (TFree x) Star =
  case lookup x (fst g) of
    Just (HasKind Star) -> return ()
    Nothing -> throwError "unknown identifier"
cKind g (Fun kk kk') Star =
  do
    cKind g kk Star
    cKind g kk' Star

iType0 :: GlobalContext -> Term -> Result Type
iType0 g = iType 0 (g, [])

iType :: Int -> (GlobalContext, Context) -> Term -> Result Type
iType ii g (Ann e ty) =
  do
    cKind g ty Star
    cType ii g e ty
    return ty
iType _ g (Bound i) = return (snd g !! i)
iType _ g (Free x) =
  case lookup x (fst g) of
    Just (HasType ty) -> return ty
    Nothing -> throwError "unknown identifier"
iType ii g (e1 :@: e2) =
  do
    si <- iType ii g e1
    case si of
      Fun ty ty' -> do
        cType ii g e2 ty
        return ty'
      _ -> throwError "illegal application"
iType _ _ (Lam _) = throwError "not inferrable"

cType :: Int -> (GlobalContext, Context) -> Term -> Type -> Result ()
cType ii g (Lam m) (Fun ty ty') =
  cType (ii + 1) (fst g, ty : snd g) m ty'
cType _ _ (Lam _) _ =
  throwError "expected a function type"
cType ii g e ty =
  do
    ty' <- iType ii g e
    unless (ty == ty') (throwError "type mismatch")
```

The interesting cases are:

 1. `iType _ g (Bound i)`: we index (`!!`) the context of bound variables `snd g` with `i`. As a result, we obtain the type of `Bound i`.
 2. `cType ii g (Lam m) (Fun ty ty')`: we check the lambda body `m` against `ty'` in an extended context of bound variables `ty : snd g`. The old context `snd g` is extended with `ty` because `Bound 0` must have the type `ty`.

As usual, let us test our implementation:

```{.haskell .numberLines}
id' = Lam (Bound 0)
const' = Lam (Lam (Bound 1))

tfree a = TFree (Global a)
free x = Free (Global x)

term1 = Ann id' (Fun (tfree "a") (tfree "a")) :@: free "y"
term2 =
  Ann
    const'
    ( Fun
        (Fun (tfree "b") (tfree "b"))
        ( Fun
            (tfree "a")
            (Fun (tfree "b") (tfree "b"))
        )
    )
    :@: id'
    :@: free "y"

env1 =
  [ (Global "y", HasType (tfree "a")),
    (Global "a", HasKind Star)
  ]
env2 = [(Global "b", HasKind Star)] ++ env1

-- Free (Global "y")
test_eval1 = quote0 (eval term1 ([], []))

-- Lam (Bound 0)
test_eval2 = quote0 (eval term2 ([], []))

-- Right (TFree (Global "a"))
test_type1 = iType0 env1 term1

-- Right (Fun (TFree (Global "b")) (TFree (Global "b")))
test_type2 = iType0 env2 term2
```

The approach we have taken here is called _Normalization by Evaluation_, abbreviated as _NbE_. Over time, it has become a standard way of implementing dependent type theories, as it is both reasonably efficient and easy to implement. An interested reader can find more information from these sources:

 - [AndrasKovacs/elaboration-zoo](https://github.com/AndrasKovacs/elaboration-zoo) by András Kovács.
 - ["Checking Dependent Types with Normalization by Evaluation: A Tutorial (Haskell Version)"](https://davidchristiansen.dk/tutorials/implementing-types-hs.pdf) by David Thrane Christiansen.

## Final words

We have not touched dependent types -- all our discussion was limited to simply typed lambda calculus. Although dependent types are a bit more interesting with respect to type checking, the general idea behind naming schemes remains the same.

In both approaches I have demonstrated, we could get rid of global variables, but I have not done that for the sake of readability of the examples. In the first approach, the essential naming schemes are De Bruijn indices (terms) and higher-order lambda abstractions (values), whereas in the second approach, the essential naming schemes are De Bruijn indices (terms) and De Bruijn levels (values).

Both approaches are reasonably simple. The downside of the first approach is that we had to implement printing manually (although it was not a big deal), whereas the usage examples of the second approach looked a tad less readable due to De Bruijn indices. The higher-order approach is typically favoured by eDSL designers, whereas the first-order approach is predominantly used as an internal representation of a compiler/interpreter. Since we can transform named variables to De Bruijn indices automatically, the end user should not notice any difference.

If you feel confused about anything, feel free to ask in the comments.

## References

[^sources]: The language sources are taken from [the official implementation].

[the official implementation]: https://www.andres-loeh.de/LambdaPi/LambdaPi.hs

[^de-bruijn-level]: Contrary to a De Bruijn index, a De Bruijn level is a natural number indicating the number of binders between the variable's binder and the term's _root_. For example, the term `\x -> \y -> x` is represented as `\_ -> \_ -> 0`, where `0` is the De Bruijn level of `x`. The De Bruijn level of `y` would be `1` if we used it instead of `x`.

[^monad-except]: The code requires `import Control.Monad.Except`.

[^bidirectional-tc]: The sole reason for this separation is that in our language, the syntax of binders does not possess typing information: for example, we cannot tell algorithmically whether `\x -> x` is a function from integers to integers, from strings to strings, and etc. -- there are infinitely many possibilities. By annotating functions, we can always tell their exact types. A more proper introduction to bidirectional type checking can be found in ["Bidirectional Typing Rules: A Tutorial"] by David Raymond Christiansen.

["Bidirectional Typing Rules: A Tutorial"]: https://davidchristiansen.dk/tutorials/bidirectional.pdf

[^term-value]: Separating terms and values can facilitate type safety, but our language is so small that this does not make any sense.

[^first-order]: An encoding is first-order iff it does not contain Haskell functions.

[^closure-env]: That being said, the _i_-th value of `Env` corresponds to the _i+1_-th variable in the lambda body. Prepending an argument to `Env` works because indices of all the previous values of `Env` will be "incremented" automatically!

[^quote-first-order]: Strictly speaking, printing does not require `quote` if values are already first-order. However, our new `quote` is algorithmically different from the previous ones: it pushes evaluation under a binder, essentially serving as a beta-normalizer. Thus, if we want to print a fully normalized term, we still need to implement `quote`.
