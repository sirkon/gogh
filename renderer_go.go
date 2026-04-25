package gogh

import (
	"bufio"
	"bytes"
	"fmt"
	"go/types"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirkon/errors"
	"github.com/sirkon/go-format/v2"
	"github.com/sirkon/message"
	"github.com/sirkon/protoast/v2"
	"github.com/sirkon/protoast/v2/past"
	"golang.org/x/exp/maps"

	"github.com/sirkon/gogh/internal/blocks"
)

// GoRenderer GoFile source file code generation.
//
// The text data it used for code rendering is kept
// in a sequence of text blocksmgr, where the renderer
// instance reference one of them, it is called
// a current block for the renderer.
//
// Renderer also provides means to control import
// statements.
//
// Overall, you can:
//   - Add new import paths.
//   - Append a text to the current block of the renderer.
//   - Insert a new text block after the current one
//     and make the current block switched to it.
//     Read Z method docs to learn what it gives.
//
// The generated text consists of two major parts:
//  1. Auto generated header with file comment,
//     package statement and import statements.
//  2. A concatenated text from an ordered sequence
//     of text blocksmgr.
//
// With the GoRenderer you can:
type GoRenderer[T Importer] struct {
	name    string
	pkg     *Package[T]
	imports T
	options []RendererOption

	linebuf             bytes.Buffer
	cmt                 *bytes.Buffer
	vals                *valScope
	blocksmgr           *blocks.Manager
	uniqs               map[string]struct{}
	uniqTags            map[any]string
	preImport           map[string]struct{}
	reuse               bool
	reuseFirstImportPos int
}

// GoRendererBuffer switches the given renderer to a new
// block two times and returns a buffer of the block that
// was the current after the first switch.
//
// See what is happening here:
//   - B is a current block before the call.
//   - A is a current block after the first switch.
//   - C is a current block after the second switch.
//
// And
//
//	Original blocks:  …, B₋, B, B₊, …
//	First switch:     …, B₋, B, A, B₊, …
//	Second switch:    …, B₋, B, A, C, B₊, …
//
// We could actually do only one switch and return a black
// that was the current before the switch, but it can be
// pretty unsafe, becaue:
//
//   - A user can mutate buffer data by an accident.
//   - Contents of blocks is always concatenated with LF
//     between them. The usage of the dedicated block
//     ensures the user is not needed to care about new lines.
//
// This double switch makes it sure we are safe from these
// sorts of issues.
//
// This function aimed for an external usage. mimchain output
// uses this BTW.
func GoRendererBuffer[T Importer](r *GoRenderer[T]) *bytes.Buffer {
	res := r.blocksmgr.Insert().Data()
	r.blocksmgr.Insert()
	return res
}

// Imports returns imports controller.
//
// Usage example:
//
//	r.Import().Add("errors").Manager("errs")
//	r.L(`    return $errs.New("error")`)
//
// Will render:
//
//	return errors.New("error")
//
// Remember, using Manager to put package name into
// the scope is highly preferable over no Manager or
// setting package name manually (via the As call):
// It will take care of conflicting package names,
// you won't need to resolve dependencies manually.
//
// Beware though: do not use the same Manager name for
// different packages and do not try to Manager with
// the name you have used with Let before.
func (r *GoRenderer[T]) Imports() T {
	return r.imports
}

// N puts the new line character into the buffer.
func (r *GoRenderer[T]) N() {
	defer r.handlePanic()
	r.imports.Imports().pushImports()
	r.newline()
}

// C concatenates given objects into a single text line using
// space character as a separator.
func (r *GoRenderer[T]) C(a ...any) {
	b := r.last()
	for i, p := range a {
		if i > 0 {
			b.WriteByte(' ')
		}
		switch v := p.(type) {
		case string:
			b.WriteString(v)
		case fmt.Stringer:
			b.WriteString(v.String())
		case *Commas:
			b.WriteString(v.String())
		case *Params:
			b.WriteString(v.String())
		case types.Type:
			b.WriteString(r.Type(v))
		case types.Object:
			b.WriteString(r.Object(v))
		case past.Type:
			b.WriteString(r.Proto(v).String())
		default:
			b.WriteString(fmt.Sprint(p))
		}
	}
	r.newline()
}

