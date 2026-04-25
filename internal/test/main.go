package main

import (
	"github.com/sirkon/errors"
	"github.com/sirkon/message"

	"github.com/sirkon/gogh"
)

func main() {
	m, err := gogh.New(
		gogh.FancyFmt,
		func(r *gogh.Imports) *gogh.Imports {
			return r
		},
	)
	if err != nil {
		message.Fatal(errors.Wrap(err, "open model object"))
		return
	}

	p, err := m.Package("", "internal/test")
	if err != nil {
		message.Fatal(errors.Wrap(err, "open current package"))
	}

	r := p.Go("add.go", gogh.Autogen("test"))

	r.Imports().Add("fmt")
	r.N()
	r.L(`func _() {`)
	r.L(`    fmt.Println("Hello World!")`)
	r.L(`}`)

	r.Imports().Add("context").Ref("ctx")
	r.Imports().Add("errors").Ref("errs")

	// r.SetReturnZeroValues(`""`, "")
	r.F("_")("ctx $ctx.Context").Returns("string", "error", "").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
		r.Imports().Add("strconv").Ref("conv")
		r.L(`<-ctx.Done()`)
		r.L(`return $ReturnZeroValues $errs.New("Hello!")`)
	})
	r.N()
	z := r.Z()

	r.Let("dt", "DataType")
	r.L("type $dt struct{}")
	r.M("d *$dt")("Method")("a string", "b int", "").Returns("bool", "error", "").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
		r.L(`if a == $conv.Itoa(b) {`)
		r.L(`    return $ReturnZeroValues $errs.New("can't be'")`)
		r.L(`}`)
		r.N()
		r.L(`var @unique int`)
		r.L(`_ = $unique`)
		r.N()
		r.L(`return true, nil`)
	})

	z.N()
	z.R(`// just a random comment`)
	z.N()

	if err := m.Render(); err != nil {
		message.Fatal(errors.Wrap(err, "render modified file"))
	}
}
