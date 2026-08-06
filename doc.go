// Package gogh renders Go source code.
//
// It builds Go files programmatically: you write formatted lines through a
// renderer and gogh takes care of collecting the import block, resolving
// package qualifier names and alias collisions, running gofmt (or fancyfmt),
// and writing the files into the current Go module.
//
// A typical session creates a Module with New, obtains a Package and a
// GoRenderer from it, writes lines with L referencing imported packages
// through Ref-bound names (e.g. $fmt), and finishes with a single Render call
// that flushes all files to disk:
//
//	prj, err := gogh.New(
//		gogh.GoFmt,
//		func(r *gogh.Imports) *gogh.Imports { return r },
//	)
//	if err != nil {
//		// handle error
//	}
//
//	pkg, err := prj.Root("project")
//	if err != nil {
//		// handle error
//	}
//
//	r := pkg.Go("main.go", gogh.Shy)
//	r.Imports().Add("fmt").Ref("fmt")
//	r.L(`func main() {`)
//	r.L(`    $fmt.Println("Hello $0!")`, "World")
//	r.L(`}`)
//
//	if err := prj.Render(); err != nil {
//		// handle error
//	}
//
// See the package README for the full API and the cmd/mimchain directory for a
// complete generator built with gogh.
package gogh
