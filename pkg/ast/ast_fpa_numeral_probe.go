// Source: src/ast/fpa_decl_plugin.cpp:136-139; src/util/scoped_numeral.h:24-33.
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

// isNumeralOnly preserves the source overload's value copy and destructor,
// even though its caller observes only the boolean result.
func (p *astFloatingPointPluginOwner) isNumeralOnly(n astNodeRef) (found bool, result error) {
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
	return p.isNumeral(n, &f.value, &f.copy)
}
