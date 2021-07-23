# Modver

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/modver.svg)](https://pkg.go.dev/github.com/bobg/modver)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/modver)](https://goreportcard.com/report/github.com/bobg/modver)

This is modver,
a Go package and command that helps you obey [semantic versioning rules](https://semver.org/) in your Go module.

It can read and compare two different versions of the same module,
from two different directories,
or two different Git commits.
It then reports whether the changes require an increase in the major-version number,
the minor-version number,
or the patchlevel.

Briefly, a major-version bump is needed for incompatible changes in the public API,
such as when a type is removed or renamed,
or parameters or results are added to or removed from a function.
Old callers cannot expect to use the new version without being updated.

A minor-version bump is needed when new features are added to the public API,
like a new entrypoint or new fields in an existing struct.
Old callers _can_ continue using the new version without being updated,
but callers depending on the new features cannot use the old version.

A patchlevel bump is needed for most other changes.

The result produced by modver is the _minimal_ change required.
The acquire change required may be greater.
For example,
if a new method is added to a type,
this function will return `Minor`.
However, if something also changed about an existing method that breaks the old contract -
it accepts a narrower range of inputs, for example,
or returns errors in some new cases -
that may well require a major-version bump,
and this function can't detect those cases.

You can be assured, however,
that if this function returns `Major`,
a minor-version bump won't suffice,
and if this function returns `Minor`,
a patchlevel bump won't suffice,
etc.
