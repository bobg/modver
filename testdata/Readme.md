Files in this testdata tree are Go text templates,
each producing an “older” version of a Go package and a “newer” version.
The `runtest` function in `modver_test.go` compares the resulting two packages.

Files in the `major` subdir are expected to produce a `Major` result when older and newer are compared.
Files in the `minor`, `patchlevel`, and `none` subdirs
are expected to produce `Minor`, `Patchlevel`, and `None` results.

Each `.tmpl` file defines a number of named templates using the `{{ define "name" }}` construct.
The name is the relative path of a file to create for testing.
“Older” package files go in the subdir `older`.
“Newer” package files go in the subdir `newer`.

If the path does not end in a filename
(that is, a path element containing a `.`),
then the name `x.go` is assumed.

If the path is _only_ a filename and no subdir part,
then the file is copied to both `older` and `newer` subdirs.

If no template with the name `go.mod` is seen,
a `go.mod` file will be synthesized in `older` and `newer`.

When copying template contents to their designated files,
any lines with the prefix `//// `
(four slashes and a space)
will first have that prefix removed.
