// Source: src/ast/bv_decl_plugin.cpp:416-435,778-789,865-868.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
)

// The actual lexical parameter RATIONALs and full-expression temporaries.
// This primary owner is embedded once; a stopped owner keeps its exact address.
type astBVNumeralFrame struct {
	self                             *astBVNumeralFrame
	owner                            *astBitVectorPluginOwner
	next                             *astBVNumeralFrame
	children                         [10]mpnum.RationalServiceInvocation
	values                           [8]mpnum.OwnedRational // utility p[0], power, mod, p0, ps[0]
	placed, complete, consumed       [8]bool
	header, invocation               uint64
	sourceDone, ownDone, retired     bool
	ownPrefixDone                    bool
	normalization                    bool
	bodyCause, sourceCause, ownCause error
}

func (f *astBVNumeralFrame) childOrder() [10]int {
	if f.normalization {
		return [10]int{9, 8, 7, 6, 5, 4, 3, 1, 0, -1}
	}
	return [10]int{6, 5, 1, 0, 2, -1, -1, -1, -1, -1}
}

func (f *astBVNumeralFrame) destroyOne(i int, own bool) error {
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
	} else if own {
		if f.ownCause == nil {
			f.ownCause = e
		}
	} else if f.sourceCause == nil {
		f.sourceCause = e
	}
	return e
}
func (f *astBVNumeralFrame) destroy(own bool) error {
	if !own {
		return f.destroyPhase(false, false)
	}
	if e := f.destroyPhase(true, false); e != nil {
		return e
	}
	return f.destroyPhase(true, true)
}
func (f *astBVNumeralFrame) destroyPhase(own, abort bool) error {
	if f == nil || f.self != f || f.owner == nil || f.owner.self != f.owner {
		return ErrASTHandle
	}
	if own && f.ownDone || !own && f.sourceDone {
		return nil
	}
	if !own && f.sourceCause != nil {
		return f.sourceCause
	}
	if own && abort {
		goto abortChildren
	}
	if own && f.ownPrefixDone {
		return nil
	}
	for _, i := range f.childOrder() {
		if i < 0 {
			continue
		}
		var e error
		if own {
			e = f.children[i].Cleanup(false)
		} else {
			e = f.children[i].SourceReady()
		}
		if e != nil {
			if own {
				if f.ownCause == nil {
					f.ownCause = e
				}
			} else {
				if f.sourceCause == nil {
					f.sourceCause = e
				}
			}
			return e
		}
	}
	for _, i := range [...]int{7, 6, 5, 4, 3, 2, 1, 0} {
		if e := f.destroyOne(i, own); e != nil {
			return e
		}
	}
	if own {
		f.ownPrefixDone = true
		return nil
	}
abortChildren:
	if own {
		for _, i := range f.childOrder() {
			if i < 0 {
				continue
			}
			if e := f.children[i].Cleanup(true); e != nil {
				if f.ownCause == nil {
					f.ownCause = e
				}
				return e
			}
		}
		f.ownDone = true
	} else {
		f.sourceDone = true
	}
	return nil
}
func (p *astBitVectorPluginOwner) numeralBytes() uint64 {
	var n uint64
	for f := p.numeralFrames; f != nil; f = f.next {
		n += f.header
	}
	return n
}
func (p *astBitVectorPluginOwner) destroyNumeralFrames(own bool) error {
	for f := p.numeralFrames; f != nil; f = f.next {
		if e := f.destroy(own); e != nil {
			return e
		}
	}
	return nil
}
func (p *astBitVectorPluginOwner) releaseNumeralFrames() {
	for f := p.numeralFrames; f != nil; f = f.next {
		for _, i := range f.childOrder() {
			if i < 0 {
				continue
			}
			if e := f.children[i].EndContext(); e != nil {
				panic(e)
			}
		}
		p.owner.budget.Release(f.header)
		f.header = 0
	}
	p.numeralFrames = nil
}
func (m *ASTManager) beginBVNumeralFrame() (*astBVNumeralFrame, error) {
	p := m.bitVectorPlugin
	if p == nil || p.self != p || !p.published {
		return nil, ErrASTHandle
	}
	if p.numeralSerial == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	f := &p.numeralPrimary
	n := uint64(0)
	if f.self != nil && !f.retired {
		n = uint64(reflect.TypeOf(astBVNumeralFrame{}).Size())
		if e := m.budget.Reserve(n); e != nil {
			return nil, e
		}
		f = &astBVNumeralFrame{}
	}
	p.numeralSerial++
	*f = astBVNumeralFrame{owner: p, header: n, invocation: p.numeralSerial, next: p.numeralFrames}
	f.self = f
	p.numeralFrames = f
	return f, nil
}
func (f *astBVNumeralFrame) copyConstruct(i int, src *mpnum.OwnedRational) error {
	q := f.owner.owner.numerals
	v, e := q.NewValue()
	if e != nil {
		return e
	}
	f.values[i] = v
	f.placed[i] = true
	if e = q.CopyWithInvocation(&f.values[i], src, &f.children[2+i]); e != nil {
		return e
	}
	f.complete[i] = true
	return nil
}

