// Package tuplefunc provides functions that convert between multiple-argument
// and multiple-return functions and single-argument, single-return functions.
// This makes it trivial to pass arbitrary functions to generic operations
// that are designed to operate on arbitrary functions.
//
// For functions with as many argument or return parameters as can be represented by
// the tuple package, this package provides a function to convert to and from those
// forms.
//
// The names of most functions in this package match the following regular expression:
//
// 	ToC?A?R?E?_[0-9]+_[0-9]+
//
// Each optional letter represents one aspect of the function that's being converted to.
//
// 	C - context.Context argument
// 	A - argument parameter
// 	R - return parameter
// 	E - error return
//
// The first number is the number of argument parameters (not including context.Context for a C function);
// the second number is the number of return parameters (not including error for an E function).
//
// So, for example:
//
// 	ToCRE_1_3
//
// converts from (for some types A, R0, R1 and R2)
//
// 	func(context.Context, A) (R0, R1, R2, error)
//
// to:
//
// 	func(context.Context, A) (tuple.T3[R0, R1, R2], error)
//
// Note that the same function could also be converted with:
//
// 	ToAR_2_4
//
// with resulting signature:
//
// 	func(tuple.T2[context.Context, A]) tuple.T4[R0, R1, R2, error]
//
package tuplefunc