// L renders text line using given [format] and puts it
// into the buffer.
//
// Usage example:
//
//	r.Let("dst", "buf")
//	r.L(`$dst = append($dst, $0)`, 12)
//
// Will render:
//
//	buf = append(buf, 12)
//
// [format]: https://github.com/sirkon/go-format
func (r *GoRenderer[T]) L(line string, a ...any) {
	defer r.handlePanic()
	r.imports.Imports().pushImports()
	r.renderLine(r.last(), line, a...)
	r.newline()
}

// R puts raw text without formatting into the buffer.
func (r *GoRenderer[T]) R(line string) {
	defer r.handlePanic()
	r.imports.Imports().pushImports()
	r.last().WriteString(line)
	r.newline()
}

// S same as L but returns string instead of buffer write.
func (r *GoRenderer[T]) S(line string, a ...any) string {
	defer r.handlePanic()
	r.imports.Imports().pushImports()
	var res bytes.Buffer
	r.renderLine(&res, line, a...)

	return res.String()
}

// Uniq is used to generate unique names, to avoid variables names
// clashes in the first place. This is how it works:
//
//	r.Uniq("name")        // name
//	r.Uniq("name")        // name1
//	r.Uniq("name")        // name2
//	r.Uniq("name", "alt") // nameAlt
//	r.Uniq("name", "alt") // name3
//	r.Uniq("name", "opt") // nameOpt
//
// Remember, Uniq's name and Let's key have nothing in common.
func (r *GoRenderer[T]) Uniq(name string, optSuffix ...string) string {
	if _, ok := r.uniqs[name]; !ok {
		r.uniqs[name] = struct{}{}
		return name
	}

	if len(optSuffix) > 0 {
		try := name + Public(optSuffix[0])
		if _, ok := r.uniqs[try]; !ok {
			r.uniqs[try] = struct{}{}
			return try
		}
	}

	for i := 1; i < math.MaxInt; i++ {
		n := name + strconv.Itoa(i+1)
		if _, ok := r.uniqs[n]; !ok {
			r.uniqs[n] = struct{}{}
			return n
		}
	}

	panic(errors.Newf("cannot find scope unique name for given base '%s'", name))
}

// BindUniq creates unique value (same as Uniq) and then binds it to the given "tag".
// Formatting will use that bound unique value instead of tag representation.
func (r *GoRenderer[T]) UniqBind(tag any, name string, optSuffix ...string) string {
	val := r.Uniq(name, optSuffix...)
	r.uniqTags[tag] = val

	return val
}

// Taken checks if the given unique name has been taken before.
func (r *GoRenderer[T]) Taken(name string) bool {
	_, ok := r.uniqs[name]
	return ok
}

// Let adds a named constant into the scope of the renderer.
// It will panic if you will try to set a different value
// for the name that exists in the current scope.
func (r *GoRenderer[T]) Let(name string, value any) {
	if strings.TrimSpace(name) == "" {
		panic(errors.New("context name must not be empty or white spaced only"))
	}

	if r.vals.CheckScope(name) {
		panic(errors.Newf("attempt to change context constant %q to a different value", name))
	}

	r.letSet(name, r.ctxValue(value))
}

// SetReturnZeroValues adds a named constant with the ReturnZeroValues name
// whose role is to represent zero return values in functions.
//
// Usage example:
//
//	r.Imports.Add("io").Manager("io")
//	r.Imports.Add("errors").Manager("errs")
//	r.F("file")("name", "string").Returns("*$io.ReadCloser", "error", "").Body(func(r *Go) {
//	    r.L(`// Look at trailing comma, it is important ... $ReturnZeroValues`)
//	    r.L(`return $ReturnZeroValues $errs.New("error")`)
//	})
//
// Output:
//
//	func file(name string) (io.ReadCloser, error) {
//	    // Look at trailing comma, it is important ... nil,
//	    return nil, errors.New("error"
//	}
//
// Take a look at the doc to know more about how results and parameters can be set up.
//
// This example may look weird and actually harder to write than a simple formatting,
// but it  makes a sense in fact when we work upon the existing source code, with
// these types.Type everywhere. You don't even need to set up this constant manually
// with them BTW, it will be done for you based on return types provided by the Returns
// call itself.
//
// This value can be overriden BTW.
func (r *GoRenderer[T]) SetReturnZeroValues(values ...string) {
	r.letSet(ReturnZeroValues, A(values...))
}

// TryLet same as Let but without a panic, it just exits
// when the variable is already there.
func (r *GoRenderer[T]) TryLet(name string, value any) {
	if strings.TrimSpace(name) == "" {
		panic(errors.New("context name must not be empty or white spaced only"))
	}

	if r.vals.CheckScope(name) {
		return
	}

	r.letSet(name, value)
}

