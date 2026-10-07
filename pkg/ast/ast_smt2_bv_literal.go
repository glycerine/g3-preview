// Source: src/ast/ast_smt2_pp.cpp:165-233; src/ast/format.h:89-104.
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type astBVLiteralFrame struct {
	self                                       *astBVLiteralFrame
	owner                                      *astBitVectorPluginOwner
	next                                       *astBVLiteralFrame
	header, invocation                         uint64
	values                                     [4]util.OwnedRational
	placed, complete, consumed                 [4]bool
	children                                   [5]util.RationalServiceInvocation
	normalization                              astBVNormalizationReceipt
	buffer                                     util.StringBuffer64
	characters                                 astBVCharBuffer16
	text                                       astBVIntegerText
	stringBuffer, characterBuffer, textReached bool
	sourceDone, ownDone, retired               bool
	ownPrefixDone                              bool
	bodyCause, sourceCause, ownCause           error
}

func (p *astBitVectorPluginOwner) literalBytes() uint64 {
	var n uint64
	for f := p.literalFrames; f != nil; f = f.next {
		n += f.header + uint64(len(f.buffer.State.Heap)) + f.text.storageBytes()
	}
	return n
}
func (f *astBVLiteralFrame) destroyValue(i int, own bool) error {
	if !f.placed[i] || f.consumed[i] || !own && !f.complete[i] {
		return nil
	}
	q := f.owner.owner.numerals
	var e error
	if own {
		e = q.CleanupValue(&f.values[i])
	} else {
		e = q.Del(&f.values[i])
	}
	if e == nil {
		f.consumed[i] = true
	}
	return e
}
func (f *astBVLiteralFrame) destroy(own bool) error {
	if !own {
		return f.destroyPhase(false, false)
	}
	if e := f.destroyPhase(true, false); e != nil {
		return e
	}
	return f.destroyPhase(true, true)
}
func (f *astBVLiteralFrame) destroyPhase(own, abort bool) (e error) {
	if f == nil || f.self != f || f.owner == nil || f.owner.self != f.owner {
		return ErrASTHandle
	}
	if own && f.ownDone || !own && f.sourceDone {
		return nil
	}
	defer func() {
		if e != nil {
			if own {
				if f.ownCause == nil {
					f.ownCause = e
				}
			} else if f.sourceCause == nil {
				f.sourceCause = e
			}
		}
	}()
	if !own && f.sourceCause != nil {
		return f.sourceCause
	}
	if own && abort {
		goto abortChildren
	}
	if own && f.ownPrefixDone {
		return nil
	}
	if own {
		if e = f.normalization.cleanup(f.owner, false); e != nil {
			return e
		}
	}
	// A reached younger normalization or numerical insertion source stop protects
	// every older caller value. Owning continuation acknowledges its own leaves.
	if !own {
		if e := f.normalization.sourceReady(f.owner); e != nil {
			return e
		}
	}
	for i := len(f.children) - 1; i >= 0; i-- {
		if own {
			e = f.children[i].Cleanup(false)
		} else {
			e = f.children[i].SourceReady()
		}
		if e != nil {
			return e
		}
	}
	if f.textReached {
		if own {
			e = f.text.cleanup(false)
		} else {
			e = f.text.invocation.SourceReady()
			if e == nil && f.text.returned {
				e = f.text.sourceDestroy()
			}
		}
		if e != nil {
			return e
		}
	}
	if f.characterBuffer {
		if own {
			e = f.characters.cleanup(f.owner.owner.numerals.RuntimeOwner())
		} else {
			e = f.characters.sourceDestroy(f.owner.owner.numerals.RuntimeOwner())
		}
		if e != nil {
			return e
		}
	}
	if f.stringBuffer && !f.buffer.State.Closed {
		if e = f.buffer.Close(); e != nil {
			return e
		}
	}
	for _, i := range [...]int{3, 2, 1, 0} {
		if e = f.destroyValue(i, own); e != nil {
			return e
		}
	}
	if own {
		f.ownPrefixDone = true
		return nil
	}
abortChildren:
	if own {
		if e = f.normalization.cleanup(f.owner, true); e != nil {
			return e
		}
	}
	if own {
		for i := len(f.children) - 1; i >= 0; i-- {
			if e = f.children[i].Cleanup(true); e != nil {
				return e
			}
		}
		if f.textReached {
			if e = f.text.cleanup(true); e != nil {
				return e
			}
		}
		f.ownDone = true
	} else {
		f.sourceDone = true
	}
	return nil
}
func (p *astBitVectorPluginOwner) destroyLiteralFrames(own bool) error {
	for f := p.literalFrames; f != nil; f = f.next {
		if e := f.destroy(own); e != nil {
			return e
		}
	}
	return nil
}
func (p *astBitVectorPluginOwner) releaseLiteralFrames() {
	for f := p.literalFrames; f != nil; f = f.next {
		for i := range f.children {
			if e := f.children[i].EndContext(); e != nil {
				panic(e)
			}
		}
		p.owner.BudgetField.Release(f.header)
		f.header = 0
	}
	p.literalFrames = nil
}
func (m *ASTManager) beginBVLiteralFrame() (*astBVLiteralFrame, error) {
	p := m.bitVectorPlugin
	if p == nil || p.self != p || !p.published {
		return nil, ErrASTHandle
	}
	if p.literalSerial == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	f := &p.literalPrimary
	var n uint64
	if f.self != nil && !f.retired {
		n = uint64(reflect.TypeOf(astBVLiteralFrame{}).Size())
		if e := m.BudgetField.Reserve(n); e != nil {
			return nil, e
		}
		f = &astBVLiteralFrame{}
	}
	p.literalSerial++
	*f = astBVLiteralFrame{owner: p, next: p.literalFrames, header: n, invocation: p.literalSerial}
	f.self = f
	p.literalFrames = f
	return f, nil
}
func (f *astBVLiteralFrame) place(i int) error {
	v, e := f.owner.owner.numerals.NewValue()
	if e != nil {
		return e
	}
	f.values[i] = v
	f.placed[i] = true
	f.complete[i] = true
	f.consumed[i] = false
	return nil
}
func (f *astBVLiteralFrame) endDigits() error {
	for i := 2; i <= 4; i++ {
		if e := f.children[i].Cleanup(false); e != nil {
			return e
		}
		if e := f.children[i].Cleanup(true); e != nil {
			return e
		}
		if e := f.children[i].EndContext(); e != nil {
			return e
		}
		f.children[i] = util.RationalServiceInvocation{}
	}
	return nil
}
func astBVPPFalse(g *util.GlobalParamsService, line int32, condition string) error {
	return util.NamespaceAssertionFailureOptional(g, "src/ast/ast_smt2_pp.cpp", line, condition)
}
func (m *ASTManager) ppBitVectorLiteral(h ASTHandle, signed, radix bool, g *util.GlobalParamsService) (out ASTHandle, result error) {
	if e := m.lock(); e != nil {
		return out, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	return m.ppBitVectorLiteralCore(h, signed, radix, g, nil)
}
func (m *ASTManager) ppBitVectorLiteralCore(h ASTHandle, signed, radix bool, g *util.GlobalParamsService, captured **astBVLiteralFrame) (out ASTHandle, result error) {
	if g != nil && (g.CheckBudget(m.BudgetField) != nil) {
		return out, util.ErrGlobalParamsOwner
	}
	fm := m.Format
	if fm == nil {
		return out, ErrASTDomain
	}
	r, e := m.requireExpr(h)
	if e != nil {
		return out, e
	}
	f, e := m.beginBVLiteralFrame()
	if e != nil {
		return out, e
	}
	if captured != nil {
		*captured = f
	}
	q := m.numerals
	defer func() {
		f.bodyCause = result
		result = errors.Join(result, f.destroy(false))
		if result == nil {
			result = f.destroy(true)
			if result == nil {
				for i := range f.children {
					if e := f.children[i].EndContext(); e != nil {
						result = e
						return
					}
				}
				f.owner.literalFrames = f.next
				m.BudgetField.Release(f.header)
				f.header = 0
				f.retired = true
			}
		}
	}()
	enabled, e := astBVAssertionEnabled(g)
	if e != nil {
		return out, e
	}
	if enabled {
		found, e := m.isBitVectorNumeral(r)
		if e != nil {
			return out, e
		}
		if !found {
			if e = astBVPPFalse(g, 167, "u.is_numeral(t)"); e != nil {
				return out, e
			}
		}
	}
	if e = f.place(0); e != nil {
		return out, e
	}
	width := uint32(1)
	_, e = m.readBitVectorNumeralWithInvocation(r, &f.values[0], &width, &f.children[0])
	if e != nil {
		return out, e
	}
	enabled, e = astBVAssertionEnabled(g)
	if e != nil {
		return out, e
	}
	if enabled {
		ok, e := q.IsInt(&f.values[0])
		if e != nil {
			return out, e
		}
		if !ok {
			if e = astBVPPFalse(g, 171, "val.is_int()"); e != nil {
				return out, e
			}
		}
	}
	normalized, e := m.normalizeBitVectorWithFrame(&f.values[0], width, signed, g, &f.normalization)
	if e != nil {
		return out, e
	}
	f.values[1] = normalized
	f.placed[1], f.complete[1] = true, true
	if e = q.CopyWithInvocation(&f.values[0], &f.values[1], &f.children[1]); e != nil {
		return out, e
	}
	if e = f.destroyValue(1, false); e != nil {
		return out, e
	}
	negative, e := q.IsNeg(&f.values[0])
	if e != nil {
		return out, e
	}
	if negative {
		if e = q.NegAssign(&f.values[0]); e != nil {
			return out, e
		}
	}
	enabled, e = astBVAssertionEnabled(g)
	if e != nil {
		return out, e
	}
	if enabled {
		bad, e := q.IsNeg(&f.values[0])
		if e != nil {
			return out, e
		}
		if bad {
			if e = astBVPPFalse(g, 178, "val.is_nonneg()"); e != nil {
				return out, e
			}
		}
	}
	if !radix {
		f.stringBuffer = true
		f.buffer.State = util.StringBufferState{Budget: m.BudgetField, Capacity: 64, Initial: 64, LongBits: 64}
		f.buffer.Self = &f.buffer
		if e = f.buffer.AppendCString("(_ bv"); e != nil {
			return out, e
		}
		f.textReached = true
		if e = f.text.construct(q, m.BudgetField, &f.values[0]); e != nil {
			return out, e
		}
		view, e := f.text.cstring()
		if e != nil {
			return out, e
		}
		if e = f.buffer.AppendFloatCString(view); e != nil {
			return out, e
		}
		if e = f.buffer.AppendCString(" "); e != nil {
			return out, e
		}
		if e = f.buffer.AppendUInt32(width); e != nil {
			return out, e
		}
		if e = f.buffer.AppendCString(")"); e != nil {
			return out, e
		}
		if e = f.text.sourceDestroy(); e != nil {
			return out, e
		}
		out, e = fm.FormatBuffer64(&f.buffer)
		if e != nil {
			return out, e
		}
	} else {
		f.characterBuffer = true
		runtime := q.RuntimeOwner()
		if e = f.characters.push(runtime, '#'); e != nil {
			return out, e
		}
		base, step, prefix := uint32(2), uint32(1), byte('b')
		if width%4 == 0 {
			base, step, prefix = 16, 4, 'x'
		}
		if e = f.characters.push(runtime, prefix); e != nil {
			return out, e
		}
		sz := uint32(0)
		for {
			positive, e := q.IsPos(&f.values[0])
			if e != nil {
				return out, e
			}
			if !positive {
				break
			}
			if e = f.place(1); e != nil {
				return out, e
			}
			if e = q.SetUint64(&f.values[1], uint64(base)); e != nil {
				return out, e
			}
			f.values[2], e = q.RemWithInvocation(&f.values[0], &f.values[1], &f.children[2])
			if e != nil {
				return out, e
			}
			f.placed[2], f.complete[2], f.consumed[2] = true, true, false
			if e = f.destroyValue(1, false); e != nil {
				return out, e
			}
			if e = f.place(1); e != nil {
				return out, e
			}
			if e = q.SetUint64(&f.values[1], uint64(base)); e != nil {
				return out, e
			}
			f.values[3], e = q.IDivWithInvocation(&f.values[0], &f.values[1], &f.children[3])
			if e != nil {
				return out, e
			}
			f.placed[3], f.complete[3], f.consumed[3] = true, true, false
			if e = q.CopyWithInvocation(&f.values[0], &f.values[3], &f.children[4]); e != nil {
				return out, e
			}
			if e = f.destroyValue(3, false); e != nil {
				return out, e
			}
			if e = f.destroyValue(1, false); e != nil {
				return out, e
			}
			enabled, e = astBVAssertionEnabled(g)
			if e != nil {
				return out, e
			}
			if enabled {
				nonnegative, e := q.GreaterEqualInt(&f.values[2], 0)
				if e != nil {
					return out, e
				}
				bad := !nonnegative
				if !bad {
					bad, e = q.LessInt(&f.values[2], int32(base))
					if e != nil {
						return out, e
					}
					bad = !bad
				}
				if bad {
					condition := "rational(0) <= c && c < rational(2)"
					line := int32(211)
					if base == 16 {
						condition = "rational(0) <= c && c < rational(16)"
						line = 194
					}
					if e = astBVPPFalse(g, line, condition); e != nil {
						return out, e
					}
				}
			}
			var glyph byte
			if base == 16 {
				below, e := q.LessEqualInt(&f.values[2], 9)
				if e != nil {
					return out, e
				}
				digit, e := q.Uint64(&f.values[2])
				if e != nil {
					return out, e
				}
				if below {
					glyph = byte('0' + digit)
				} else {
					glyph = byte('a' + digit - 10)
				}
			} else {
				zero, e := q.IsZero(&f.values[2])
				if e != nil {
					return out, e
				}
				if zero {
					glyph = '0'
				} else {
					glyph = '1'
				}
			}
			if e = f.characters.push(runtime, glyph); e != nil {
				return out, e
			}
			sz += step
			if e = f.destroyValue(2, false); e != nil {
				return out, e
			}
			if e = f.endDigits(); e != nil {
				return out, e
			}
		}
		for sz < width {
			if e = f.characters.push(runtime, '0'); e != nil {
				return out, e
			}
			sz += step
		}
		enabled, e = astBVAssertionEnabled(g)
		if e != nil {
			return out, e
		}
		if enabled && sz != width {
			if e = astBVPPFalse(g, 223, "sz == bv_size"); e != nil {
				return out, e
			}
		}
		for left, right := uint32(2), f.characters.pos; left < right; {
			right--
			if left >= right {
				break
			}
			a, e := f.characters.read(runtime, left)
			if e != nil {
				return out, e
			}
			z, e := f.characters.read(runtime, right)
			if e != nil {
				return out, e
			}
			if e = f.characters.write(runtime, left, z); e != nil {
				return out, e
			}
			if e = f.characters.write(runtime, right, a); e != nil {
				return out, e
			}
			left++
		}
		if e = f.characters.push(runtime, 0); e != nil {
			return out, e
		}
		out, e = fm.formatBVCharacters(&f.characters)
		if e != nil {
			return out, e
		}
	}
	if negative {
		return fm.formatBVNegOne(out)
	}
	return out, nil
}
func (m *ASTManager) formatBVNegOne(value ASTHandle) (ASTHandle, error) {
	space, e := m.FormatString(" ")
	if e != nil {
		return ASTHandle{}, e
	}
	empty, e := m.FormatCompose(nil)
	if e != nil {
		return ASTHandle{}, e
	}
	close, e := m.FormatString(")")
	if e != nil {
		return ASTHandle{}, e
	}
	inside, e := m.FormatCompose([]ASTHandle{space, value, empty, close})
	if e != nil {
		return ASTHandle{}, e
	}
	indent, e := m.FormatIndent(7, inside)
	if e != nil {
		return ASTHandle{}, e
	}
	open, e := m.FormatString("(")
	if e != nil {
		return ASTHandle{}, e
	}
	name, e := m.FormatString("bvneg")
	if e != nil {
		return ASTHandle{}, e
	}
	body, e := m.FormatCompose([]ASTHandle{open, name, indent})
	if e != nil {
		return ASTHandle{}, e
	}
	return m.FormatGroup(body)
}
