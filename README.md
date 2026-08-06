# gogh

Go code generation library. The name `gogh` comes from both `GO Generator` and from the fact I adore Van Gogh
writings.

**LLM-friendly.** An [LLM_CONTEXT.md](LLM_CONTEXT.md) — a condensed API reference — lives in the
repo to drop into an assistant's context. See [Using gogh with an LLM](#using-gogh-with-an-llm).

**A paradigm shift.** I used to write one big generator that spat out an entire service structure.
The current paradigm is different: keep a context for the LLM that describes the structure, and a
tool of small, domain-specific code generators the LLM reaches for point by point. The LLM carries
the shape; the generators carry the exactness.

# Installation

```shell script
go get github.com/sirkon/gogh
```

# Simple usage

```go
package main

import (
	"github.com/sirkon/errors"
	"github.com/sirkon/gogh"
	"github.com/sirkon/message"
)

func main() {
	prj, err := gogh.New(
		gogh.GoFmt,
		func(r *gogh.Imports) *gogh.Imports {
			return r
		},
	)
	if err != nil {
		message.Fatal(errors.Wrap(err, "setup module info"))
	}

	pkg, err := prj.Root("project")
	if err != nil {
		message.Fatal(errors.Wrap(err, "setup package "+prj.Name()))
	}

	r := pkg.Go("main.go", gogh.Shy)

	r.Imports().Add("fmt").Ref("fmt")

	r.L(`func main() {`)
	r.L(`    $fmt.Println("Hello $0!")`, "World")
	r.L(`}`)

	if err := prj.Render(); err != nil {
		message.Fatal(errors.Wrap(err, "render module"))
	}
}
```

# Importers

It would be great to have shortcuts for frequently imported packages besides generic

```go
r.Imports().Add("<pkg path>")
```

isn't it?

Luckily, it is possible and pretty easy since Go supports generics now. All you need is to define your custom type
satisfying `gogh.Importer` interface

```go
// Importer an abstraction covert Imports
type Importer interface {
    Imports() *Imports
    Add(pkgpath string) *ImportAliasControl
    Module(relpath string) *ImportAliasControl
}
```

Something like this will work:

```go
package pkg

import "github.com/sirkon/gogh"

func NewCustomImporter(i *gogh.Imports) *CustomImporter {
	return &CustomImporter{
		i: i,
	}
}

type CustomImporter struct {
	i *gogh.Imports
}

func (i *CustomImporter) Imports() *gogh.Imports {
	return i.i
}

func (i *CustomImporter) Add(pkgpath string) *gogh.ImportAliasControl {
	return i.i.Add(pkgpath)
}

func (i *CustomImporter) Module(pkgpath string) *gogh.ImportAliasControl {
	return i.i.Module(pkgpath)
}

func (i *CustomImporter) Company(relpath string) *gogh.ImportAliasControl {
	return i.i.Add("company.org/gopkgs/" + relpath)
}
```

And then just

