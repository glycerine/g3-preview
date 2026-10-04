// Licensed under the MIT license.
// Translation of literal/model parts of src/sat/sat_types.h, Leonardo de Moura.
package g3

import (
	"strconv"
	"strings"
)

type satBoolVar uint32

const satNullBoolVar satBoolVar = ^satBoolVar(0) >> 1

// A literal is fixed uint32 in both build modes, distinct from arena handles.
// Its Go zero value denotes positive variable zero. Source default construction
// must call satDefaultLiteral explicitly to produce the null sentinel.
type satLiteral uint32

func satDefaultLiteral() satLiteral { return satLiteral(satNullBoolVar << 1) }
func satMakeLiteral(v satBoolVar, sign bool) satLiteral {
	if v > satNullBoolVar {
		panic("sat: Boolean variable outside literal encoding")
	}
	bits := uint32(v) << 1
	if sign {
		bits++
	}
	return satLiteral(bits)
}
func satLiteralFromUint(v uint32) satLiteral { return satLiteral(v) }
func (l satLiteral) variable() satBoolVar    { return satBoolVar(l >> 1) }
func (l satLiteral) sign() bool              { return l&1 != 0 }
func (l satLiteral) unsign() satLiteral      { return l &^ 1 }
func (l satLiteral) index() uint32           { return uint32(l) }
func (l satLiteral) hash() uint32            { return uint32(l) }
func (l *satLiteral) neg()                   { *l ^= 1 }
func (l satLiteral) negated() satLiteral     { return l ^ 1 }
func (l satLiteral) String() string {
	prefix := ""
	if l.sign() {
		prefix = "-"
	}
	return prefix + strconv.FormatUint(uint64(l.variable()), 10)
}
func (l satLiteral) dimacsString() string {
	prefix := ""
	if l.sign() {
		prefix = "-"
	}
	return prefix + strconv.FormatUint(uint64(l.variable())+1, 10)
}

// satModel owns a pointer-free slice; consumers retain their own result storage.
// This is the source partial model representation, not a model producer.
type satModel []lbool

func (m satModel) valueAt(l satLiteral) lbool {
	r := m[l.variable()]
	if l.sign() {
		return r.negated()
	}
	return r
}
func (m satModel) String() string {
	var out strings.Builder
	first := true
	for v, value := range m {
		if value == lUndef {
			continue
		}
		if !first {
			out.WriteByte(' ')
		}
		first = false
		if value != lTrue {
			out.WriteByte('-')
		}
		out.WriteString(strconv.FormatUint(uint64(v), 10))
	}
	return out.String()
}
func satLiteralsString(ls []satLiteral) string {
	var out strings.Builder
	for i, l := range ls {
		if i > 0 {
			out.WriteByte(' ')
		}
		out.WriteString(l.String())
	}
	return out.String()
}
