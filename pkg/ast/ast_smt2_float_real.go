// Source: src/ast/ast_smt2_pp.cpp:235-268.
package ast

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	utext "github.com/glycerine/g3/pkg/util/text"
)

// Concrete source stack values live in this charged cold invocation header.
// Retained receipts are diagnostic history, distinct from their actual child.
type astFloatLiteralFrame struct {
	next                                               *astFloatLiteralFrame
	self                                               *astFloatLiteralFrame
	owner                                              *astFloatingPointPluginOwner
	value                                              util.ArbitraryFloat
	sig                                                util.Int
	sigCopy                                            util.ArbitraryFloatValueCopyReceipt
	sigConstructed, sigConsumed                        bool
	bvValues                                           [2]util.OwnedRational
	bvPlaced, bvComplete, bvConsumed                   [2]bool
	bvInvocations                                      [2]util.RationalServiceInvocation
	bvRefs                                             [2]astNodeRef
	bvRefAcquired, bvRefConsumed                       [2]bool
	bvChildren                                         [4]astFloatBVChild
	copy                                               util.ArbitraryFloatValueCopyReceipt
	buffer                                             utext.Buffer
	text                                               *util.ArbitraryFloatFormatText
	header, invocation                                 uint64
	valueConstructed, valueConsumed, bufferConstructed bool
	sourceDone, ownDone                                bool
	retired                                            bool
	bodyCause, sourceCause, ownCause                   error
}

func (p *astFloatingPointPluginOwner) literalBytes() uint64 {
	var n uint64
	for f := p.literalFrames; f != nil; f = f.next {
		n += f.header
		if t := f.text; t != nil {
			n += t.StorageBytes()
		}
	}
	return n
}

func (f *astFloatLiteralFrame) destroy(own bool) error {
	if f == nil || f.self != f || f.owner == nil || f.owner.self != f.owner {
		return ErrASTHandle
	}
	if own && f.ownDone || !own && f.sourceDone {
		return nil
	}
	if !own && f.sourceCause != nil {
		return f.sourceCause
	}
	stop := func(e error) error {
		if e != nil {
			if own {
				if f.ownCause == nil {
					f.ownCause = e
				}
			} else if f.sourceCause == nil {
				f.sourceCause = e
			}
		}
		return e
	}
	fm := f.owner.numeric
	if e := f.destroyBVParts(own); e != nil {
		return stop(e)
	}
	if own {
		if f.text != nil {
			if e := f.text.NumericInvocation().Cleanup(false); e != nil {
				return stop(e)
			}
		}
		if e := fm.CleanupValueCopy(&f.copy, false); e != nil {
			return stop(e)
		}
	} else if f.text != nil && !f.text.SourceEnded() {
		if e := f.text.SourceFinalize(); e != nil {
			return stop(e)
		}
	}
	if f.bufferConstructed && !f.buffer.Closed() {
		if e := f.buffer.Close(); e != nil {
			return stop(e)
		}
	}
	if f.valueConstructed && !f.valueConsumed {
		var e error
		if own {
			e = fm.CleanupValue(&f.value)
		} else {
			e = fm.Del(&f.value)
		}
		if e != nil {
			return stop(e)
		}
		f.valueConsumed = true
	}
	if own {
		if e := f.abortBVParts(); e != nil {
			return stop(e)
		}
		if e := fm.CleanupValueCopy(&f.copy, true); e != nil {
			return stop(e)
		}
		if f.text != nil {
			if e := f.text.Close(); e != nil {
				return stop(e)
			}
		}
		f.ownDone = true
	} else {
		f.sourceDone = true
	}
	return nil
}
func (p *astFloatingPointPluginOwner) destroyLiteralFrames(own bool) error {
	for f := p.literalFrames; f != nil; f = f.next {
		if e := f.destroy(own); e != nil {
			return e
		}
	}
	return nil
}
func (p *astFloatingPointPluginOwner) releaseLiteralFrames() {
	for f := p.literalFrames; f != nil; f = f.next {
		p.owner.BudgetField.Release(f.header)
		f.header = 0
	}
	p.literalFrames = nil
}

