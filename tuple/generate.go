//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"strings"
)

const prelude = `

// WithContextAR returns a function with a context argument that
// calls f without the context and returns its result.
func WithContextAR[A, R any](f func(A) R) func(context.Context, A) R {
	return func(ctx context.Context, a A) R {
		return f(a)
	}
}

// WithContext returns a function with a context argument
// that calls f without the context.
func WithContextA[A any](f func(A)) func(context.Context, A) {
	return func(ctx context.Context, a A) {
		f(a)
	}
}

// WithErrorAR returns an error-returning function that
// calls f and returns a nil error.
func WithErrorAR[A, R any](f func(A) R) func(A) (R, error) {
	return func(a A) (R, error) {
		return f(a), nil
	}
}
`

var buf = new(bytes.Buffer)

const N = 7

func main() {
	generateTupleCode()
	code, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot format code: %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile("tuple-gen.go", code, 0666); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	buf.Reset()

	generateTupleFuncCode()
	code, err = format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot format code: %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile("tuplefunc/tuplefunc-gen.go", code, 0666); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func generateTupleCode() {
	P("// Code generated by tuple/generate.go. DO NOT EDIT.\n")
	P("")
	P("package tuple\n")
	for i := 0; i < N; i++ {
		generateTuple(i)
		P("\n")
	}
}

func generateTupleFuncCode() {
	P("// Code generated by tuple/generate.go. DO NOT EDIT.\n")
	P("\n")
	P("package tuplefunc\n")
	P(`
import (
	"context"

	"github.com/rogpeppe/generic/tuple"
)
`)

	P("%s", prelude)
	P("\n")
	P("\n")
	generate(generateAFunc)
	generate(generateRFunc)
	generate(generateARFunc)
	generate(generateAEFunc)
	generate(generateAREFunc)
	generate(generateREFunc)
	generate(generateCAREFunc)
	// TODO
	//	ARE		argument and return with error
	//	CAE		context with argument; only error return
	//	CRE		context only; return with error
	//	CA		context with argument; no return
	//	CR		context only and return
	//	CAR		context, argument and return
}

func generate(f func(a, r int)) {
	for a := 0; a < N; a++ {
		for r := 0; r < N; r++ {
			blen := buf.Len()
			f(a, r)
			if buf.Len() > blen {
				P("\n")
			}
		}
	}
}

func P(format string, args ...interface{}) {
	fmt.Fprintf(buf, format, args...)
}

func generateTuple(n int) {
	switch {
	case n == 1:
		P("// There is no 1-tuple - a 1-tuple is represented by the type itself.\n")
		return
	case n == 0:
		P("// T%d holds a tuple of %d values.\n", n, n)
		P("type T0 = struct{}\n")
		return
	}
	P("// T%d holds a tuple of %d values.\n", n, n)
	P("type T%d%s struct {\n", n, typeParams(n, 0))
	for i := 0; i < n; i++ {
		P("\tA%d A%d\n", i, i)
	}
	P("}\n")
	P("\n")
	P("// T returns all the tuple's values.\n")
	P("func (t T%d[%s]) T() %s {\n",
		n,
		commaSep("A", n),
		retTypes("A", n, false),
	)
	P("\treturn %s\n", commaSep("t.A", n))
	P("}\n")
	P("\n")
	P("// MkT%d returns a %d-tuple formed from its arguments.\n", n, n)
	P("func MkT%d[%s any](%s) T%d[%s] {\n",
		n,
		commaSep("A", n),
		argParams(n),
		n,
		commaSep("A", n),
	)
	P("\treturn T%d[%s]{%s}\n", n, commaSep("A", n), commaSep("a", n))
	P("}\n")
}

func generateARFunc(a, r int) {
	name := fmt.Sprintf("ToAR_%d_%d", a, r)
	P("// %s returns a single-argument, single-return function that calls f.\n", name)
	P("func %s%s(f func(%s) %s) func(%s) %s {\n",
		name,
		typeParams(a, r),
		argParams(a),
		retTypes("R", r, false),
		tuple("A", a),
		tuple("R", r),
	)
	P("\treturn func(a %s) %s {\n",
		tuple("A", a),
		tuple("R", r),
	)
	expr := fmt.Sprintf("f(%s)", argTuple("a", a))
	switch {
	case r == 0:
		P("\t\t%s\n", expr)
		P("\t\treturn struct{}{}\n")
	case r == 1:
		P("\t\treturn %s\n", expr)
	default:
		P("\t\treturn tuple.MkT%d(%s)\n", r, expr)
	}
	P("\t}\n")
	P("}\n")
}

func generateAFunc(a, r int) {
	if r != 0 {
		return
	}
	if a == 1 {
		// No need - it's already in the correct form.
		return
	}
	name := fmt.Sprintf("ToA_%d_%d", a, r)
	P("// %s returns a single-argument function that calls f.\n", name)
	P("func %s%s(f func(%s)) func(%s) {\n",
		name,
		typeParams(a, r),
		argParams(a),
		tuple("A", a),
	)
	P("\treturn func(a %s) {\n",
		tuple("A", a),
	)
	P("\t\tf(%s)\n", argTuple("a", a))
	P("\t}\n")
	P("}\n")
}

func generateRFunc(a, r int) {
	if a != 0 {
		return
	}
	if r == 1 {
		// No need - it's already in the correct form.
		return
	}
	name := fmt.Sprintf("ToR_%d_%d", a, r)
	P("// %s returns a single-return function that calls f.\n", name)
	P("func %s%s(f func() %s) func() %s {\n",
		name,
		typeParams(a, r),
		retTypes("R", r, false),
		tuple("R", r),
	)
	P("\treturn func() %s {\n",
		tuple("R", r),
	)
	switch {
	case r == 0:
		P("\t\tf()\n")
		P("\t\treturn struct{}{}\n")
	case r == 1:
		P("\t\treturn f()\n")
	default:
		P("\t\treturn tuple.MkT%d(f())\n", r)
	}
	P("\t}\n")
	P("}\n")
}

func generateAEFunc(a, r int) {
	if r != 0 {
		return
	}
	if a == 1 {
		// No need - it's already in the correct form.
		return
	}
	name := fmt.Sprintf("ToAE_%d_%d", a, r)
	P("// %s returns a single-argument function that calls f.\n", name)
	P("func %s%s(f func(%s) error) func(%s) error {\n",
		name,
		typeParams(a, r),
		argParams(a),
		tuple("A", a),
	)
	P("\treturn func(a %s) error {\n",
		tuple("A", a),
	)
	P("\t\treturn f(%s)\n", argTuple("a", a))
	P("\t}\n")
	P("}\n")
}

func generateAREFunc(a, r int) {
	name := fmt.Sprintf("ToARE_%d_%d", a, r)
	P("// %s returns a single-argument, single-return-with-error function that calls f.\n", name)
	P("func %s%s(f func(%s) %s) func(%s) (%s, error) {\n",
		name,
		typeParams(a, r),
		argParams(a),
		retTypes("R", r, true),
		tuple("A", a),
		tuple("R", r),
	)
	P("\treturn func(a %s) (%s, error) {\n",
		tuple("A", a),
		tuple("R", r),
	)
	expr := fmt.Sprintf("f(%s)", argTuple("a", a))
	switch {
	case r == 0:
		P("\t\terr := %s\n", expr)
		P("\t\treturn struct{}{}, err\n")
	case r == 1:
		P("\t\treturn %s\n", expr)
	default:
		P("\t\t%s, err := %s\n", commaSep("r", r), expr)
		P("\t\treturn tuple.MkT%d(%s), err\n", r, commaSep("r", r))
	}
	P("\t}\n")
	P("}\n")
}

func generateREFunc(a, r int) {
	if a != 0 {
		return
	}
	if r == 1 {
		// No need - it's already in the correct form.
		return
	}
	name := fmt.Sprintf("ToRE_%d_%d", a, r)
	P("// %s returns a single-return-with-error function that calls f.\n", name)
	P("func %s%s(f func() %s) func() (%s, error) {\n",
		name,
		typeParams(a, r),
		retTypes("R", r, true),
		tuple("R", r),
	)
	P("\treturn func() (%s, error) {\n",
		tuple("R", r),
	)
	expr := "f()"
	switch {
	case r == 0:
		P("\t\terr := %s\n", expr)
		P("\t\treturn struct{}{}, err\n")
	case r == 1:
		P("\t\treturn %s\n", expr)
	default:
		P("\t\t%s, err := %s\n", commaSep("r", r), expr)
		P("\t\treturn tuple.MkT%d(%s), err\n", r, commaSep("r", r))
	}
	P("\t}\n")
	P("}\n")
}

func generateCAREFunc(a, r int) {
	name := fmt.Sprintf("ToCARE_%d_%d", a, r)
	P("// %s returns a context-with-single argument, single-return-with-error function that calls f.\n", name)
	P("func %s%s(f func(%s) %s) func(context.Context, %s) (%s, error) {\n",
		name,
		typeParams(a, r),
		argParamsWithContext(a),
		retTypes("R", r, true),
		tuple("A", a),
		tuple("R", r),
	)
	P("\treturn func(ctx context.Context, a %s) (%s, error) {\n",
		tuple("A", a),
		tuple("R", r),
	)
	var expr string
	switch a {
	case 0:
		expr = ""
	case 1:
		expr = ", a"
	default:
		expr = ", " + commaSep("a.A", a)
	}
	expr = fmt.Sprintf("f(ctx%s)", expr)
	switch {
	case r == 0:
		P("\t\terr := %s\n", expr)
		P("\t\treturn struct{}{}, err\n")
	case r == 1:
		P("\t\treturn %s\n", expr)
	default:
		P("\t\t%s, err := %s\n", commaSep("r", r), expr)
		P("\t\treturn tuple.MkT%d(%s), err\n", r, commaSep("r", r))
	}
	P("\t}\n")
	P("}\n")
}

func argTuple(argName string, n int) string {
	switch n {
	case 0:
		return ""
	case 1:
		return argName
	}
	return argName + ".T()"
}

func typeParams(a, r int) string {
	if a+r == 0 {
		return ""
	}
	p := make([]string, 0, a+r)
	for i := 0; i < a; i++ {
		p = append(p, enum("A", i, a))
	}
	for i := 0; i < r; i++ {
		p = append(p, enum("R", i, r))
	}
	return "[" + strings.Join(p, ", ") + " any]"
}

func argParams(a int) string {
	p := make([]string, 0, a)
	for i := 0; i < a; i++ {
		p = append(p, fmt.Sprintf("%s %s", enum("a", i, a), enum("A", i, a)))
	}
	return strings.Join(p, ", ")
}

func argParamsWithContext(a int) string {
	if a == 0 {
		return "ctx context.Context"
	}
	return "ctx context.Context, " + argParams(a)
}

func tuple(category string, n int) string {
	var t string
	switch n {
	case 0:
		t = "T0"
	case 1:
		return category
	default:
		t = fmt.Sprintf("T%d[%s]", n, commaSep(category, n))
	}
	pkgQual := "tuple."
	return pkgQual + t
}

func retTypes(prefix string, n int, withError bool) string {
	items := make([]string, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, enum(prefix, i, n))
	}
	if withError {
		items = append(items, "error")
	}
	s := strings.Join(items, ", ")
	if len(items) > 1 {
		return "(" + s + ")"
	}
	return s
}

func commaSep(prefix string, n int) string {
	items := make([]string, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, enum(prefix, i, n))
	}
	return strings.Join(items, ", ")
}

func enum(prefix string, i int, n int) string {
	if n <= 0 {
		panic("no types")
	}
	if n == 1 {
		return prefix
	}
	return fmt.Sprintf("%s%d", prefix, i)
}