# gogh — LLM usage context

> Reference for an LLM that needs to **write or modify Go code that uses the `gogh` library** to generate Go source.
> Module: `github.com/sirkon/gogh` · Go 1.26+ · Root package is `gogh` (all public API lives there).

## What gogh is

`gogh` is a **Go source rendering library**. You build Go files programmatically by writing formatted lines through a renderer; gogh takes care of the parts that are tedious and error-prone in hand-written generators:

- collecting and emitting the `import` block,
- resolving package qualifier names and alias collisions automatically,
- running `gofmt` (or `fancyfmt`) on the result,
- writing the files into the correct location inside the current Go module.

Use it when writing `go generate` tools, protoc plugins, schema-to-Go generators, etc. The model is **"write lines that look like Go, with `$placeholders` for imported-package qualifiers and positional args"**.

---

## Mental model (read this first)

1. **`Module[T]`** — one generation session, tied to the Go module on disk (discovered via `go env` / `go mod edit`). You create it once with `gogh.New(...)`. `Render()` at the end flushes **all** files to disk. Must run inside a module directory; shells out to the `go` toolchain and uses a boltDB cache for package-name lookups. That cache is **self-invalidating**: versioned deps are keyed by `(package path, declared version)` from `go.mod` so a bump invalidates exactly the affected entry; replaced/workspaced/local packages are read straight from disk (cheap, always fresh, never bolt-cached); stdlib/undeclared-transitive are cached by package path. A per-module in-memory cache ensures each name is resolved at most once per Module.
2. **`Package[T]`** — a Go package within the module. Get one from the module (`Root`, `Current`, `Package`, `PackageName`). Each package produces one or more file renderers.
3. **`GoRenderer[T]`** — the writing surface for one Go file. You write lines into it; it owns that file's imports and rendering context.
4. **`RawRenderer`** — like `GoRenderer` but for plain-text files (no package header, no imports). Same line API.
5. **`Importer` / `Imports`** — import management. `Add(pkgpath).Ref("name")` registers an import and binds its qualifier into the rendering context so `$name` expands to the (possibly aliased) package name.
6. **Rendering context / scope** — a hierarchical map of named values per renderer. Populated by `Ref`, `Let`, `Uniq`, `UniqBind`, and the function helpers (`Returns` sets `ReturnZeroValues`). Exposed in format strings as `$name`; positional args are `$0`, `$1`, …

The generic type parameter `T` is the importer type. For most uses it's just `*gogh.Imports` (see the "simple usage" boilerplate below). A custom `T` is only needed when you want typed shortcut methods for frequently-imported packages.

---

## Minimal end-to-end example

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
		func(r *gogh.Imports) *gogh.Imports { return r }, // identity importer
	)
	if err != nil {
		message.Fatal(errors.Wrap(err, "setup module info"))
	}

	pkg, err := prj.Root("project")
	if err != nil {
		message.Fatal(errors.Wrap(err, "setup package "+prj.Name()))
	}

	r := pkg.Go("main.go", gogh.Shy) // Shy = don't overwrite if file exists
	r.Imports().Add("fmt").Ref("fmt")

	r.L(`func main() {`)
	r.L(`    $fmt.Println("Hello $0!")`, "World")
	r.L(`}`)

	if err := prj.Render(); err != nil { // writes file(s) to disk
		message.Fatal(errors.Wrap(err, "render module"))
	}
}
```

The two-arg boilerplate `func(r *gogh.Imports) *gogh.Imports { return r }` is the default importer. Remember to call `prj.Render()` — nothing is written until then.

---

## Entry points

### `gogh.New[T Importer](formatter, importer, opts...) (*Module[T], error)`

- `formatter`: `gogh.GoFmt` (gofmt) or `gogh.FancyFmt` (github.com/sirkon/fancyfmt).
- `importer`: `func(r *gogh.Imports) T`. Use the identity function for the default, or `NewCustomImporter` for typed shortcuts.
- `opts`: `WithAliasCorrector(fn)`, `WithFixedDeps(map[path]semver.Version)`, `WithProtoRegistry(*protoast.Registry)`.

### `Module[T]` methods

| Method | Purpose |
|---|---|
| `Root(name) (*Package[T], error)` | package at the module root |
| `Current(name) (*Package[T], error)` | package at the current working directory |
| `Package(name, pkgpath) (*Package[T], error)` | subpackage; `pkgpath` may be module-relative or full (module prefix stripped) |
| `PackageName(pkgpath) (string, error)` | look up an existing package's name |
| `Raw(relpath, opts...) *RawRenderer` | plain-text file at module root |
| `Name() string` | module path |
| `GetDependency(path, version)` / `GetDependencyLatest(path)` | run `go get` |
| `Render() error` | **flush all files to disk** (terminal call) |

### `Package[T]` methods

| Method | Purpose |
|---|---|
| `Go(name, opts...) *GoRenderer[T]` | create/reuse a Go source file renderer |
| `Reuse(name) (*GoRenderer[T], error)` | render into an **existing** file (parses its imports, appends) |
| `Void() *GoRenderer[T]` | renderer that produces no file — use for side-effect imports when calling `Type`/`Object`/`Proto` |
| `Raw(name, opts...) *RawRenderer` | plain-text file in this package |
| `Package(name, pkgpath) (*Package[T], error)` | subpackage of this package |
| `Path() string` | full import path |

### `RendererOption`s (passed to `Go`/`Raw`)

| Option | Effect |
|---|---|
| `gogh.Shy` | skip writing if the file already exists |
| `gogh.Autogen(appname)` | prepend `// Code generated by <appname> … DO NOT EDIT.` |
| `gogh.WithValues(map[string]any)` | seed the rendering context |
| `gogh.WithValue(name, value)` | seed a single context value |