func (m *ASTManager) mkBitVectorNumeral(value *mpnum.OwnedRational, bits uint32) (astNodeRef, error) {
	return m.mkBitVectorNumeralCaptured(value, bits, nil)
}
func (m *ASTManager) mkBitVectorNumeralCaptured(value *mpnum.OwnedRational, bits uint32, receipt *astBVNormalizationReceipt) (node astNodeRef, result error) {
	f, e := m.beginBVNumeralFrame()
	if e != nil {
		return node, e
	}
	if receipt != nil {
		*receipt = astBVNormalizationReceipt{frame: f, invocation: f.invocation}
	}
	defer func() {
		defer func() {
			if receipt != nil {
				receipt.retired = f.retired
			}
		}()
		f.bodyCause = result
		result = errors.Join(result, f.destroy(false))
		if receipt != nil {
			receipt.sourceDone = f.sourceDone
			receipt.bodyCause = f.bodyCause
			receipt.sourceCause = f.sourceCause
		}
		if result == nil && f.sourceDone {
			result = f.destroy(true)
			if result == nil {
				for _, i := range f.childOrder() {
					if i < 0 {
						continue
					}
					if e := f.children[i].EndContext(); e != nil {
						result = e
						return
					}
				}
				f.owner.numeralFrames = f.next
				m.budget.Release(f.header)
				f.header = 0
				f.retired = true
			}
		}
	}()
	if e = f.copyConstruct(0, value); e != nil {
		return node, e
	}
	// The inner plugin checks parameter kinds/count before unsigned width.
	if bits == 0 {
		return node, ASTException{NewDefaultException("bit-vector size must be greater than zero")}
	}
	q := m.numerals
	f.values[1], e = q.PowerOfTwoWithInvocation(bits, &f.children[0])
	if e != nil {
		return node, e
	}
	f.placed[1], f.complete[1] = true, true
	f.values[2], e = q.ModWithInvocation(&f.values[0], &f.values[1], &f.children[1])
	if e != nil {
		return node, e
	}
	f.placed[2], f.complete[2] = true, true
	if e = f.copyConstruct(3, &f.values[2]); e != nil {
		return node, e
	}
	// Full-expression temporary order is mod result, then power return value.
	if e = f.destroyOne(2, false); e != nil {
		return node, e
	}
	if e = f.destroyOne(1, false); e != nil {
		return node, e
	}
	if e = f.copyConstruct(4, &f.values[3]); e != nil {
		return node, e
	}
	sort, e := f.owner.mkSort(bits)
	if e != nil {
		return node, e
	}
	parameter := astParameterRecord{kind: ASTParameterRational, rational: f.values[4]}
	if e = m.beginFormatInfo(false, f.owner.family, 0, &parameter); e != nil {
		return node, e
	}
	defer m.endFormatInfo()
	m.formatLexical.secondParameter = astParameterRecord{kind: ASTParameterInt, integer: int32(bits)}
	m.formatLexical.localPair = true
	m.formatLexical.info.count = 2
	epoch := m.formatLexical.epoch * 2
	ps := [2]ASTParameter{{m.origin, math.MaxUint32 - 1, epoch}, {m.origin, math.MaxUint32 - 1, epoch + 1}}
	decl, e := m.pluginDecl(f.owner.names[0], f.owner.family, 0, ps[:], nil, sort, ASTFuncFlags{}, false)
	if e != nil {
		return node, e
	}
	return m.mkApp(decl, nil)
}
func (m *ASTManager) BitVectorNumeral(value *mpnum.OwnedRational, bits uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	r, e := m.mkBitVectorNumeral(value, bits)
	return m.external(r), e
}

func (m *ASTManager) readBitVectorNumeral(r astNodeRef, out *mpnum.OwnedRational, width *uint32) (bool, error) {
	return m.readBitVectorNumeralWithInvocation(r, out, width, nil)
}
func (m *ASTManager) readBitVectorNumeralWithInvocation(r astNodeRef, out *mpnum.OwnedRational, width *uint32, invocation *mpnum.RationalServiceInvocation) (bool, error) {
	if e := m.validNode(r); e != nil {
		return false, e
	}
	if m.nodeAt(r.slot).kind != ASTApp {
		return false, nil
	}
	d := m.appAt(r.slot).decl
	family, kind, e := m.declIdentity(d)
	if e != nil {
		return false, e
	}
	p := m.bitVectorPlugin
	if p == nil || family != p.family || kind != 0 {
		return false, nil
	}
	info, e := m.funcInfo(m.declAt(d.slot).info)
	if e != nil {
		return false, e
	}
	token, e := m.infoParameter(info, 0)
	if e != nil {
		return false, e
	}
	value, e := m.parameter(token)
	if e != nil {
		return false, e
	}
	if value.kind != ASTParameterRational {
		return false, ErrASTDomain
	}
	if invocation == nil {
		e = m.numerals.Copy(out, &value.rational)
	} else {
		e = m.numerals.CopyWithInvocation(out, &value.rational, invocation)
	}
	if e != nil {
		return false, e
	}
	token, e = m.infoParameter(info, 1)
	if e != nil {
		return false, e
	}
	value, e = m.parameter(token)
	if e != nil {
		return false, e
	}
	if value.kind != ASTParameterInt || width == nil {
		return false, ErrASTDomain
	}
	*width = uint32(value.integer)
	return true, nil
}
func (m *ASTManager) BitVectorNumeralValue(h ASTHandle, out *mpnum.OwnedRational, width *uint32) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	r, e := m.requireExpr(h)
	if e != nil {
		return false, e
	}
	return m.readBitVectorNumeral(r, out, width)
}

func (m *ASTManager) isBitVectorNumeral(r astNodeRef) (bool, error) {
	if e := m.validNode(r); e != nil {
		return false, e
	}
	if m.nodeAt(r.slot).kind != ASTApp {
		return false, nil
	}
	d := m.appAt(r.slot).decl
	family, kind, e := m.declIdentity(d)
	if e != nil {
		return false, e
	}
	p := m.bitVectorPlugin
	return p != nil && family == p.family && kind == 0, nil
}
