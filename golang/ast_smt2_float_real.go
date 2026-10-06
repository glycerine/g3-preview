// Source: src/ast/ast_smt2_pp.cpp:235-268.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
)

// Concrete source stack values live in this charged cold invocation header.
// Retained receipts are diagnostic history, distinct from their actual child.
type astFloatLiteralFrame struct {
	next                                               *astFloatLiteralFrame
	self                                               *astFloatLiteralFrame
	owner                                              *astFloatingPointPluginOwner
	value                                              mpnum.ArbitraryFloat
	sig                                                mpnum.Int
	sigCopy                                            mpnum.ArbitraryFloatValueCopyReceipt
	sigConstructed, sigConsumed                        bool
	bvValues                                           [2]mpnum.OwnedRational
	bvPlaced, bvComplete, bvConsumed                   [2]bool
	bvInvocations                                      [2]mpnum.RationalServiceInvocation
	bvRefs                                             [2]astNodeRef
	bvRefAcquired, bvRefConsumed                       [2]bool
	bvChildren                                         [4]astFloatBVChild
	copy                                               mpnum.ArbitraryFloatValueCopyReceipt
	buffer                                             StringBuffer64
	text                                               *ArbitraryFloatFormatText
	header, invocation                                 uint64
	valueConstructed, valueConsumed, bufferConstructed bool
	sourceDone, ownDone                                bool
	retired                                            bool
	bodyCause, sourceCause, ownCause                   error
}

func (p *astFloatingPointPluginOwner) literalBytes() uint64 {
	var n uint64
	for f := p.literalFrames; f != nil; f = f.next {
		n += f.header + uint64(len(f.buffer.state.heap))
		if t := f.text; t != nil {
			n += t.header + uint64(len(t.result.external)+len(t.result.pending)+len(t.temporary.external)+len(t.temporary.pending))
			if s := t.stream.NumericTextStream; s != nil {
				stats := s.Stats()
				n += stats.OwnerBytes + stats.ScratchBytes + stats.OutputOwnerBytes + stats.OutputOverflowBytes
			}
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
			if e := f.text.numeric.Cleanup(false); e != nil {
				return stop(e)
			}
		}
		if e := fm.CleanupValueCopy(&f.copy, false); e != nil {
			return stop(e)
		}
	} else if f.text != nil && !f.text.sourceEnded {
		if e := f.text.SourceFinalize(); e != nil {
			return stop(e)
		}
	}
	if f.bufferConstructed && !f.buffer.state.closed {
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
		p.owner.budget.Release(f.header)
		f.header = 0
	}
	p.literalFrames = nil
}

// The use_float_real_lits profile of the actual source method. The BV-literal
// suffix is a separate reached dependency, not substituted here.
func (m *ASTManager) ppFloatRealLiteral(h ASTHandle, g *GlobalParamsService) (ASTHandle, error) {
	return m.ppFloatLiteral(h, false, true, g)
}
func (m *ASTManager) ppFloatBitVectorLiteral(h ASTHandle, radix bool, g *GlobalParamsService) (ASTHandle, error) {
	return m.ppFloatLiteral(h, radix, false, g)
}
func (m *ASTManager) ppFloatLiteral(h ASTHandle, radix, real bool, g *GlobalParamsService) (out ASTHandle, result error) {
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
	if g != nil && (g.self != g || g.config.Budget != m.budget) {
		return out, ErrGlobalParamsOwner
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
		if e = m.budget.Reserve(n); e != nil {
			return out, e
		}
		f = &astFloatLiteralFrame{}
	}
	p.literalSerial++
	*f = astFloatLiteralFrame{owner: p, header: n, invocation: p.literalSerial, next: p.literalFrames, valueConstructed: true}
	f.self = f
	p.literalFrames = f
	// source scoped_mpf precedes get_manager and the default string_buffer.
	f.buffer = StringBuffer64{state: stringBufferState{budget: m.budget, capacity: 64, initial: 64, longBits: 64}}
	f.buffer.self = &f.buffer
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
				m.budget.Release(f.header)
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
		if g == nil || g.namespaceAssertions == nil {
			return out, ErrSATAssertion
		}
		if e = g.namespaceAssertions.VerifyFailure("src/ast/ast_smt2_pp.cpp", 241, "get_futil().is_numeral(t, v)"); e != nil {
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
		f.text, e = NewArbitraryFloatFormatText(m.budget)
		if e != nil {
			return out, e
		}
		view, e := p.numeric.ToStringInto(&f.value, f.text)
		if e != nil {
			return out, e
		}
		if e = b.AppendFloatCString(view); e != nil {
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
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	bytes, e := b.state.cstr(b.inline[:])
	if e != nil {
		return out, e
	}
	fm := m.format
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
	symbol, e := fm.symbols.InternBytes(bytes)
	if e != nil {
		return out, e
	}
	parameter := astParameterRecord{kind: ASTParameterSymbol, symbol: symbol}
	node, e := fm.formatApp(ASTFormatString, &parameter, nil)
	return fm.external(node), e
}
