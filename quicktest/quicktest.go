/// Package quicktest implements assertion and other helpers wrapped
// around the standard library's testing types.
//package quicktest
package quicktest

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// Checker is implemented by types used as part of Check/Assert invocations.
// The type parameter will be the type of the first argument passed
// to Check or Assert.
type Checker[T any] interface {
	// Check checks that the provided argument passes the check.
	// On failure, the returned error is printed along with
	// the checker arguments (obtained by calling ArgNames and Args)
	// and key-value pairs added by calling the note function.
	//
	// If Check returns ErrSilent, neither the checker arguments nor
	// the error are printed; values with note are still printed.
	Check(got T, note func(key string, value interface{})) error

	// ArgNames returns the arguments passed to the checker.
	ArgNames() []string

	// Args returns the arguments corresponding to ArgNames.
	Args() []interface{}
}

// Assert checks that the provided argument passes the given check
// and calls tb.Error otherwise, including any Comment arguments
// in the failure.
func Assert[T any](tb testing.TB, got T, op Checker[T], comment ...Comment) {
	if !Check(tb, got, op, comment...) {
		tb.FailNow()
	}
}

type Comment struct {
	format string
	args   []interface{}
}

// String outputs a string formatted according to the stored format specifier
// and args.
func (c Comment) String() string {
	return fmt.Sprintf(c.format, c.args...)
}

// Check checks that the provided argument passes the given check
// and calls tb.Fatal otherwise, including any Comment arguments
// in the failure.
func Check[T any](tb testing.TB, got T, op Checker[T], comment ...Comment) bool {
	var notes []string
	note := func(key string, value interface{}) {
		notes = append(notes, fmt.Sprintf("%s: %v", key, value))
	}
	err := op.Check(got, note)
	if err == nil {
		return true
	}
	tb.Errorf("assertion failed: %v; notes %v", err, notes)
	return false
}

// Patch sets *dest to value, then adds a Cleanup
// to tb that will set it back to its original value at the
// end of the test.
func Patch[T any](tb testing.TB, dest *T, value T) {
	old := *dest
	*dest = value
	tb.Cleanup(func() {
		*dest = old
	})
}

// IsZero checks that a value is the zero value for its type.
// Maybe this might be better with a shorter name like Z.
func IsZero[T any]() Checker[T] {
	var z T
	return DeepEquals[T](z)
}

// Equals checks that the argument is equal to want.
func Equals[T comparable](want T) Checker[T] {
	return equalsChecker[T]{
		argNames: []string{"got", "want"},
		want:     want,
	}
}

type equalsChecker[T comparable] struct {
	argNames
	want T
}

func (c equalsChecker[T]) Args() []interface{} {
	return []interface{}{c.want}
}

func (c equalsChecker[T]) Check(got T, note func(key string, value interface{})) error {
	if got != c.want {
		return errors.New("values are not equal")
	}
	return nil
}

//// Contains returns a checker that checks whether
//// a slice contains the given element.
//func Contains[T comparable](want T) Checker[[]T] {
//	return Any(Equals(want))
//}

//// StrContains returns a checker that checks whether
//// a string contains the given sub-string.
//func StrContains(substr string) Checker[string] {
//}
//
//// Any returns a checker that uses c to check elements
//// in a slice. It succeeds if any element passes the check.
//func Any[T any](c Checker[T]) Checker[[]T]
//
//// AnyMapValue returns a checker that uses c to check the
//// value elements in a map. It succeeds if any value
//// passes the check.
//func AnyMapValue[Key comparable, Value any](c Checker[Value]) Checker[map[Key]Value]
//
//
//func CmpEquals[T any](opts ...cmpOption) func(want T) Checker[T]

func DeepEquals[T any](want T) Checker[T] {
	return deepEqualsChecker[T]{
		argNames: []string{"got", "want"},
		want:     want,
	}
}

type deepEqualsChecker[T any] struct {
	argNames
	want T
}

func (c deepEqualsChecker[T]) Args() []interface{} {
	return []interface{}{c.want}
}

func (c deepEqualsChecker[T]) Check(got T, note func(key string, value interface{})) error {
	// TODO use go-cmp
	if !reflect.DeepEqual(got, c.want) {
		return errors.New("values are not equal")
	}
	return nil
}

// cmpOption represents the cmp.Option type from the github.com/google/go-cmp/cmp
// package.
type cmpOption struct {
}

//type Data interface {
//	type []byte, string
//}
//
//func JSONEquals(want interface{}) Checker[[]byte]
//
//func ErrorMatches(pattern string) Checker[error]
//
//func Matches(pattern string) Checker[string]
//
//func StringerMatches(pattern string) Checker[interface{ String() string }]
//
//
//func Satisfies[T any](f func(T) bool) Checker[T]

// argNames helps implementing Checker.ArgNames.
type argNames []string

// ArgNames implements Checker.ArgNames by returning the argument names.
func (a argNames) ArgNames() []string {
	return a
}

func Not[T any](c Checker[T]) Checker[T] {
	return notChecker[T]{
		checker: c,
	}
}

type notChecker[T any] struct {
	checker Checker[T]
}

func (c notChecker[T]) ArgNames() []string {
	return c.checker.ArgNames()
}

func (c notChecker[T]) Args() []interface{} {
	return c.checker.Args()
}

func (c notChecker[T]) Check(got T, note func(key string, value interface{})) error {
	if err := c.checker.Check(got, note); err == nil {
		return fmt.Errorf("unexpected success")
	}
	return nil
}
