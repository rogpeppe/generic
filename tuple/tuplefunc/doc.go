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
// 	ToC?A?R?E?_[0-9]+(_[0-9]+)?
//
// Each optional letter represents one aspect of the function that's being converted to.
//
// 	C - context.Context argument
// 	A - argument parameter
// 	R - return parameter
// 	E - error return
//
// When there are both argument and return parameters (both A and R are present), the first number holds the
// number of argument parameters (not including context.Context for a C function)
// and the second holds the number of return parameters (not including error for an E function).
//
// For a function form that can never include argument parameters (no A is present), there's
// only a single number holding the number of return parameters.
//
// For a function form that can never include return parameters (no R is present), there's
// only a single number holding the number of argument parameters.
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
//
// Another example:
//
//	ToA_2
//
// converts from (for some types A0, A1)
//
//
//	func(A0, A1)
//
// to:
//
//	func(tuple.T2[A0, A1])
package tuplefunc