---

## Imports & the rendering context

```go
r.Imports().Add("fmt").Ref("fmt")                 // foreign package
r.Imports().Module("internal/configs").Ref("cfg") // package within current module
r.Imports().Add("company.org/gopkgs/configs").As("c").Ref("configs")
```

- `Add(pkgpath)` / `Module(relpath)` → `*ImportAliasControl`.
- `.As(alias)` → force an explicit alias → `*ImportReferenceControl`.
- `.Ref(name)` → bind the resolved qualifier into the context as `$name`.

**Alias conflicts are resolved automatically**: gogh tries the package's real name, then the `AliasCorrector` (if any), then `<name>2`, `<name>3`, …. So `$fmt` always expands to the correct, non-conflicting qualifier whether or not an alias was needed.

**Imports are file-global**: a `Ref` made on any renderer derived from the same file (via `Z`, `Scope`, `InnerScope`) is visible to all of them.

---

## Writing lines — `GoRenderer` / `RawRenderer` API

| Method | Description |
|---|---|
| `L(format, args...)` | render a formatted line + newline (**primary method**) |
| `C(args...)` | concatenate args with spaces + newline |
| `R(text)` | raw (unformatted) line + newline |
| `N()` | blank line |
| `S(format, args...)` | like `L` but returns the string instead of writing it |
| `Z()` | **laZy** writer — returns a renderer whose writes appear *before* subsequent writes of the caller (inserts a block). Enables multi-pass generation in one loop. |
| `T()` | "temporary" renderer for the same package that produces no file — use so `Type`/`Proto` calls register imports without emitting text |
| `Scope()` | new renderer with an inherited-but-isolated scope (uniqs + `Let` values copied; later changes don't propagate back; imports still shared) |
| `InnerScope(func(r))` | convenience: `func(r.Scope())` |
| `Parent()` | the renderer that created this one (panics if none) |

### Names & context

| Method | Description |
|---|---|
| `Uniq(name, optSuffix...)` | generate a unique identifier in the scope (tries `name`, then `name<suffix>`, then `name1`, `name2`, …) |
| `UniqBind(tag, name, optSuffix...)` | `Uniq` + bind `tag` so it formats as the unique value |
| `Taken(name) bool` | was this name already taken in the scope? |
| `Let(name, value)` | set an **immutable** named context value (panics if re-set with a different value) |
| `TryLet(name, value)` | like `Let` but no panic if already set |
| `SetReturnZeroValues(vals...)` | set the `ReturnZeroValues` context value |
| `InCtx(name) bool` | is this name present in the context? |

### Type rendering (auto-imports)

| Method | Description |
|---|---|
| `Type(t types.Type) string` | fully-qualified Go type name from `go/types` (pointers/slices/maps/chans/signatures/arrays/named/alias). Auto-adds imports. |
| `Object(item types.Object) string` | fully-qualified object |
| `PkgObject(pkgRef, name) string` | qualified object in a referenced package; `pkgRef` may be `types.Object`, `*types.Named`, `*GoRenderer[T]`, or a string path |
| `Proto(t past.Type) ProtocType` | protoc-gen-go type name from a [protoast](https://github.com/sirkon/protoast) type; handles google wrappers, nested messages/enums, `go_package` |

`ProtocType` methods: `String()`, `Impl()` (with pointer if needed), `Local()`, `LocalImpl()`, `Pkg()`.

> **`Type` caveat — names used only in strings/comments.** `Type` (and `Object`/`PkgObject`) register an import for the type's package as a side effect. If you render a type name only to embed it in a **comment or a string literal** (not as real code), gogh still adds the import — producing an unused import the Go toolchain will reject. To get the fully-qualified name *without* registering an import on the real file, render it through a throwaway renderer: `r.T().S(...)` via `Type`/`Object`, or use `T()` + `S` on the temporary renderer and interpolate the returned string into your comment. The `T()` renderer belongs to the same package (so it shares import resolution) but its output is discarded.

---

## Format strings

Built on [go-format](https://github.com/sirkon/go-format). Two kinds of substitution:

- **Positional**: `$0`, `$1`, `$2`, … map to the variadic args of `L`/`S`.
- **Named**: `$name` expands a context value set via `Ref`/`Let`/`UniqBind` (e.g. `$fmt`, `$cfg`, `$ReturnZeroValues`).

### Auto-conversion of arg types

- `types.Type` and `ast.Type` (proto `past.Type`) are converted to their string form automatically.
- `Commas` / `Params` render as comma-separated lists; with the `\n` format option they render one-per-line (multiline).

### String / `fmt.Stringer` formatting options

| Option | Effect |
|---|---|
| `P` | `gogh.Public(value)` — exported identifier casing |
| `p` | `gogh.Private(value)` — unexported identifier casing |
| `R` | `gogh.Proto(value)` — proto casing |
| `_` | `gogh.Underscored(value)` |
| `-` | `gogh.Striked(value)` |

### Inline unique names (`@` syntax)

`@ident` (or `@{ident}`) in a format string is exactly equivalent to
`r.Let("ident", r.Uniq("ident"))` followed by referring to `$ident`: it mints a
unique name in the current scope and binds it into the rendering context under
that name, so every subsequent `@ident` **or** `$ident` in the same scope
resolves to the same identifier. `@@` is a literal `@`.

```go
r.L(`@id := indexService.Get()`) // ≡ r.Let("id", r.Uniq("id")); writes "$id := …"
r.L(`$fmt.Println(@id)`)         // same scope → same name "id"
```

Because it binds through `Let` (immutable for the scope's lifetime), the name
stays constant across all uses in that scope. To mint a *different* unique name
for the same base, use a new scope (`Scope()` / `InnerScope`).

---

## Functions and methods — `F` / `M`

> **Not the default way to render functions/methods.** `F` and `M` are a
> **specialized instrument for generating them en masse** — typically by
> mirroring existing `go/types` signatures in bulk. For ordinary,
> hand-specified functions and methods, prefer writing the signature directly
> with `L`:
>
> ```go
> r.L(`func (d *DataType) MethodName(name string) (Result, error) {`)
> r.L(`    …`)
> r.L(`}`)
> ```
>
> Reach for `F`/`M` only when you are stamping out many definitions from
> existing signatures (or need the auto-computed `ReturnZeroValues`).

`F` and `M` build function/method definitions, often by mirroring existing signatures.

```go
r.F("newExample")().Returns("error").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
    r.L(`return errors.New("something failed")`)
})
```

- `F(name)` → `func(params...) *GoFuncRenderer[T]`
- `M(rcvr...)` → `func(name)` → `func(params...) *GoFuncRenderer[T]`. Receiver is a single type/string/`*types.Var`/`types.Type`/`past.Type`, or a `(name, type)` pair.
- `GoFuncRenderer.Returns(results...)` → `*GoFuncBodyRenderer[T]`. **Auto-computes zero values** for the return types and binds them under the `ReturnZeroValues` context key, so error bodies can write `return $ReturnZeroValues $errors.New(...)`.
- `Body(func(r *GoRenderer[T]))` — render the body.

**Param/result argument forms** (flexible but strict): `Params`, `Commas`, `*types.Tuple`, a `*types.Var` list, or alternating `key/value` pairs where values may be `string` / `fmt.Stringer` / `types.Type` / `past.Type`. Don't mix named and unnamed forms; identifiers must be valid Go identifiers.

---

## Lazy generation with `Z()`

`Z()` inserts a text block and hands you a renderer that writes into the *gap before* the caller's next write. This lets you build several interleaved parts of a declaration in a single loop pass.

```go
r.L(`type Data struct {`)
s := r.Z() // struct fields
r.L(`}`)

r.N()
r.L(`func dataConstructor(`)
a := r.Z() // constructor arguments
r.L(`) *Data {`)
r.L(`    return &Data{`)
c := r.Z() // field assignments
r.L(`    }`)
r.L(`}`)

for _, item := range items {
    s.L(`$0 $1`, item.name, item.typeName)
    a.L(`$0 $1,`, item.name, item.typeName)
    c.L(`$0: $1,`, item.name)
}
```

---

## Scopes

- `r.Z()` produces a new renderer **sharing** `r`'s scope (uniqs and context are the same object).
- `r.Scope()` produces a new renderer with a **child** scope:
  - starts with the same set of uniqs and context values (so identical `Uniq`/`Let` calls reproduce the same output),
  - splits after creation — new uniqs/`Let` values in the child do not propagate back to the parent or to siblings,
  - **imports stay shared** (file-global) — `Ref` on any renderer for a file affects all of them.

This is the recent "scopes get hierarchy" behavior: the context is a parent-pointer chain; `Get` walks up, `Set`/`Uniq` write only the current level.

---

## Custom importer (typed shortcuts)

Wrap `*gogh.Imports` to satisfy `gogh.Importer` and add shortcut methods:

```go
type CustomImporter struct{ i *gogh.Imports }

func (i *CustomImporter) Imports() *gogh.Imports          { return i.i }
func (i *CustomImporter) Add(p string) *gogh.ImportAliasControl   { return i.i.Add(p) }
func (i *CustomImporter) Module(p string) *gogh.ImportAliasControl { return i.i.Module(p) }

func (i *CustomImporter) Company(relpath string) *gogh.ImportAliasControl {
	return i.i.Add("company.org/gopkgs/" + relpath)
}
```

```go
mod, _ := gogh.New(gogh.GoFmt, NewCustomImporter) // pass the constructor as the importer arg
r.Imports().Company("configs").Ref("configs")
r.L(`type Config struct{`)
r.L(`    TLS *$configs.TLS`)
r.L(`}`)
```

---

## Supporting helpers

| Symbol | Purpose |
|---|---|
| `Params` / `Commas` | param/comma-list builders with `.Add(...)`; `.String()` (one line) and `.Multi()` (one per line) |
| `A(a...) Commas` | quick comma list from strings |
| `Q("x")` | render as a **quoted** Go string literal |
| `L("x")` | render a string value **literally** (un-quoted) |
| `QuoteBias(v)` | quote string values, pass others through |
| `Public`, `Private`, `Underscored`, `Striked`, `Proto` | casing functions |
| `RegisterInitalism(word)` | register a custom initialism (note: the symbol name has this historical typo) |
| `ReturnZeroValues` | context key name for zero-return values |
| `GoFmt`, `FancyFmt`, `Formatter` | formatters |

---

## Gotchas an LLM must respect

- **Always call `prj.Render()`** at the end. Nothing is written to disk until then.
- **Run inside a Go module.** `New` shells out to `go env` / `go mod edit` / `go list` and opens a boltDB cache under the user cache dir; foreign package-name lookups need the local Go toolchain (and possibly network). The cache is version-keyed off `go.mod` (dep bumps auto-invalidate); replaced/workspaced/local packages are read from disk (no `go list`); stdlib/undeclared are path-cached. A per-module in-memory cache avoids re-resolving within a Module. Manual clearing is rarely needed (see README → Troubleshooting).
- **`Ref` names must be unique per file.** gogh panics on conflicting `Ref`/`Let` redefinitions with differing values. Use `TryLet` when you can't guarantee uniqueness.
- **`Uniq` and `Let` are independent namespaces** — a `Let("x", ...)` does not reserve the name `x` against `Uniq("x")`, and vice-versa.
- **Imports are file-global**, not scope-local. A `Ref` on a `Scope()`-child affects the whole file by design.
- **`New` may panic** during rendering; panics are recovered and reformatted into a generator call-trace, then `os.Exit(1)`. Prefer to validate inputs before writing lines.
- **Prefer `Ref` over hand-written qualifiers.** Let gogh resolve aliases instead of hard-coding package names.
- **Use a type alias** to keep signatures readable: `type goRenderer = gogh.GoRenderer[*gogh.Imports]`, then `func renderSomething(r *goRenderer) {...}`.
- **`Shy`** prevents overwriting existing files — useful for files users may hand-edit; `Autogen(name)` marks machine-generated files.
- **`Void()` / `T()`** for side-effect-only type rendering (registering imports without emitting text).

---

## Reference layout of a generator program

```go
func main() {
    prj, err := gogh.New(gogh.GoFmt, func(r *gogh.Imports) *gogh.Imports { return r })
    if err != nil { /* fatal */ }

    pkg, err := prj.Current("generated")   // or Root / Package
    if err != nil { /* fatal */ }

    r := pkg.Go("gen_types.go", gogh.Autogen("mytool"))
    r.Imports().Add("fmt").Ref("fmt")
    r.Imports().Module("internal/config").Ref("cfg")

    // 1. (optional) build scaffolding with Z() blocks
    // 2. loop over source data, filling blocks and calling Type()/F()/M()
    // 3. Let() any shared context values

    if err := prj.Render(); err != nil { /* fatal */ }
}
```

## Where to look in the repo for more

- `README.md` — the canonical, human-oriented doc (covers the same API with the author's framing).
- `cmd/mimchain/` — a real generator built with gogh (see the next section).
- `cmd/mimchain/internal/testexample/` — a runnable worked example: `testexample_test.go` drives generation; `rendering_example.go` is the generated output.
- Godoc on `module.go`, `package.go`, `imports.go`, `renderer_go.go`, `renderer_go_func.go`, `renderer_options.go`, `module_options.go` — thorough per-symbol docs.

---

## mimchain — generating chaining-renderer helper types

`cmd/mimchain` is a ready-made generator (itself built with gogh) that produces **code-renderer types mirroring an existing chaining type**. Given a type whose constructors and methods return the type itself, it emits a `<Name>[T gogh.Importer]` / `<Name>Attr[T]` pair that lets other generators write fluent call chains with IDE completion, instead of hand-writing one long `r.L(...)` format string:

```go
// instead of:
r.L(`return $ReturnZeroValues $errors.Wrap(err, "count $0").Int("stopped-count-at", $count).Str("place", "placeName")`, what)
// write:
wrp.R(r, what).Wrap("err", "count $0").Int("stopped-count-at", "$count").Str("place", placeName)
```

### Installation & invocation

```shell
go install github.com/sirkon/gogh/cmd/mimchain
```

```
mimchain [--string-args-quoted|-q] [--package-name|-p STRING] <type> <dst>
```

- Both `<type>` and `<dst>` are `<pkgpath>:<TypeName>` points (module-relative paths like `./internal/wrp` are fine; the path is resolved via `go list`). Both type names must be **exported** (`gogh.Public` casing is enforced).
- Output is written with `gogh.Shy` — an existing `<name>_generated.go` is **never overwritten**; delete it first to regenerate.
- `<type>` — the existing chaining type to mirror.
- `<dst>` — the wrapper type to generate. Output goes to `<underscored_type>_generated.go` in the destination package.
- `-p/--package-name` — name for the destination package **if it doesn't exist yet**. Note the directory must already contain at least one Go file (a `doc.go` is enough), otherwise `go list` fails; ignored when the package exists.
- `-q/--string-args-quoted` — in the *generated renderer*, auto-quote `string` args (via `strconv.Quote`) at positions that are **always strings across the whole group** of mirrored signatures. Typical use: `errors.New(msg)` → `New("literal message")` while still passing format strings unquoted to go through `r.S(...)` with positional args. Alternatively handle this per-call-site with `gogh.Q` / `gogh.L` / `gogh.QuoteBias` values.
- Use `--` to separate from `go:generate`-style args: `//go:generate go run . -- -q github.com/sirkon/errors:Error ./internal/testexample:Error`.

### What gets mirrored (eligibility rules)

A package-level function is mirrored as a **constructor** (on `<Name>`) if it is exported and its single result is exactly the `<type>` **value type**. A method is mirrored (on `<Name>Attr`) under the same condition. Consequences:

- Only types whose chain methods return the **value type** work (`func (c Chain) Add(...) Chain`, not `*Chain` — return-type identity, not assignability, is checked, so `*Error` results don't qualify; pointer receivers on the source are fine).
- Functions/methods with zero results or multiple results are skipped.
- Mirrored constructors/methods with the same parameter count (+variadicness) share one generated base implementation (`constructor1`, `method2variadic`, …), grouped by signature weight.

### What the generated code looks like and how to use it

The generated `<Name>` and `<Name>Attr` types have `r *gogh.GoRenderer[T]`, `buf *bytes.Buffer`, `a []any` fields plus `String()`. Each chained call appends text into `buf` and returns the `Attr` type. Arg rendering per call: `fmt.Stringer` args are rendered via their `String()` (this is how `gogh.Q`/`gogh.L`/`gogh.QuoteBias` control quoting), plain `string` args go through `r.S(v, a...)` so `$0`/`$name` substitution still applies, anything else falls back to `fmt.Sprint`. The `a` slice comes from your hand-written entry-point constructor (e.g. `R(r, what)` makes `$0` expand to `what` in every string of the chain).

**mimchain does not generate the entry-point constructor** (the analog of `R(r)` in the example below) — write it by hand, typically using `gogh.GoRendererBuffer(r)` to get a buffer that flows into the right place of the file, plus `r.S("$"+gogh.ReturnZeroValues)` if you're building a `return` expression:

```go
// R creates `return ....` renderer with an error expression in it.
func R[T gogh.Importer](r *gogh.GoRenderer[T], a ...any) *Error[T] {
	r = r.Scope()
	buffer := gogh.GoRendererBuffer(r)
	buffer.WriteString("return ")
	buffer.WriteString(r.S("$" + gogh.ReturnZeroValues))
	return &Error[T]{r: r, buf: buffer, a: a}
}
```

Then inside `F`/`M` bodies (which set `ReturnZeroValues` for you):

```go
r.F("newExample")().Returns("error").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
	R(r).New("something failed").Str("something", gogh.Q("value"))
})
```

Remember to call `r.SetReturnZeroValues(...)` when constructing the renderer yourself (outside `F`/`M`), and that generated files have no "DO NOT EDIT" header — tweaking them by hand is expected.

### Runnable example

`cmd/mimchain/internal/testexample/`:

- `error.go` — the hand-written `R` entry-point constructor.
- `error_generated.go` — mimchain output for `github.com/sirkon/errors:Error` (note: generated against errors **v0.5.0**, where constructors returned `Error` by value; the current errors releases return `*Error` and no longer match the eligibility rules).
- `testexample_test.go` — regenerates `rendering_example.go` via `prj.Package("", "cmd/mimchain/internal/testexample")` and `r.F(...)` bodies using `R(r).Newf(...)` etc.
- `rendering_example.go` — the committed result of that generation.

### mimchain as gogh usage reference

`cmd/mimchain/*.go` is also the canonical "real generator" sample: kong CLI parsing with `sourcePoint`-style args validated through `UnmarshalText`, loading source types via `golang.org/x/tools/go/packages`, `Z()` + `g.b` for two-phase rendering (per-method facades first, shared base implementations after), `Scope()`/`Let` for per-group context isolation, `M` with `$x`/`*$gtype[T]` receiver strings, and `FancyFmt` as the formatter.