// The use_float_real_lits profile of the actual source method. The BV-literal
// suffix is a separate reached dependency, not substituted here.
func (m *ASTManager) ppFloatRealLiteral(h ASTHandle, g *util.GlobalParamsService) (ASTHandle, error) {
	return m.ppFloatLiteral(h, false, true, g)
}
func (m *ASTManager) ppFloatBitVectorLiteral(h ASTHandle, radix bool, g *util.GlobalParamsService) (ASTHandle, error) {
	return m.ppFloatLiteral(h, radix, false, g)
}
func (m *ASTManager) ppFloatLiteral(h ASTHandle, radix, real bool, g *util.GlobalParamsService) (out ASTHandle, result error) {
	if e := m.lock(); e != nil {
		return out, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	p := m.floatingPointPlugin
	if p == nil || p.self != p || p.sourceEnded || !p.published {
		return out, ErrASTHandle
	}
	r, e := m.requireExpr(h)
	if e != nil {
		return out, e
	}
	if g != nil && (g.CheckBudget(m.BudgetField) != nil) {
		return out, util.ErrGlobalParamsOwner
	}
	if p.literalSerial == math.MaxUint64 {
		return out, arena.ErrBounds
	}
	// The ordinary lexical owner is physically embedded and charged once in
	// the plugin constructor. A stopped invocation keeps that exact address;
	// overlapping retained calls need a separate real owner, never a moved NUM.
	f := &p.literalPrimary
	n := uint64(0)
	if f.self != nil && !f.retired {
		n = uint64(reflect.TypeOf(astFloatLiteralFrame{}).Size())
		if e = m.BudgetField.Reserve(n); e != nil {
			return out, e
		}
		f = &astFloatLiteralFrame{}
	}
	p.literalSerial++
	*f = astFloatLiteralFrame{owner: p, header: n, invocation: p.literalSerial, next: p.literalFrames, valueConstructed: true}
	f.self = f
	p.literalFrames = f
	// source scoped_mpf precedes get_manager and the default string_buffer.
	if e = f.buffer.Init(utext.Config{}); e != nil {
		return out, e
	}
	f.bufferConstructed = true
	defer func() {
		f.bodyCause = result
		result = errors.Join(result, f.destroy(false))
		// Successful source completion proves both lexical values consumed and
		// the actual callee completed. These exact ACKs cannot abort an unfinished
		// constructor: every failing body/source path retains the live owner.
		if result == nil && f.sourceDone {
			result = f.destroy(true)
			if result == nil {
				p.literalFrames = f.next
				m.BudgetField.Release(f.header)
				f.header = 0
				f.retired = true
			}
		}
	}()
	found, e := p.isNumeral(r, &f.value, &f.copy)
	if e != nil {
		return out, e
	}
	if !found {
		if g == nil {
			return out, util.ErrGlobalParamsAssertion
		}
		if e = g.NamespaceVerifyFailure("src/ast/ast_smt2_pp.cpp", 241, "get_futil().is_numeral(t, v)"); e != nil {
			return out, e
		}
	}
	if !real {
		return m.floatBitVectorSuffix(f, radix, g)
	}
	name := ""
	if yes, e := p.numeric.IsNaN(&f.value); e != nil {
		return out, e
	} else if yes {
		name = "NaN"
	} else if yes, e := p.numeric.IsPositiveInf(&f.value); e != nil {
		return out, e
	} else if yes {
		name = "+oo"
	} else if yes, e := p.numeric.IsNegativeInf(&f.value); e != nil {
		return out, e
	} else if yes {
		name = "-oo"
	} else if yes, e := p.numeric.IsPositiveZero(&f.value); e != nil {
		return out, e
	} else if yes {
		name = "+zero"
	} else if yes, e := p.numeric.IsNegativeZero(&f.value); e != nil {
		return out, e
	} else if yes {
		name = "-zero"
	}
	b := &f.buffer
	if name != "" {
		if e = b.AppendCString("(_ "); e != nil {
			return out, e
		}
		if e = b.AppendCString(name); e != nil {
			return out, e
		}
		if e = b.AppendCString(" "); e != nil {
			return out, e
		}
	} else if e = b.AppendCString("((_ to_fp "); e != nil {
		return out, e
	}
	if e = b.AppendUInt32(f.value.EBits()); e != nil {
		return out, e
	}
	if e = b.AppendCString(" "); e != nil {
		return out, e
	}
	if e = b.AppendUInt32(f.value.SBits()); e != nil {
		return out, e
	}
	if name == "" {
		if e = b.AppendCString(") RTZ "); e != nil {
			return out, e
		}
		f.text, e = util.NewArbitraryFloatFormatText(m.BudgetField)
		if e != nil {
			return out, e
		}
		view, e := p.numeric.ToStringInto(&f.value, f.text)
		if e != nil {
			return out, e
		}
		if e = view.AppendTo(b); e != nil {
			return out, e
		}
	}
	if e = b.AppendCString(")"); e != nil {
		return out, e
	}
	// Full-statement temporary DT precedes mk_string's argument/registration.
	if f.text != nil {
		if e = f.text.SourceFinalize(); e != nil {
			return out, e
		}
	}
	text, e := b.CStr()
	if e != nil {
		return out, e
	}
	fm := m.Format
	if fm == nil {
		fm = m
	} else {
		if e = fm.lock(); e != nil {
			return out, e
		}
		defer fm.mu.Unlock()
		fm.formatExecuting = true
		defer func() { fm.formatExecuting = false }()
	}
	symbol, e := fm.Symbols.InternBytes([]byte(text))
	if e != nil {
		return out, e
	}
	parameter := astParameterRecord{Kind: ASTParameterSymbol, symbol: symbol}
	node, e := fm.formatApp(ASTFormatString, &parameter, nil)
	return fm.external(node), e
}
