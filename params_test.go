package gogh_test

import (
	"fmt"

	"github.com/sirkon/gogh"
)

func ExampleParams() {
	var p gogh.Params
	fmt.Println(p)
	p.Append("ctx", "context.Context")
	p.Append("payload", "...interface{}")
	fmt.Println(p)

	// Output:
	//
	// ctx context.Context, payload ...interface{}
}

func ExampleMultiline() {
	var p gogh.Params
	fmt.Println(p)
	p.Append("ctx", "context.Context")
	fmt.Println(p.Multiline())
	p.Append("payload", "...interface{}")
	fmt.Println(p.Multiline())

	// Output:
	//
	//
	// ctx context.Context
	//
	// ctx context.Context,
	// payload ...interface{},
	//
}
