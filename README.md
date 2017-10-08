# nakedret

nakedret is a Go static analysis tool to find naked returns in functions greater than a specified function length.

# Installation

    go get -u github.com/alexkohler/nakedret

# Usage

Similar to other Go static anaylsis tools, nakedret can be invoked with one or more filenames, directories, or packages named by its import path.

nakedret [flags] packages/files

Currently, the only flag supported is -l, which is an optional flag to specify the maximum length a function can be (in terms of line length). If not specified, it defaults to 5.

# Purpose
As noted in Go's (Code Review comments)[https://github.com/golang/go/wiki/CodeReviewComments#named-result-parameters]:

> Naked returns are okay if the function is a handful of lines. Once it's a medium sized function, be explicit with your return > values. Corollary: it's not worth it to name result parameters just because it enables you to use naked returns. Clarity of  > docs is always more important than saving a line or two in your function.

This tool aims to catch naked returns on non-trivial functions.
