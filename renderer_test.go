package gogh_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/sirkon/gogh"
)

var importGroups = []gogh.ImportsGroup{
	{
		{
			Path: "C",
		},
	},
	{
		{
			Path: "fmt",
		},
	},
	{
		{
			Alias: "pkg",
			Path:  "github.com/sirkon/gogh",
		},
	},
}

func TestMustPanicOnNilDefaultContext(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("panic expected")
			return
		}
		t.Log(r)
	}()

	var r gogh.Renderer
	gogh.WithCtx(&r, nil)
}

func ExampleGoRendererNoHeadCommentError() {
	var r gogh.Renderer

	gogh.Line(&r, "func main() {")
	gogh.Line(&r, "    var ah string")

	_, err := gogh.Render(&r, "", "main", importGroups)
	fmt.Println(err)
	// Output:
	// expected ';', found 'EOF'
	//     var ah string
}

func ExampleGoRendererHeadSingleLineCommentOK() {
	var r gogh.Renderer

	gogh.Line(&r, `func main() {`)
	gogh.Line(&r, `}`)

	src, err := gogh.RenderAutogen(&r, "application", "comment", "main", importGroups)
	if err != nil {
		fmt.Println(err)
	} else {
		_, _ = io.Copy(os.Stdout, src)
	}
	// Output:
	// // Code generated with application. DO NOT EDIT.
	//
	// // comment
	//
	// package main
	//
	// import "C"
	//
	// import (
	//	"fmt"
	//
	//	pkg "github.com/sirkon/gogh"
	// )
	//
	// func main() {
	// }
}

func ExampleGoRendererHeadMultiLineCommentOK() {
	var r gogh.Renderer

	gogh.Line(&r, `func main() {`)
	gogh.Line(&r, `}`)

	src, err := gogh.Render(&r, "autogenerated code\napplication", "main", importGroups)
	if err != nil {
		fmt.Println(err)
	} else {
		_, _ = io.Copy(os.Stdout, src)
	}
	// Output:
	// /*
	// autogenerated code
	// application
	// */
	//
	// package main
	//
	// import "C"
	//
	// import (
	//	"fmt"
	//
	//	pkg "github.com/sirkon/gogh"
	// )
	//
	// func main() {
	// }
}

func ExampleWithRawlAndNewlAndContext() {
	var text struct {
		Text string
	}
	text.Text = "Hello world!"

	var r gogh.Renderer
	gogh.WithCtx(&r, text)

	imports := gogh.NewImports(gogh.GenericWeighter())
	imports.Add("", "fmt")

	gogh.Rawl(&r, `func main() {`)
	gogh.Line(&r, `    text := "${Text}"`)
	gogh.Newl(&r)
	gogh.Line(&r, `    fmt.Println(text)`)
	gogh.Rawl(&r, `}`)
	src, err := gogh.Render(&r, "", "main", imports.Result())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, _ = io.Copy(os.Stdout, src)

	// Output:
	// package main
	//
	// import (
	//	"fmt"
	// )
	//
	// func main() {
	//	text := "Hello world!"
	//
	//	fmt.Println(text)
	// }
}

func TestRendererRawString(t *testing.T) {
	var r gogh.Renderer
	line := `a := &Struct{`
	gogh.Line(&r, line)

	lineNL := line + "\n"
	if gogh.RawString(&r) != lineNL {
		t.Errorf("`%s` expected, got `%s`", lineNL, gogh.RawString(&r))
	}
	if !bytes.Equal(gogh.RawBytes(&r), []byte(lineNL)) {
		t.Errorf("`%s` expected, got `%s`", lineNL, gogh.RawString(&r))
	}
}
