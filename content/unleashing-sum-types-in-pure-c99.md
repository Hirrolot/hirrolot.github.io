---
title: "Unleashing Sum Types in Pure C99"
author: Hirrolot's Blog
date: Feb 6, 2021
---

Some time ago, I was writing a [TIFF] encoder. A TIFF file consists of a header and a sequence of TIFF entries. To represent a TIFF entry, I wrote this:

[TIFF]: https://en.wikipedia.org/wiki/Tagged_Image_File_Format

```c
typedef struct {
    enum {
        TIFFEntryValueTagSmall,
        TIFFEntryValueTagBlock,
    } tag;
    union {
        uint32_t small_value;
        struct {
            size_t size;
            const void *data;
        } block;
    };
} TIFFEntryValue;

inline static TIFFEntryValue TIFFEntryValueSmall(uint32_t value) {
    return (TIFFEntryValue){
        .tag = TIFFEntryValueTagSmall,
        .small_value = value,
    };
}

inline static TIFFEntryValue TIFFEntryValueBlock(size_t size, const void *ptr) {
    return (TIFFEntryValue){
        .tag = TIFFEntryValueTagBlock,
        .block = {.data = ptr, .size = size},
    };
}
```

Well, clearly not the best code I ever wrote. The pattern here is called a [tagged union] -- a structure consisting of a tag and a payload. Here, the tag is the enumeration of `TIFFEntryValueTagSmall` and `TIFFEntryValueTagBlock`, both of which correspond to `small_value` and `block`, respectively: if `tag` is `TIFFEntryValueTagSmall`, then `small_value` must be present, and if `TIFFEntryValueTagBlock`, then block must be present.

[tagged union]: https://medium.com/r/?url=https%3A%2F%2Fen.wikipedia.org%2Fwiki%2FTagged_union

Below the structure itself, the two functions are called _value constructors_ -- they construct values of `TIFFEntryValue`.

To match a TIFF value, a switch statement is used:

```c
switch (entry.tag) {
    case TIFFEntryValueTagSmall:
        // Work with entry.small_value...
        break;
    case TIFFEntryValueTagBlock:
        // Work with entry.block...
        break;
}
```

Bad news: when matching `TIFFEntryValueTagSmall`, we can still access `entry.block` -- the same holds for `TIFFEntryValueTagBlock`. Even more, writing value constructors is tedious, and it is not surprising that someone will skip them and later construct something like `{ .tag = TIFFEntryValueTagBlock, .small_value = 42 }`.

Hopefully, our problem has been solved a long time ago; the solution is called _sum types_. Put it simple, a sum type encodes alternative data representations and allows to work with them in a safe, convenient manner. For example, our `TIFFEntryValue` would look like this:

```c
datatype(
    TIFFEntryValue,
    (TIFFEntryValueSmall, uint32_t),
    (TIFFEntryValueBlock, size_t, const void *)
);
```

To construct a TIFF value, write `TIFFEntryValueSmall(42)` or `TIFFEntryValueBlock(3 + 1, "abc")`. To match a TIFF value, write

```c
match(entry) {
    of(TIFFEntryValueSmall, x) {
        // Work with uint32_t x...
    }
    of(TIFFEntryValueBlock, size, data) {
        // Work with size_t size and const void *data...
    }
}
```

There are two differences between the first and the last versions:

 1. Significantly less boilerplate with sum types.
 2. Significantly safer manipulation with sum types: normally we cannot access a block data when we match a small value and vice versa.

Believe or not, exactly the same syntax sugar is possible in pure, standard-conforming C99! The key is my new library, [Datatype99]. I have released it after approximately a year of experimentation with different approaches and underlying metaprogramming libraries. It features a clean, type-safe interface, as well as formally defined code generation semantics, thus allowing to write an FFI for libraries exposing sum types in their interface.

[Datatype99]: https://github.com/Hirrolot/datatype99

The installation instructions are laid out in [README.md]. Now, I am going to present a few more examples of sum types, first of which is a traversal of a binary tree:

[README.md]: https://github.com/Hirrolot/datatype99#installation

[[`examples/binary_tree.c`](https://github.com/Hirrolot/datatype99/blob/master/examples/binary_tree.c)]
```c
#include <datatype99.h>

#include <stdio.h>

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

#define TREE(tree)                ((BinaryTree *)(BinaryTree[]){tree})
#define NODE(left, number, right) TREE(Node(left, number, right))
#define LEAF(number)              TREE(Leaf(number))

int main(void) {
    const BinaryTree *tree = NODE(NODE(LEAF(1), 2, NODE(LEAF(3), 4, LEAF(5))), 6, LEAF(7));

    printf("%d\n", sum(tree));
}
```

<details>
  <summary>Output</summary>

```
28
```

</details>

Here, `BinaryTree` is a sum type with two alternatives: `Leaf` and `Node`. It is recursively traversed inside `sum`.

Sum types are also highly applicable in compiler/interpreter development. Consider this encoding of an [abstract syntax tree]:

[abstract syntax tree]: https://en.wikipedia.org/wiki/Abstract_syntax_tree

[[`examples/ast.c`](https://github.com/Hirrolot/datatype99/blob/master/examples/ast.c)]
```c
#include <datatype99.h>

#include <stdio.h>

datatype(
    Expr,
    (Const, double),
    (Add, Expr *, Expr *),
    (Sub, Expr *, Expr *),
    (Mul, Expr *, Expr *),
    (Div, Expr *, Expr *)
);

double eval(const Expr *expr) {
    match(*expr) {
        of(Const, number) return *number;
        of(Add, lhs, rhs) return eval(*lhs) + eval(*rhs);
        of(Sub, lhs, rhs) return eval(*lhs) - eval(*rhs);
        of(Mul, lhs, rhs) return eval(*lhs) * eval(*rhs);
        of(Div, lhs, rhs) return eval(*lhs) / eval(*rhs);
    }
}

#define EXPR(expr)       ((Expr *)(Expr[]){expr})
#define OP(lhs, op, rhs) op(EXPR(lhs), EXPR(rhs))

int main(void) {
    // 53 + ((155 / 5) - 113)
    Expr expr = OP(Const(53), Add, OP(OP(Const(155), Div, Const(5)), Sub, Const(113)));

    printf("%f\n", eval(&expr));
}
```

<details>
  <summary>Output</summary>

```
-29.000000
```

</details>

Our little language consists of constants of `double` and compound expressions derived from either +, -, *, or /. The interpreter, also called an evaluator, evaluates the language and returns `double`.

Yet another example is a representation of tokens:

[[`examples/token.c`](https://github.com/Hirrolot/datatype99/blob/master/examples/token.c)]
```c
#include <datatype99.h>

#include <stdio.h>

datatype(
    Token,
    (Ident, const char *),
    (Int, int),
    (LParen),
    (RParen),
    (Plus)
);

void print_token(Token token) {
    match(token) {
        of(Ident, ident) printf("%s", *ident);
        of(Int, x) printf("%d", *x);
        of(LParen) printf("(");
        of(RParen) printf(")");
        of(Plus) printf(" + ");
    }
}

int main(void) {
    Token tokens[] = {
        LParen(),
        Ident("x"),
        Plus(),
        Int(123),
        RParen(),
    };

    for (size_t i = 0; i < sizeof(tokens) / sizeof(tokens[0]); i++) {
        print_token(tokens[i]);
    }

    puts("");
}
```

<details>
  <summary>Output</summary>

```
(x + 123)
```

</details>


I hope that now, the usage and syntax of sum types is perfectly clear, as well as the rationale behind them. The next article will be dedicated to zero-cost, convenient error handling using sum types.

## Links

 - [Datatype99 installation instructions](https://github.com/Hirrolot/datatype99#installation)
 - [The original post](https://hirrolot.medium.com/unleashing-sum-types-in-pure-c99-31544302d2ba)