```go
mod, err := gogh.New(gogh.GoFmt, NewCustomImporter)
…

r.Imports().Company("configs").Ref("configs")
r.L(`// Config service $0 config definition`, serviceName)
r.L(`type Config struct{`)
r.L(`    TLS *$configs.TLS`)
r.L(`    Service *$configs.Service`)
r.L(`}`)
```

# How to use text renderer.

| Method                 | Description                                                                                                                              |
|------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| `L(format, params...)` | Render and put text line using custom format. <br/>See [further](#formatting) for details.                                               |
| `C(params...)`         | Render a text concatenation of given parameters.                                                                                         |
| `R(text)`              | Put raw text                                                                                                                             |
| `N()`                  | Put new line                                                                                                                             |
| `S(format, params...)` | Same as `L` but returns rendered text as a string without saving it.                                                                     |
| `Z()`                  | Returns a new renderer which will put lines before<br/>any line made by the original renderer.<br/> Set details below.                   |
| `T()`                  | Returns a new "temporary" renderer which belong to<br/>the same package but will not produce<br/>any new file.                           |
| `F(…)`                 | Renders definition of a function. The primary goal is to simplify building functions<br/>definitions based on existing signatures.       |
| `M(…)`                 | Similar to `F` but for methods this time.                                                                                                |
| `Type(t)`              | Renders fully qualified type name  of `types.Type` instance.<br/>Will take care of package qualifier names and imports.                  |
| `Proto(t)`             | Renders fully qualified type name defined in [protoast](https://github.com/sirkon/protoast/tree/master/ast).                             |                                                                                                
| `Uniq(name, hints)`    | Returns unique name using value of name as a basis. <br/>See further details below.                                                      |
| `Taken(name)`)         | Checks if this name was taken before.                                                                                                    |                                                                                                                               |
| `Let(name, value)`     | Sets immutable variable into the rendering context.<br/>Can be addressed in format strings further.<br/>See details below.               |
| `TryLet(name, value)`  | Same as let but won't panic if the name was taken before.                                                                                |
| `Scope()`              | Produce a new renderer with its local context.<br/>`Uniq` and `*Let` calls will not touch the original renderer.<br/> See details below. |
| `InnerScope(func)`     | Produce a new scope and feed it to the given function.                                                                                   |

## Formatting.

The formatting is built upon the [go-format](https://github.com/sirkon/go-format) library, but there is some extra
functionality.

- `types.Type` and `ast.Type` are supported out of the box and converted into strings automatically.
- `(*)Commas` and `(*)Params` are also supported with their custom format option `\n`, which will render
  their multiline representation.

And then `string` (and `fmt.Stringer`) arguments have these dedicated formatting options:

| format option | details                                      |
|---------------|----------------------------------------------|
| `P`           | Applies `gogh.Public` function to the value. |
| `p`           | Applies `gogh.Private` function.             |
| `R`           | Applies `gogh.Proto` function.               |
| `_`           | Applies `gogh.Underscored` function.         |
| `-`           | Applies `gogh.Striked` function.             |


## Lazy generation.

Imagine you have a list of `[{ name, typeName }]` and want to generate:

1. Structured type having respective fields.
2. Constructor of this type.
3. Both in just one pass over that list.

This will work:

```go
r.L(`type Data struct {`)
s := r.Z() // s is for structure generation
r.L(`}`)

r.N()
r.L(`func dataConstructor(`)
a := r.Z() // a for constructor arguements generation
r.L(`) *Data {`)
r.L(`    return &Data{`)
c := r.Z() // c for fields assigns
r.L(`    }`)
r.L(`}`)


