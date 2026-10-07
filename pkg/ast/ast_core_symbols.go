// Licensed under the MIT license.
// Source symbol interning byte-input boundary used by actual AST name buffers.
package ast

import (
	"strconv"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Source string_buffer<64> keeps its first64 bytes on the stack, doubles on
// overflow and appends a NUL via c_str(). The byte interner avoids a Go string
// intermediate even for long names. All overflow arrays have lexical ownership.
func (m *ASTManager) modelValueName(sortName util.Symbol, index int32) (util.Symbol, error) {
	var initial [64]byte
	buffer := initial[:0]
	var charged uint64
	defer func() { m.BudgetField.Release(charged) }()
	appendByte := func(v byte) error {
		if len(buffer) == cap(buffer) {
			n := uint64(cap(buffer)) * 2
			if n > uint64(^uint(0)>>1) {
				return arena.ErrBounds
			}
			if e := m.BudgetField.Reserve(n); e != nil {
				return e
			}
			next := make([]byte, len(buffer), int(n))
			copy(next, buffer)
			m.BudgetField.Release(charged)
			charged = n
			buffer = next
		}
		buffer = append(buffer, v)
		return nil
	}
	view, e := m.Symbols.CStringView(sortName)
	if e == nil && (sortName.IsNumerical() || sortName.IsNull()) {
		e = ErrASTDomain
	}
	if e == nil {
		var n uint64
		n, e = view.Length()
		if e == nil {
			for i := uint64(0); i < n; i++ {
				var v byte
				v, e = view.Byte(i)
				if e != nil {
					break
				}
				e = appendByte(v)
				if e != nil {
					break
				}
			}
		}
	}

	if e != nil {
		return util.Symbol{}, e
	}
	for i := 0; i < len("!val!"); i++ {
		if e = appendByte("!val!"[i]); e != nil {
			return util.Symbol{}, e
		}
	}
	var digits [24]byte
	number := strconv.AppendInt(digits[:0], int64(index), 10)
	for _, v := range number {
		if e = appendByte(v); e != nil {
			return util.Symbol{}, e
		}
	}
	if e = appendByte(0); e != nil {
		return util.Symbol{}, e
	}
	return m.Symbols.InternBytes(buffer)
}
