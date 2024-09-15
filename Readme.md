# Modver

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/modver/v2.svg)](https://pkg.go.dev/github.com/bobg/modver/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/modver/v2)](https://goreportcard.com/report/github.com/bobg/modver/v2)
[![Tests](https://github.com/bobg/modver/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/modver/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/modver/badge.svg?branch=master)](https://coveralls.io/github/bobg/modver?branch=master)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

This is modver,
a tool that helps you obey [semantic versioning rules](https://semver.org/) in your Go module.

It can read and compare two different versions of the same module,
from two different directories,
or two different Git commits,
or the base and head of a Git pull request.
It then reports whether the changes require an increase in the major-version number,
the minor-version number,
or the patchlevel.

## Installation and usage

Modver can be used from the command line,
or in your Go program,
or with [GitHub Actions](https://github.com/features/actions).

### Command-line interface

Install the `modver` command like this:

```sh
go install github.com/bobg/modver/v2/cmd/modver@latest
```

Assuming the current directory is the root of a cloned Git repository,
you can run it like this:

```sh
$ modver -git .git HEAD~1 HEAD
```

to tell what kind of version-number change is needed for the latest commit.
The `-git .git` gives the path to the repository’s info;
it can also be something like `https://github.com/bobg/modver`.
The arguments `HEAD~1` and `HEAD` specify two Git revisions to compare;
in this case, the latest two commits on the current branch.
These could also be tags or commit hashes.

### GitHub Action

You can arrange for Modver to inspect the changes on your pull-request branch
as part of a GitHub Actions-based continuous-integration step.
It will add a comment to the pull request with its findings,
and will update the comment as new commits are pushed to the branch.

To do this, you’ll need a directory in your GitHub repository named `.github/workflows`,
and a Yaml file containing (at least) the following:

```yaml
name: Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.19

      - name: Modver
        if: ${{ github.event_name == 'pull_request' }}
        uses: bobg/modver@v2.5.0
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          pull_request_url: https://github.com/${{ github.repository }}/pull/${{ github.event.number }}
```

This can be combined with other steps that run unit tests, etc.
You can change `Tests` to whatever name you like,
and should change `main` to the name of your repository’s default branch.
If your pull request is on a GitHub server other than `github.com`,
change the hostname in the `pull_request_url` parameter to match.

Note the `fetch-depth: 0` parameter for the `Checkout` step.
This causes GitHub Actions to create a clone of your repo with its full history,
as opposed to the default,
which is a shallow clone.
Modver requires enough history to be present in the clone
for it to access the “base” and “head” revisions of your pull-request branch.

For more information about configuring GitHub Actions,
see [the GitHub Actions documentation](https://docs.github.com/actions).

### Go library

Modver also has a simple API for use from within Go programs.
Add it to your project with `go get github.com/bobg/modver/v2@latest`.
See [the Go doc page](https://pkg.go.dev/github.com/bobg/modver/v2) for information about how to use it.

## Semantic versioning

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
The actual change required may be greater.
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

The `modver` command
(in the `cmd/modver` subdirectory)
can be used,
among other ways,
to test that each commit to a Git repository increments the module’s version number appropriately.
This is done for modver itself using GitHub Actions,
[here](https://github.com/bobg/modver/blob/dd93eccb5674b13161a91bf6a6666889c21adb5b/.github/workflows/go.yml#L25-L26).

(Note that the standard `actions/checkout@v2` action,
for cloning a repository during GitHub Actions,
creates a shallow clone with just one commit’s worth of history.
For the usage here to work,
you’ll need more history:
at least two commit’s worth and maybe more to pull in the latest tag for the previous revision.
The clone depth can be overridden with the `fetch-depth` parameter,
which modver does [here](https://github.com/bobg/modver/blob/dd93eccb5674b13161a91bf6a6666889c21adb5b/.github/workflows/go.yml#L14-L15).)
