// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Translation of src/util/lbool.h and lbool.cpp, Leonardo de Moura.
package g3

// lbool preserves false/undefined/true as distinct source values. Source
// enum arithmetic is fixed int32; undefined is not converted to Boolean false.
type lbool int32

const (
	lFalse lbool = -1
	lUndef lbool = 0
	lTrue  lbool = 1
)

func (l lbool) negated() lbool { return -l }
func toLBool(b bool) lbool {
	if b {
		return lTrue
	}
	return lFalse
}
func (l lbool) String() string {
	switch l {
	case lFalse:
		return "l_false"
	case lTrue:
		return "l_true"
	default:
		return "l_undef"
	}
}
func (l lbool) satString() string {
	switch l {
	case lFalse:
		return "unsatisfiable"
	case lTrue:
		return "satisfiable"
	default:
		return "unknown"
	}
}
