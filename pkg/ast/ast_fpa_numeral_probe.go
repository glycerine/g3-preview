// Source: src/ast/fpa_decl_plugin.cpp:136-139;
// src/ast/fpa_decl_plugin.h:257-267; src/util/scoped_numeral.h:24-33.
package ast

import (
	"errors"
	"reflect"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// The boolean is_numeral overload owns one caller-local scoped_mpf. A stopped
// source destructor leaves this exact value and its copy child for owning
// cleanup at plugin teardown.
type astFloatNumeralProbe struct {
	next, self  *astFloatNumeralProbe
	owner       *astFloatingPointPluginOwner
	value       unumeral.ArbitraryFloat
	copy        unumeral.ArbitraryFloatValueCopyReceipt
	header      uint64
	consumed    bool
	sourceDone  bool
	ownDone     bool
	retired     bool
	bodyCause   error
	sourceCause error
	ownCause    error
}

func (p *astFloatingPointPluginOwner) numeralProbeBytes() uint64 {
	var n uint64
	for f := p.numeralProbeFrames; f != nil; f = f.next {
		n += f.header
	}
	return n
}

func (f *astFloatNumeralProbe) destroy(own bool) error {
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
	m := f.owner.numeric
	if own {
		if e := m.CleanupValueCopy(&f.copy, false); e != nil {
			return stop(e)
		}
	}
	if !f.consumed {
		var e error
		if own {
			e = m.CleanupValue(&f.value)
		} else {
			e = m.Del(&f.value)
		}
		if e != nil {
			return stop(e)
		}
		f.consumed = true
	}
	if own {
		if e := m.CleanupValueCopy(&f.copy, true); e != nil {
			return stop(e)
		}
		f.ownDone = true
	} else {
		f.sourceDone = true
	}
	return nil
}

func (p *astFloatingPointPluginOwner) destroyNumeralProbes(own bool) error {
	for f := p.numeralProbeFrames; f != nil; f = f.next {
		if e := f.destroy(own); e != nil {
			return e
		}
	}
	return nil
}

func (p *astFloatingPointPluginOwner) releaseNumeralProbes() {
	for f := p.numeralProbeFrames; f != nil; f = f.next {
		p.owner.BudgetField.Release(f.header)
		f.header = 0
	}
	p.numeralProbeFrames = nil
}

type astFloatClass uint8

const (
	astFloatAny astFloatClass = iota
	astFloatNaN
	astFloatInf
	astFloatPInf
	astFloatNInf
	astFloatZero
	astFloatPZero
	astFloatNZero
	astFloatNormal
	astFloatSubnormal
	astFloatPositive
	astFloatNegative
)

// isNumeralClass preserves each source query's local value copy and destructor.
// The predicate runs only after is_numeral succeeds and before that destructor.
func (p *astFloatingPointPluginOwner) isNumeralClass(n astNodeRef, class astFloatClass) (matched bool, result error) {
	if p == nil || p.self != p || p.owner == nil || p.owner.floatingPointPlugin != p || !p.published || p.sourceEnded {
		return false, ErrASTHandle
	}
	f := &p.numeralProbePrimary
	if f.self != nil && !f.retired {
		bytes := uint64(reflect.TypeOf(astFloatNumeralProbe{}).Size())
		if e := p.owner.BudgetField.Reserve(bytes); e != nil {
			return false, e
		}
		f = &astFloatNumeralProbe{header: bytes}
	}
	header := f.header
	*f = astFloatNumeralProbe{owner: p, header: header, next: p.numeralProbeFrames}
	f.self = f
	p.numeralProbeFrames = f
	defer func() {
		f.bodyCause = result
		result = errors.Join(result, f.destroy(false))
		if result == nil && f.sourceDone {
			result = f.destroy(true)
			if result == nil {
				p.numeralProbeFrames = f.next
				p.owner.BudgetField.Release(f.header)
				f.header = 0
				f.retired = true
			}
		}
	}()
	found, e := p.isNumeral(n, &f.value, &f.copy)
	if e != nil || !found {
		return false, e
	}
	switch class {
	case astFloatAny:
		return true, nil
	case astFloatNaN:
		return p.numeric.IsNaN(&f.value)
	case astFloatInf:
		return p.numeric.IsInf(&f.value)
	case astFloatPInf:
		return p.numeric.IsPositiveInf(&f.value)
	case astFloatNInf:
		return p.numeric.IsNegativeInf(&f.value)
	case astFloatZero:
		return p.numeric.IsZero(&f.value)
	case astFloatPZero:
		return p.numeric.IsPositiveZero(&f.value)
	case astFloatNZero:
		return p.numeric.IsNegativeZero(&f.value)
	case astFloatNormal:
		return p.numeric.IsNormal(&f.value)
	case astFloatSubnormal:
		return p.numeric.IsDenormal(&f.value)
	case astFloatPositive:
		return p.numeric.IsPositive(&f.value)
	case astFloatNegative:
		return p.numeric.IsNegative(&f.value)
	default:
		return false, ErrASTDomain
	}
}

// isNumeralOnly is fpa_decl_plugin::is_numeral(expr*).
func (p *astFloatingPointPluginOwner) isNumeralOnly(n astNodeRef) (bool, error) {
	return p.isNumeralClass(n, astFloatAny)
}
