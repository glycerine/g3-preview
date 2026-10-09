// Source: src/util/lbool.h; src/util/lbool.cpp.
package util

import "github.com/glycerine/g3/pkg/util/lbool"

// Compatibility aliases while consumers move to the leaf package.
type LBool = lbool.LBool

const (
	LFalse = lbool.False
	LUndef = lbool.Undef
	LTrue  = lbool.True
)

func ToLBool(b bool) LBool { return lbool.FromBool(b) }
