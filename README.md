# gogh
Go source code rendering library

## Installation

```shell script
go get github.com/sirkon/gogh
```

## Direct usage

```go
package main

import (
    "fmt"
    "io"
    "os"

    "github.com/sirkon/gogh"
)

func main() {
    imports := gogh.NewImports(gogh.GenericWeighter())
    imports.Add("", "fmt")
    
    var r gogh.Renderer
    r.Line(`func main() {`)
    r.Line(`    greet := "World"`)
    r.Line(`    fmt.Println("Hello $0!", greet)`)
    r.Line(`}`)
    src, err := r.RenderAutogen("appName", "main", r.Result())
    if err != nil {
        panic(err)        
    }

    _, _ = io.Copy(os.Stdout, src)
}
```

## Recommended usage

It is not actually recommended to use it directly. Embed provided objects and use it in integrated
maner. For instance, it would be sensible to combine both `gogh.Imports` and `gogh.Go` in an single
instance somehow as imports are naturally a part of code generation. They are splitted with render
object for the only reason: there can be frequently used libraries what it would be nice to have
shortucts for. In addition, some custom import weighter would be handy to split imports to groups of
your liking.

I would recommend something like that:

```go
package renderer

import (
    "strings"

    "github.com/sirkon/gogh"
)

type someCustomWeighter struct {
    Weighter
}

func (w Weighter) Weight(path string) int {
    switch v:= w.Weighter.Weight(path) {
    case 0, 1:
        return v
    default:
        if strings.HasPrefix(path, "company.org") {
            return v + 1
        }
        return v
    }   
}

type Imports struct {
    gogh.Imports
}

func (i *Imports) Errors() {
    i.Add("", "company.org/common/errors")
}

type File struct {
    gogh.Renderer
    pkgName string
    imports *Imports
}

func NewFile(pkgName string) *File {
    return &File{
        pkgName: pkgName,
        imports: &Imports{gogh.NewImports(someCustomWeighter{gogh.GenericWeighter()})},
    }
}

func (f *File) Render(comment string) (io.Reader, error) {
    return f.Renderer.Render(comment, f.pkgName, f.imports.Result())
}

func (f *File) RenderAutogen(appName string) (io.Reader, error) {
	return f.Renderer.RenderAutogen(appName, f.pkgName, f.imports.Result())
}

func (f *File) Imports() *Imports {
    return f.imports
}
``` 