for _, item := range typesTypeNamesList {
	s.L(`$0 $1`, item.name, item.typeName)
	a.L(`$0 $1,`, item.name, item.typeName)
	c.L(`$0: $1,`, item.name)
}
```

## Scope.

Every renderer has a scope which can be used to generate unique values and keep rendering context values.
Different renderers can share the same scope though: `r.Z()` call produces a new renderer but its scope is
identical to one `r` has.

`r.Scope()` called in a moment of time `t` produces a new renderer with a new scope, which:

* Has the same set of uniqs registered. So their consecutive `Uniq` calls with same names and hints will
  have the same output.
* Has identical rendering context, so all variables available at the moment of time `t` for the original renderer
  will be avaiable for the new one too.
* Scopes splits after this, meaning new uniqs and context values made for one renderer will not reflect into the 
  another.
* Yet, imports with `Ref` made with one of renderers will reflect into all others rendering on the same file. 
  This is a reasonable decision as package imports are global for a given Go file and all renderers produced
  with `Z` or `Scope` belong to the same file.
  
## Unique scope values.

Let we have to ensure unique values. For, to say, function arguments. `Uniq` method is to help us here.
How it works:

* There's a base name.
* There'is optional hint suffix. It is defined as a vararg, but only the first one can be taken into account.

It tries:

1. Just a base name first. Return if it was not taken.
2. Base name is busy. It tries `<baseName><Hint suffix>` if there's a hint.
3. If both base name and even a hinted base name are busy it looks for the first unique `<base>N` for N = 1, 2, 3, …
   which have not been taken yet.

## Scope rendering context.

Using positional values for formatting can be annoying. You can push some constant values into the so-called
scope rendering context. Example:

```go
r.Let("val", someReallyAnnoyingVariableName)
r.L(`$fmt.Println($val)`)
```

`Let` panics if you tries to define a new value for the variable you have added already.

# Advices.

* Use `Ref` to assign rendering context value is the preferable way to access imported packages:
  `*gogh.GoRenderer` will take care of conflicting names, aliases, etc. Just make sure reference name is unique for the
  renderer.
* Use type aliases if your function calls have renderers in their arguments. Because it is awkward to have something
  like
  ```go 
  func (g *Generator) renderSomething(r *gogh.GoRenderer[*gogh.Imports]) {…}
  ```
  Just put
  ```go
  type goRenderer = gogh.GoRenderer[*gogh.Imports]
  ```
  somewhere and then you will have
  ```go
  func (g *Generator) renderSomething(r *goRenderer) {…}
  ```
* You can use `M` or `F` methods to copy signatures of existing functions in an easy way.

# Rationale.

The line-by-line model was chosen as an overall superior approach to code generation. The two
metrics that drive every design decision in gogh are **composability** and **discoverability**.

## Composability.

This is where templates fail miserably. A template fragment is opaque text — you cannot tell one
template "render your struct fields into the middle of my constructor" without re-threading all the
data through both. Templates compose by string concatenation, which collapses the moment nesting or
ordering becomes non-trivial.

AST/tree builders (jennifer, `go/types`/`ast` construction) are the opposite case: they compose
perfectly. They are object graphs you assemble by calling methods and nesting nodes, which is as
composable as it gets. Their problem is not composition — it is the next metric.

The line-by-line model aims to keep that composability while not sacrificing discoverability. A
sub-generator simply takes a `*GoRenderer[T]` and writes lines; composition is just function calls
sharing a write surface. `Z()` makes even *interleaved* composition work: one pass over a list can
fill a struct definition, its constructor arguments, and its field assignments in a single loop,
because each `Z()` opens a write cursor positioned before the caller's next write.

## Discoverability.

Discoverability is the ability to answer "which codegen line caused this line in the output?". It
is the discriminator between approaches, and it is where templates and AST/tree builders both
degrade — for different reasons.

* **Templates** are good at discoverability only at zero composition. With no composition, an output
  line sits right there in the template. Add `{{define}}`/`{{template}}` composition and that same
  output line becomes the product of a template plus a caller plus a pipeline, scattered across
  files, with indentation and whitespace emerging from their interaction rather than living anywhere
  single. The output line no longer exists at one source location.
* **AST/tree builders** compose perfectly but fail discoverability: they build a structure that a
  printer then reformats, reorders, and reflows. A printed `ast.File` bears little spatial
  resemblance to the calls that built it — the printer is a lossy, position-destroying transform
  sitting between you and the output. Because the builder optimizes *shape correctness*, the
  call-site→output-line mapping is destroyed in the process. Note that shape here means **grammar**,
  not styling — and the builders do provide a grammatically correct shape by definition, since the
  representation cannot hold malformed Go. That advantage is not that valuable, though: with line-by-line
  rendering the formatter (`gofmt`/`fancyfmt`) will fail on grammatically incorrect output, and per
  the panic/`os.Exit` philosophy that failure is a hard stop. The same grammatical guarantee is thus
  achieved by validation rather than by construction — without paying for it with discoverability.

The line-by-line model keeps the mapping from a generator call site to an output line largely
intact: `L(format, args...)` is one logical line per call, so when generated code is wrong you grep
the generator for the offending format string. `gofmt` moves braces and aligns, but it does not
reorder the mental model of which line came from where.

On the {composability, discoverability} frontier: templates optimize discoverability only at zero
composition and lose it as composition grows; AST/tree builders optimize composability fully but
lose discoverability to the printer. Line-by-line keeps both, which is why it was chosen.

## Panics and `os.Exit` are intentional.

If something cannot be rendered properly, the generated code is worthless. A generator that emits
*plausible but wrong* code is strictly worse than one that stops: plausible-wrong code compiles,
passes superficial review, merges, and fails in production where the failure is expensive and far
from its cause. Generated code is trusted precisely because it is machine-produced, so the
correctness guarantee *is* the product — a hard stop with a trace is unambiguous and happens at the
source.

The recovered trace is the discoverability mechanism, relocated: you do not get "which output line
was being written" from a partial file, you get "which generator call was executing" from the stack.
For a model where one `L` maps to one line, the call site *is* the discoverability unit, so the
panic path preserves the metric the library cares about. Producing a half-rendered file would
*degrade* discoverability by forcing the defects to be mapped back to calls by eye.

## A note on locality.

The one place where discoverability is at risk *from the library itself* is the format-string
mini-language (`$name`, `@ident`, the `P`/`p`/`R`/`_`/`-` verbs): a single `L` call's output can
depend on `Let`/`Uniq` state set elsewhere, so the call-site→meaning mapping is sometimes
non-local. This is a deliberate trade of terseness for locality. It stays reconstructable rather
than mutable-and-lost because `Let` is immutable and a panic trace walks the scopes — but it is not
free, and it is worth being aware of when reading generated output.

## On exact source mapping.

Discoverability here is approximate, not exact-by-construction. An earlier prototype of this
library (not in this repo) tracked generator call sites against output lines, so that a `gofmt`
`file:line` failure pointed straight at the codegen line that produced the offending output. It was
dropped as too expensive — capturing the mapping relied on `runtime.Stack`, which is not cheap, and
the cost would only grow here: `Z()` reorders output relative to call order, so mapping output lines
back to their source calls would have to track block insertions on top of the per-line capture.
gogh settles for the cheap, approximate mapping that one-`L`-per-line already gives, plus the
panic trace on failure.

# Using gogh with an LLM.

There is an [LLM_CONTEXT.md](LLM_CONTEXT.md) file in the repository root — a condensed, LLM-oriented
reference of the gogh API (mental model, entry points, format strings, gotchas). Drop it into the
context of the code generator you are writing so the assistant works with gogh the way the library
expects, instead of guessing the API.

## Fetching it into your project.

The file is plain markdown, so a single `curl` is enough:

```shell
curl -fsSL https://raw.githubusercontent.com/sirkon/gogh/master/LLM_CONTEXT.md -o gogh-context.md
```

If you prefer to pull it straight from a checkout (and keep it trivially updatable), clone into a
temporary directory and copy:

```shell
tmpdir="$(mktemp -d)"
git clone --depth 1 https://github.com/sirkon/gogh.git "$tmpdir/gogh"
cp "$tmpdir/gogh/LLM_CONTEXT.md" ./gogh-context.md
rm -rf "$tmpdir"
```

Put the file wherever your tool looks for context. A couple of common cases:

* **Claude Code** — keep the file in the repo (e.g. `gogh-context.md`) and reference it from
  `CLAUDE.md` with an import:
  ```
  @gogh-context.md
  ```
  or mention it with `@gogh-context.md` in the prompt when working on the generator.

* **Cursor** — save it as a rule under `.cursor/rules/`, e.g.
  `.cursor/rules/gogh.mdc` (add the usual rule frontmatter on top).

Re-run the `curl`/`clone` snippet to refresh the context when you bump the gogh version.

# Troubleshooting.

## The package-name cache.

`gogh.New` opens a boltDB cache under the user cache directory to remember the package names of
imported packages (so it does not have to shell out to `go list` — which can take seconds — for them
on every run). The file lives at:

```
<user cache dir>/GoghProjects/bolt.db
```

which resolves to:

| OS      | Path                                          |
|---------|-----------------------------------------------|
| macOS   | `~/Library/Caches/GoghProjects/bolt.db`       |
| Linux   | `~/.cache/GoghProjects/bolt.db`               |
| Windows | `%LocalAppData%\GoghProjects\bolt.db`         |

The cache is self-invalidating, so manual clearing is rarely needed:

* Entries are keyed by `(package path, declared version)` parsed from `go.mod`. When you bump a
  dependency, its version changes, the key changes, and the old entry is simply not hit — gogh
  re-fetches that package and stores it under the new key. Nothing else is touched.
* Replaced, workspaced and local packages are never bolt-cached: their package name is read
  straight from the on-disk source (parsing only the package clause), which is cheap and always
  fresh. They never go through `go list`.
* Packages absent from `go.mod` (the standard library, transitive packages not declared there) are
  cached by package path alone — their names change rarely.
* A per-module in-memory cache additionally ensures each package name is resolved at most once per
  `Module`, even across the `Type`/`Object` rendering paths that resolve names directly.

In the rare event the cache file becomes corrupt, delete `<user cache dir>/GoghProjects/bolt.db` —
gogh recreates it on the next run.

## `go build ./...` fails on `golang.org/x/tools`.

A stale `golang.org/x/tools` dependency may fail to compile on newer Go toolchains (e.g. `invalid
array length -delta * delta`). Bump it:

```shell
go get golang.org/x/tools@latest
go mod tidy
```

## Generation panics with a "Calls trace in generator".

This is intentional (see [Rationale](#rationale)). The trace lists the generator call sites that
led to the failure — start from the topmost frame that is your code. The most common causes are a
`Ref`/`Let` name collision (use `TryLet` or pick a unique reference name), an unsupported
`types.Type`/`past.Type` passed to `Type`/`Proto`, or a format-string error (wrong number of
positional args, unknown `$name`, bad `@ident`).

# About mimchain utility.

## Installation.

```shell
go install github.com/sirkon/gogh/cmd/mimchain
```

## What is it?

It is a tool to generate rendering helpers mimicking types with chaining methods. Take a look at my 
custom [errors](https://github.com/sirkon/errors) package. It is done to deliver structured context with
errors, for structured loggers mostly in order to follow "log only once" approach:

```go
return 0, errors.Wrap(err, "count something").Int("stopped-count-at", count).Str("place", "placeName")
```

where we collect context, including structured context into errors and log them just once at the root level.

Building these with just a renderer can be pretty annoying:

```go
r.L(`return $ReturnZeroValues $errors.Wrap(err, "count $0").Int("stopped-count-at, $countVar).Str("place", "some error place")`, what)
```

This utility can generate dedicated code renderers that can be somewhat easier to use with an IDE support:

```go
ers.R(r, what).Wrap("err", "count $0").Int("stopped-count-at", "$countVar").Str("place", placeName)
```

The code it produces is not ready to use though:

  - No constructors like `R` for generated rendering entities. You need to write what's needed.
  - Another issue is with string arguments. See at the code sample above: some methods like `Bool`, `Str`, `Uint64`, 
    etc, will be called with a direct string literal as their first argument mostly and the second argument is very 
    likely to be a variable. 

The first part is trivial, you can write it yourself with all tweaks you want. 
The second is harder. There's an option currently which enables force quotes for constructors and type 
methods renderers. A code generated will quote an argument if it always has string type for functions having the 
same amount of parameters.

And remember: it is not a crime to tweak generated code manually, the lack of "DO NOT EDIT" header there
is not a coincidence.

This library provides `Q`, `L` and `QuoteBias` helpers to deal with string quotes:

- `Q` is useful when string values are meant to have literal rendering – it will turn them into quoted strings.
- `L` is a vice versa – it is useful when string values are meant to have quoted rendering.
- `QuoteBias` function turns strings values into quoted strings.

These are meant to be used for relatively easy tweaking of a source code generated.

## Example.

It is [testexample](https://github.com/sirkon/gogh/tree/master/cmd/mimchain/internal/testexample). 



