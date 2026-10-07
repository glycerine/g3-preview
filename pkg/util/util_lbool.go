// Licensed under the MIT license.
// Translation of src/util/lbool.h and lbool.cpp, Leonardo de Moura.
package util

// lbool preserves false/undefined/true as distinct source values. Source
// enum arithmetic is fixed int32; undefined is not converted to Boolean false.
type LBool int32

const (
	LFalse LBool = -1
	LUndef LBool = 0
	LTrue  LBool = 1
)

func (l LBool) Negated() LBool { return -l }
func ToLBool(b bool) LBool {
	if b {
		return LTrue
	}
	return LFalse
}
func (l LBool) String() string {
	switch l {
	case LFalse:
		return "l_false"
	case LTrue:
		return "l_true"
	default:
		return "l_undef"
	}
}
func (l LBool) SATString() string {
	switch l {
	case LFalse:
		return "unsatisfiable"
	case LTrue:
		return "satisfiable"
	default:
		return "unknown"
	}
}