func (r *GoRenderer[T]) letSet(name string, value any) {
	switch vv := value.(type) {
	case string:
		value = casesFormatter{value: vv}
	case fmt.Stringer:
		value = casesFormatter{value: vv.String()}
	default:
	}

	r.vals.Set(name, value)
}

// InCtx checks if this name is already in the rendering context.
func (r *GoRenderer[T]) InCtx(name string) bool {
	_, ok := r.vals.Get(name)
	return ok
}

// Scope returns a new renderer with a scope inherited from the original.
// Any scope changes made with this renderer will not reflect into the
// scope of the original renderer.
func (r *GoRenderer[T]) Scope() (res *GoRenderer[T]) {
	defer func() {
		r.pkg.addRenderer(res)
	}()

	return &GoRenderer[T]{
		name:      r.name,
		pkg:       r.pkg,
		imports:   r.imports,
		vals:      r.vals.Next(),
		blocksmgr: r.blocksmgr,
		uniqs:     maps.Clone(r.uniqs),
		uniqTags:  maps.Clone(r.uniqTags),
	}
}

// InnerScope creates a new scope and feeds it into the given function.
func (r *GoRenderer[T]) InnerScope(f func(r *GoRenderer[T])) {
	f(r.Scope())
}

// Z provides a renderer instance of "laZy" writing.
//
// What it does:
//  1. Inserts a new text block and switches the current
//     renderer to it.
//  2. Return a new renderer which references a block
//     which was the current before.
//
// So, with this renderer you will write into the previous
// "current", while the original renderer will write into
// the next. This means you will have text rendered
// with the returned GoRenderer instance will appear
// before the one made with the original renderer after
// the Z call. Even if writes with the original were made
// before the writes with the returned.
//
// Example:
//
//	r.R(`// Hello`)
//	x := r.Z()
//	r.R(`// World!`)
//	x.R(`// 你好`)
//
// Output:
//
//	// Hello
//	// 你好
//	// World!
//
// See, even though we wrote Chinese("Hello") after the
// "World!" it appears before it after the rendering.
func (r *GoRenderer[T]) Z() (res *GoRenderer[T]) {
	defer func() {
		r.pkg.addRenderer(res)
	}()
	r.last()

	res = &GoRenderer[T]{
		name:      r.name,
		pkg:       r.pkg,
		imports:   r.imports,
		vals:      r.vals,
		blocksmgr: r.blocksmgr.Insert().Prev(),
		uniqs:     r.uniqs,
		uniqTags:  r.uniqTags,
	}

	return res
}

// T produces a temporary renderer which renders for the same package
// but will not save its content anywhere. It is meant to deal with
// side effects caused by Type, PkgObject, Object, Proto and alike –
// – they do imports for the file generated with this renderer.
func (r *GoRenderer[T]) T() *GoRenderer[T] {
	return r.pkg.Void()
}

