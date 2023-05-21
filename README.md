# gogh

Go source code rendering library. The name `gogh` comes from both `GO Generator` and from the fact I adore Van Gogh
writings.

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
| `L(format, params...)` | Render and put text line using custom format. <br/>See [go-format](github.com/sirkon/go-format) for details.                             |
| `R(text)`              | Put raw text                                                                                                                             |
| `N()`                  | Put new line                                                                                                                             |
| `S(format, params...)` | Same as `L` but returns rendered text as a string without saving it.                                                                     |
| `Z()`                  | Returns new renderer which will put lines before<br/>any line made by the original renderer.<br/> Set details below.                     |
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

The first part is trivial, you see. The second is harder. There's an option currently which enables force quotes
for constructors and type methods renderers. A code generated will quote an argument if it always has string type
for functions having the same amount of parameters.

And remember: it is not a crime to tweak generated code manually, the lack of "DO NOT EDIT" header there
is not a coincidence.

## Example.

It is [testexample](https://github.com/sirkon/gogh/tree/master/cmd/mimchain/internal/testexample). 


