# gogh
Go source code rendering library. The name `gogh` comes from both `GO Generator` and from the fact I love Van Gogh works.

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
    gogh.Line(&r, `func main() {`)
    gogh.Line(&r, `    greet := "World"`)
    gogh.Line(&r, `    fmt.Println("Hello $0!", greet)`)
    gogh.Line(&r, `}`)
    src, err := r.RenderAutogen("appName", "main", imports.Result())
    if err != nil {
        panic(err)        
    }

    _, _ = io.Copy(os.Stdout, src)
}
```

## Recommended usage

It is not actually recommended to use it directly. Embed provided objects and use it in integrated
manner. For instance, it would be sensible to combine both `gogh.Imports` and `gogh.Go` in an single
instance somehow as imports are naturally a part of code generation. They are splitted with render
object for the only reason: there can be frequently used libraries what it would be nice to have
shortucts for. In addition, some custom import weighter would be handy to split imports to groups of
your liking.