// Type renders fully qualified type name based on go/types representation.
// You don't need to care about importing a package this type defined in
// or to use package name to access a type. This method will do this
// all.
//
// Beware though, the produced code may be incorrect if your type names
// are only used in strings or comments. You will have an import statement
// for them, but won't use them at the same time.
func (r *GoRenderer[T]) Type(t types.Type) string {
	switch v := t.(type) {
	case *types.Named:
		typ := v.Obj()
		pkg := typ.Pkg()
		if pkg == nil || pkg.Path() == r.pkg.Path() {
			return typ.Name()
		}
		alias := r.imports.Add(pkg.Path()).push()

		var res strings.Builder
		res.WriteString(alias)
		res.WriteByte('.')
		res.WriteString(typ.Name())
		if v.TypeParams().Len() != 0 {
			res.WriteByte('[')
			for i := 0; i < v.TypeParams().Len(); i++ {
				if i > 0 {
					res.WriteString(", ")
				}

				res.WriteString(v.TypeParams().At(i).Obj().Name())
				res.WriteByte(' ')
				res.WriteString(r.Type(v.TypeParams().At(i).Obj().Type()))
			}
			res.WriteByte(']')
		}

		return res.String()
	case *types.Pointer:
		return "*" + r.Type(v.Elem())
	case *types.Slice:
		return "[]" + r.Type(v.Elem())
	case *types.Interface:
		// Вообще, здесь может быть похитрее, но на практике мало кто использует нечто в духе `interface{ M() }`
		// в объявлениях параметров или возвращаемых значений, поэтому пока так. Но возможно придётся этим
		// заморачиваться
		return v.String()
	case *types.Struct:
		// аналогично предыдущему пункту
		return v.String()
	case *types.Basic:
		return v.String()
	case *types.Alias:
		typ := v.Obj()
		pkg := typ.Pkg()
		if pkg == nil || pkg.Path() == r.pkg.Path() {
			return typ.Name()
		}
		alias := r.imports.Add(pkg.Path()).push()

		var res strings.Builder
		res.WriteString(alias)
		res.WriteByte('.')
		res.WriteString(typ.Name())
		if v.TypeParams().Len() != 0 {
			res.WriteByte('[')
			for i := 0; i < v.TypeParams().Len(); i++ {
				if i > 0 {
					res.WriteString(", ")
				}

				res.WriteString(v.TypeParams().At(i).Obj().Name())
				res.WriteByte(' ')
				res.WriteString(r.Type(v.TypeParams().At(i).Obj().Type()))
			}
			res.WriteByte(']')
		}

		return res.String()
	case *types.Map:
		return fmt.Sprintf("map[%s]%s", r.Type(v.Key()), r.Type(v.Elem()))
	case *types.Signature:
		var args []string
		for i := 0; i < v.Params().Len(); i++ {
			p := v.Params().At(i)
			t := p.Type()
			if v.Variadic() && i == v.Params().Len()-1 {
				t = t.(*types.Slice).Elem()
				args = append(args, fmt.Sprintf("%s ...%s", p.Name(), r.Type(t)))
			} else {
				args = append(args, fmt.Sprintf("%s %s", p.Name(), r.Type(t)))
			}
		}
		var rets []string
		for i := 0; i < v.Results().Len(); i++ {
			v := v.Results().At(i)
			rets = append(rets, fmt.Sprintf("%s %s", v.Name(), r.Type(v.Type())))
		}
		return fmt.Sprintf("func (%s) (%s)", strings.Join(args, ", "), strings.Join(rets, ", "))
	case *types.Array:
		return fmt.Sprintf("[%d]%s", v.Len(), r.Type(v.Elem()))
	case *types.Chan:
		switch v.Dir() {
		case types.RecvOnly:
			return "<-chan " + r.Type(v.Elem())
		case types.SendOnly:
			return "chan<- " + r.Type(v.Elem())
		case types.SendRecv:
			return "chan " + r.Type(v.Elem())
		default:
			panic(errors.Newf("channel direction %v is not supported", v.Dir()))
		}
	default:
		panic(errors.Newf("type %T is not supported", t))
	}
}

// PkgObject renders fully qualified object name used with the referenced package.
// The reference can be done with one of:
//   - *types.Named.
//   - types.Object.
//   - *GoRenderer[T].
//   - string containing package path.
func (r *GoRenderer[T]) PkgObject(pkgRef any, name string) string {
	var pkg string
	switch v := pkgRef.(type) {
	case types.Object:
		pkg = v.Pkg().Path()
	case *types.Named:
		pkg = v.Obj().Pkg().Path()
	case *GoRenderer[T]:
		pkg = v.pkg.Path()
	case string:
		pkg = v
	default:
		panic(errors.Newf("type %T cannot reference a package", pkgRef))
	}

	if pkg != r.pkg.Path() {
		r = r.Scope()
		r.Imports().Add(pkg).Ref("packageReference")
		return r.S("$packageReference.$0", name)
	}

	return name
}

// Object renders fully qualified object name.
func (r *GoRenderer[T]) Object(item types.Object) string {
	pkg := item.Pkg().Path()
	if pkg != r.pkg.Path() {
		r = r.Scope()
		r.Imports().Add(pkg).Ref("packageReference")
		return r.S("$packageReference.$0", item.Name())
	}

	return item.Name()
}

