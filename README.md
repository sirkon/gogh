# gogh

Go source code rendering library. The name `gogh` comes from both `GO Generator` and from the fact I adore Van Gogh 
writings.

## Installation

```shell script
go get github.com/sirkon/gogh
```

## Simple usage

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

## Importers

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

And you will just

```go
r.Imports().Company("configs").Ref("configs")
r.L(`// Config service $0 config definition`, serviceName)
r.L(`type Config struct{`)
r.L(`    TLS *$configs.TLS`)
r.L(`    Service *&configs.Service`)
r.L(`}`)
```

And remember, using `Ref` to assign rendering context value is the preferable way to access imported packages:
`*gogh.GoRenderer` will take care of conflicting names, aliases, etc. Just make sure reference name is unique for the
renderer.

