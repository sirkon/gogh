package main

import (
	"context"

	"github.com/sirkon/errors"
	"github.com/sirkon/gogh"
	"github.com/sirkon/message"
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

	p, err := m.Current("")
	if err != nil {
		message.Fatal(errors.Wrap(err, "open current package"))
	}

	r, err := p.Reuse("main.go")
	if err != nil {
		message.Fatal(errors.Wrap(err, "reuse package"))
	}

	r.Imports().Add("context").Ref("ctx")
	r.Imports().Add("github.com/sirkon/errors")

	r.R(`// Hello World!`)
	r.N()
	r.L(`func _($ctx.Context) {}`)

	r = p.Go("add.go", gogh.Autogen("reuse"))

	r.Imports().Add("fmt")
	r.L(`func __() {`)
	r.L(`    fmt.Println("Hello World!")`)
	r.L(`}`)

	if err := m.Render(); err != nil {
		message.Fatal(errors.Wrap(err, "render modified file"))
	}
}

// Hello World!

func _(context.Context) {}

// Hello World!

func _(context.Context) {}

// Hello World!

func _(context.Context) {}