// Proto renders protoc-gen-go generated name based on [protoast] protobuf types representation.
// Provides the same guarantees as Type, i.e. imports, package qualifiers, etc.
//
// [protoast]: https://github.com/sirkon/protoast/tree/master/ast
func (r *GoRenderer[T]) Proto(t past.Type) ProtocType {
	switch v := t.(type) {
	case *past.Int32, *past.Sint32, *past.Sfixed32:
		return raw("int32")
	case *past.Int64, *past.Sint64, *past.Sfixed64:
		return raw("int64")
	case *past.Uint32, *past.Fixed32:
		return raw("uint32")
	case *past.Uint64, *past.Fixed64:
		return raw("uint64")
	case *past.Float:
		return raw("float32")
	case *past.Double:
		return raw("float64")
	case *past.Bool:
		return raw("bool")
	case *past.Bytes:
		return raw("[]byte")
	case *past.String:
		return raw("string")
	case *past.Enum:
		parent := r.protoRegistry().NodeParent(v)
		switch p := parent.(type) {
		case *past.Message:
			res := r.Proto(p)
			res.pointer = false
			res.selector += "_" + Proto(v.Name())
			return res
		case *past.File:
			var alias string
			if !r.isInSamePackage(v) {
				alias = r.Imports().Add(r.protocTypePkgPath(v)).push()
			}
			return ProtocType{
				source:   alias,
				selector: Proto(v.Name()),
			}
		default:
			panic(errors.Newf(
				"past not with a tie to %T when %T or %T are the only valid options",
				p, new(past.Message), new(past.File),
			))
		}
	case *past.EnumValue:
		res := r.Proto(r.protoRegistry().NodeParent(v).(*past.Enum))
		res.selector += "_" + v.Name()
		return res
	case *past.Repeated:
		return raw("[]" + r.Proto(v.Type).Impl())
	case *past.Map:
		return raw("map[" + r.Proto(v.Key()).Impl() + "]" + r.Proto(v.Value(r.protoRegistry())).Impl())
	case *past.Message:
		file := r.protoRegistry().NodeFile(v)
		if file == nil {
			panic("no file for message " + v.Name())
		}
		// если это гугловые врапперы, то для них своя процедура
		if file.Name() == "google/protobuf/wrappers.proto" {
			// ура, ето врапперы!
			r.imports.Add("google.golang.org/protobuf/Protos/known/wrapperspb").Ref("wrappers")
			switch v.Name() {
			case "DoubleValue", "FloatValue", "Int64Value", "UInt64Value",
				"Int32Value", "UInt32Value", "BoolValue", "StringValue", "BytesValue":
			default:
				panic(errors.Newf("unsupported google wrapper %s.%s", file.Name(), v.Name()))
			}
			return ProtocType{
				pointer:  true,
				source:   r.S("$wrappers"),
				selector: Proto(v.Name()),
			}
		}
		parent := r.protoRegistry().NodeParent(v)
		switch p := parent.(type) {
		case *past.File:
			var alias string
			if !r.isInSamePackage(v) {
				alias = r.imports.Add(r.protocTypePkgPath(p)).push()
			}
			return ProtocType{
				pointer:  true,
				source:   alias,
				selector: Proto(v.Name()),
			}
		default:
			res := r.Proto(p.(past.Type))
			res.selector += "_" + Proto(v.Name())
		}
	default:
		panic(errors.Newf("Proto %T is not supported", v))
	}

	panic("unreachable")
}

// path returns generated file path
func (r *GoRenderer[T]) path() string {
	return filepath.Join(r.pkg.mod.root, r.pkg.rel, r.name)
}

// localPath returns file path within the module
func (r *GoRenderer[T]) localPath() string {
	return path.Join(r.pkg.Path(), r.name)
}

func (r *GoRenderer[T]) render() error {
	data := &bytes.Buffer{}

	if !r.reuse {
		for _, option := range r.options {
			if !option(r) {
				return nil
			}
		}

		if r.cmt != nil {
			_, _ = io.Copy(data, r.cmt)
			data.WriteString("\n")
		}

		data.WriteString("package ")
		data.WriteString(r.pkg.name)
		data.WriteString("\n\n")

		if len(r.imports.Imports().pkgs) > 0 {
			data.WriteString("import (")
			for pkgpath, alias := range r.imports.Imports().pkgs {
				name := r.imports.Imports().getPkgName(pkgpath)
				if name != alias {
					data.WriteString(alias)
					data.WriteByte(' ')
				}

				data.WriteByte('"')
				data.WriteString(pkgpath)
				data.WriteString(`"`)
				data.WriteByte('\n')
			}
			data.WriteString(")\n\n")
		}
	}

	for _, block := range r.blocksmgr.Collect() {
		_, _ = io.Copy(data, block)
	}

	if r.reuse && len(r.imports.Imports().pkgs) > 0 {
		var tmp bytes.Buffer
		s := bufio.NewScanner(data)
		var i int
		for s.Scan() {
			if i == r.reuseFirstImportPos {
				tmp.WriteString("import (\n")

				for pkgpath, alias := range r.imports.Imports().pkgs {
					if _, ok := r.preImport[pkgpath]; ok {
						continue
					}

					name := r.imports.Imports().getPkgName(pkgpath)
					if name != alias {
						tmp.WriteString(alias)
						tmp.WriteByte(' ')
					}

					tmp.WriteByte('"')
					tmp.WriteString(pkgpath)
					tmp.WriteString(`"`)
					tmp.WriteByte('\n')
				}

				tmp.WriteString(")\n\n")
			}

			tmp.Write(s.Bytes())
			tmp.WriteByte('\n')
			i++
		}

		data.Reset()
		_, _ = tmp.WriteTo(data)
	}

	res, err := r.pkg.mod.fmt(data.Bytes())
	if err != nil {
		message.Error(err)
		return errors.New("failed to format rendered file")
	}

	if err := os.WriteFile(r.path(), res, 0644); err != nil {
		return errors.Wrap(err, "write rendered file")
	}

	return nil
}

