PyParsing -- A Parsing Module, in Go
====================================

|Build Status| |License|

Introduction
============

The pyparsing module is an alternative approach to creating and
executing simple grammars, vs. the traditional lex/yacc approach, or the
use of regular expressions. The pyparsing module provides a library of
elements that client code uses to construct the grammar directly in
code.

*[Since first writing this description of pyparsing in late 2003, this
technique for developing parsers has become more widespread, under the
name Parsing Expression Grammars - PEGs. See more information on PEGs*
`here <https://en.wikipedia.org/wiki/Parsing_expression_grammar>`__
*.]*

This repository is the Go port of pyparsing: the Go module
``github.com/pyparsing/pyparsing-go`` (package ``pyparsing``), written in
Go with no dependencies outside the standard library. It reproduces the
behaviour of pyparsing 3.3.3 under CPython 3.12, including the parts the
Python library inherits from its language: the ``re`` module's pattern
syntax and matching, Python's string semantics (positions count code
points), and the ``repr()`` of the values in parse results.

Here is a program to parse ``"Hello, World!"`` (or any greeting of the form
``"salutation, addressee!"``):

.. code:: go

    package main

    import (
        "fmt"

        pp "github.com/pyparsing/pyparsing-go"
    )

    func main() {
        word, _ := pp.Word(pp.Alphas, nil, 1, 0, 0, false, nil)
        greet := word.Add(",").Add(word).Add("!")
        hello := "Hello, World!"
        result, err := greet.ParseString(hello, false)
        if err != nil {
            fmt.Println(err)
            return
        }
        fmt.Println(hello, "->", result)
    }

The program outputs the following::

    Hello, World! -> ['Hello', ',', 'World', '!']

Python's operators become methods: ``+`` is ``Add``, ``|`` is ``BitOr``,
``^`` is ``BitXor``, ``&`` is ``BitAnd``, ``-`` is ``Sub``, ``~`` is
``Invert``, ``*`` is ``Mul``, ``[]`` is ``GetItem`` and ``<<=`` is
``ILShift``. Each element constructor takes the arguments of the Python
class in the same order.

The parsed results returned from ``ParseString()`` are a ``*ParseResults``,
which can be accessed as a nested list, a dictionary, or by results name.

The pyparsing module handles some of the problems that are typically
vexing when writing text parsers:

- extra or missing whitespace (the above program will also handle ``"Hello,World!"``, ``"Hello , World !"``, etc.)
- quoted strings
- embedded comments

Documentation
=============

The Go API is documented in the package's doc comments (``go doc
github.com/pyparsing/pyparsing-go``). The concepts, element classes and
results behave as described in the pyparsing documentation in ``docs/``
and the `online docs <https://pyparsing-docs.readthedocs.io/en/latest/>`__.

Building and testing
====================

::

    go build ./...
    go vet ./...
    go test ./...

License
=======

MIT License. See the `LICENSE <LICENSE>`__ file.

History
=======

See `CHANGES <https://github.com/pyparsing/pyparsing/blob/master/CHANGES>`__ file.


.. |Build Status| image:: https://github.com/pyparsing/pyparsing/actions/workflows/ci.yml/badge.svg
   :target: https://github.com/pyparsing/pyparsing/actions/workflows/ci.yml

.. |License| image:: https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square
    :target: LICENSE
    :alt: License
