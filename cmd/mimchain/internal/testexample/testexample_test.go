package testexample

import (
	"testing"

	"github.com/sirkon/errors"
	"github.com/sirkon/gogh"
)

func TestErrorRenderer(t *testing.T) {
	if err := doSomething(); err != nil {
		t.Error(err)
	}
}

func doSomething() error {
	prj, err := gogh.New(
		gogh.FancyFmt,
		func(r *gogh.Imports) *gogh.Imports {
			return r
		},
	)
	if err != nil {
		return errors.Wrap(err, "setup project renderer")
	}

	pkg, err := prj.Package("", "cmd/mimchain/internal/testexample")
	if err != nil {
		return errors.Wrap(err, "setup package usage")
	}

	r := pkg.Go("rendering_example.go", gogh.Autogen("testexample"))
	r.SetReturnZeroValues("")

	r.F("newExample")().Returns("error").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
		R(r).New("something failed").Str("something", gogh.Q("value"))
	})

	r.F("newfExample")().Returns("error").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
		R(r).Newf("%s %d", gogh.Q("error"), 1)
	})

	r.F("wrapExample")().Returns("error").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
		R(r).Wrap("newExample()", "call new")
	})

	r.F("wrapfExample")().Returns("error").Body(func(r *gogh.GoRenderer[*gogh.Imports]) {
		R(r).Wrapf("newExample()", "call %s", gogh.Q("new"))
	})

	if err := prj.Render(); err != nil {
		return errors.Wrap(err, "render generated source code")
	}

	return nil
}
