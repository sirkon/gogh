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
	"github.com/sirkon/protoast/ast"
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

	cmt                 *bytes.Buffer
	vals                *valScope
	blocksmgr           *blocks.Manager
	uniqs               map[string]struct{}
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
	defer handlePanic()
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
		case ast.Type:
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
	defer handlePanic()
	r.imports.Imports().pushImports()
	r.renderLine(r.last(), line, a...)
	r.newline()
}

// R puts raw text without formatting into the buffer.
func (r *GoRenderer[T]) R(line string) {
	defer handlePanic()
	r.imports.Imports().pushImports()
	r.last().WriteString(line)
	r.newline()
}

// S same as L but returns string instead of buffer write.
func (r *GoRenderer[T]) S(line string, a ...any) string {
	defer handlePanic()
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
		panic(errors.Newf("attempt to change context constant for %s to a different value", name))
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
func (r *GoRenderer[T]) Proto(t ast.Type) ProtocType {
	switch v := t.(type) {
	case *ast.Int32:
		return raw("int32")
	case *ast.Int64:
		return raw("int64")
	case *ast.Uint32:
		return raw("uint32")
	case *ast.Uint64:
		return raw("uint64")
	case *ast.Float32:
		return raw("float32")
	case *ast.Float64:
		return raw("float64")
	case *ast.Bool:
		return raw("bool")
	case *ast.Bytes:
		return raw("[]byte")
	case *ast.String:
		return raw("string")
	case *ast.Enum:
		if v.ParentMsg == nil {
			var alias string
			if !r.isInSamePackage(v) {
				alias = r.imports.Add(v.File.GoPath).push()
			}
			return ProtocType{
				source:   alias,
				selector: Proto(v.Name),
			}
		}
		res := r.Proto(v.ParentMsg)
		res.pointer = false
		res.selector += "_" + Proto(v.Name)
		return res
	case *ast.Repeated:
		return raw("[]" + r.Proto(v.Type).Impl())
	case *ast.Map:
		return raw("map[" + r.Proto(v.KeyType).Impl() + "]" + r.Proto(v.ValueType).Impl())
	case *ast.Message:
		// если это гугловые врапперы, то для них своя процедура
		if v.File.Name == "google/protobuf/wrappers.proto" {
			// ура, ето врапперы!
			r.imports.Add("google.golang.org/protobuf/Protos/known/wrapperspb").Ref("wrappers")
			switch v.Name {
			case "DoubleValue", "FloatValue", "Int64Value", "UInt64Value",
				"Int32Value", "UInt32Value", "BoolValue", "StringValue", "BytesValue":
			default:
				panic(errors.Newf("unsupported google wrapper %s.%s", v.File.Package, v.Name))
			}
			return ProtocType{
				pointer:  true,
				source:   r.S("$wrappers"),
				selector: Proto(v.Name),
			}
		}

		if v.ParentMsg == nil {
			var alias string
			if !r.isInSamePackage(v) {
				alias = r.imports.Add(v.File.GoPath).push()
			}
			return ProtocType{
				pointer:  true,
				source:   alias,
				selector: Proto(v.Name),
			}
		}
		res := r.Proto(v.ParentMsg)
		res.selector += "_" + Proto(v.Name)
		return res
	case *ast.Any:
		r.imports.Add("google.golang.org/protobuf/Protos/known/anypb").Ref("anypkg")
		return ProtocType{
			pointer:  true,
			source:   r.S("$anypkg"),
			selector: "Any",
		}
	default:
		panic(errors.Newf("Proto %T is not supported", v))
	}
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

// isInSamePackage определяет, относится ли генерируемый файл к тому же пакету, что и данный тип сгенерированный protoc-gen-go
func (r *GoRenderer[T]) isInSamePackage(t ast.Unique) bool {
	reference := r.protocTypePkgPath(t)
	return reference == r.pkg.Path()
}

func (r *GoRenderer[T]) protocTypePkgPath(t ast.Unique) string {
	switch v := t.(type) {
	case *ast.File:
		return v.GoPath
	case *ast.Service:
		return v.File.GoPath
	case *ast.Method:
		return v.File.GoPath
	case *ast.Message:
		return v.File.GoPath
	case *ast.Enum:
		return v.File.GoPath
	case *ast.OneOf:
		return v.ParentMsg.File.GoPath
	case *ast.OneOfBranch:
		return v.ParentOO.ParentMsg.File.GoPath
	default:
		return ""
	}
}

func handlePanic() {
	r := recover()
	if r == nil {
		return
	}

	frame := getOuterFrame()
	if frame == nil {
		// что-то странное
		panic(r)
	}

	message.Errorf("%s:%d %s", frame.File, frame.Line, r)
	panic(r)
}

func getOuterFrame() *runtime.Frame {
	stack := assembleWholeFrame(32)

	var wasFormattingFrame bool

	for {
		frame, ok := stack.Next()
		isFormattingFrame := isInternalStuff(frame.File)
		if wasFormattingFrame && !isFormattingFrame {
			return &frame
		}
		wasFormattingFrame = isFormattingFrame
		if !ok {
			// все пакеты внутри — наверное это тестирование!
			return &frame
		}
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

func isInternalStuff(path string) bool {
	if strings.Index(path, goghPkg) >= 0 {
		return true
	}

	if strings.Index(path, goFormatPkg) >= 0 {
		return true
	}

	return false
}