func (r *GoRenderer[T]) last() *bytes.Buffer {
	return r.blocksmgr.Data()
}

func (r *GoRenderer[T]) renderCtx() *format.ContextBuilder {
	res := format.NewContextBuilder()
	for name, value := range r.vals.Map() {
		res.Add(name, value)
	}

	return res
}

func (r *GoRenderer[T]) comment() *bytes.Buffer {
	if r.cmt == nil {
		r.cmt = &bytes.Buffer{}
	}

	return r.cmt
}

func (r *GoRenderer[T]) setVals(vals map[string]any) {
	for name := range vals {
		if r.vals.CheckScope(name) {
			panic(errors.Newf("attempt to '%s' into different value", name))
		}
	}
}

func (r *GoRenderer[T]) newline() {
	r.last().WriteByte('\n')
}

func (r *GoRenderer[T]) protoRegistry() *protoast.Registry {
	return r.pkg.mod.registry
}

// isInSamePackage определяет, относится ли генерируемый файл к тому же пакету, что и данный тип сгенерированный protoc-gen-go
func (r *GoRenderer[T]) isInSamePackage(t past.Node) bool {
	reference := r.protocTypePkgPath(t)
	return reference == r.pkg.Path()
}

// TODO probably a cache would make it a bit better.
func (r *GoRenderer[T]) protocTypePkgPath(t past.Node) string {
	registry := r.protoRegistry()
	for node := range registry.NodeHierarchy(t) {
		f, ok := node.(*past.File)
		if !ok {
			continue
		}

		option := registry.OptionNamed(f, "go_package")
		pkg := registry.GoPackageOption(option)
		if pkg == nil {
			panic("missing go_package option")
		}
		return pkg.Path
	}

	panic("orphan node without a file in its hierarchy ties")
}

func (r *GoRenderer[T]) handlePanic() {
	rr := recover()
	if rr == nil {
		return
	}
	if err := r.pkg.mod.bolt.Close(); err != nil {
		message.Warning(errors.Wrap(err, "failed to close bolt"))
	}

	frame := r.getOuterFrame()
	if frame == nil {
		// что-то странное
		panic(rr)
	}

	message.Errorf("%s:%d %s", frame.File, frame.Line, r)
	os.Exit(1)
}

func (r *GoRenderer[T]) getOuterFrame() *runtime.Frame {
	stack := assembleWholeFrame(32)

	var lastFrame *runtime.Frame

	for {
		frame, ok := stack.Next()
		if !ok {
			return lastFrame
		}
		lastFrame = new(frame)
		if r.isInternalStuff(frame.File) {
			continue
		}
		return lastFrame
	}
}

func assembleWholeFrame(startSize int) *runtime.Frames {
	for {
		pc := make([]uintptr, startSize)
		n := runtime.Callers(2, pc)
		if n == 0 {
			return nil
		}

		if n == startSize {
			startSize *= 2
			continue
		}

		pc = pc[:n]
		return runtime.CallersFrames(pc[:n])
	}
}

func (r *GoRenderer[T]) isInternalStuff(path string) bool {
	if pos := strings.Index(path, goghPkg); pos >= 0 {
		rest := path[pos+len(goghPkg):]
		if strings.IndexRune(rest, os.PathSeparator) < 0 {
			return true
		}
	}

	if strings.Index(path, goFormatPkg) >= 0 {
		return true
	}

	if strings.HasPrefix(path, r.pkg.mod.goroot) {
		return true
	}

	return false
}
