// Source: src/util/gparams.cpp:557-631; src/util/gparams.h:36-143;
// src/util/symbol.h:96; src/util/debug.h:60-65.
// Author: Leonardo de Moura (gparams: 2012-11-29; symbol: 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrGlobalParamsAssertion = errors.New("g3: global parameters source assertion")

// The source global assertion context is represented by one concrete cold
// borrow. Binding only stores its pointer; its lifetime is checked at reached
// DEBUG SASSERT sites. Nil retains the source enabled default. Release erases
// assertion accesses. This does not install a process-wide global binding.
func (s *GlobalParamsService) BindNamespaceAssertions(d *DebugContext) error {
	if e := s.open(); e != nil {
		return e
	}
	s.namespaceAssertions = d
	return nil
}
func (s *GlobalParamsService) NamespaceAssertionsEnabled() (bool, error) {
	if !GlobalParamsSourceDebug {
		return false, nil
	}
	if s.namespaceAssertions == nil {
		return true, nil
	}
	return s.namespaceAssertions.AssertionsEnabled()
}
func (s *GlobalParamsService) NamespaceAssertionFailure(file string, line int32, condition string) error {
	d := s.namespaceAssertions
	if d == nil || d.assertionActions == nil {
		return ErrGlobalParamsAssertion
	}
	if e := d.NotifyAssertionViolation(file, line, condition); e != nil {
		return e
	}
	return d.assertionActions.InvokeDebugger()
}
func (s *GlobalParamsService) namespaceImpAt(line int32) (*GlobalParams, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	// The macro gate is reached before evaluating the actual current pointer.
	enabled, e := s.NamespaceAssertionsEnabled()
	if e != nil {
		return nil, e
	}
	if enabled && s.Current == nil {
		if e := s.NamespaceAssertionFailure("src/util/gparams.cpp", line, "g_imp != 0"); e != nil {
			return nil, e
		}
	}
	// Continued or erased null dereference is a reached native undefined domain.
	if s.Current == nil {
		return nil, ErrGlobalParamsUninitialized
	}
	return s.Current, nil
}
func (s *GlobalParamsService) SourceReset() error {
	g, e := s.namespaceImpAt(558)
	if e != nil {
		return e
	}
	return g.Reset()
}
func (s *GlobalParamsService) SourceSetName(name, value string) error {
	g, e := s.namespaceImpAt(564)
	if e != nil {
		return e
	}
	return g.SetName(name, value)
}
func (s *GlobalParamsService) SourceSetSymbol(name Symbol, value string) error {
	g, e := s.namespaceImpAt(569)
	if e != nil {
		return e
	}
	enabled, e := s.NamespaceAssertionsEnabled()
	if e != nil {
		return e
	}
	if enabled && name.IsNumerical() {
		if e = s.NamespaceAssertionFailure("src/util/symbol.h", 96, "!is_numerical()"); e != nil {
			return e
		}
	}
	if name.IsNumerical() {
		return g.SetName("", value)
	} // actual bare_str ternary
	return g.SetSymbol(name, value)
}
func (s *GlobalParamsService) SourceGetValueName(name string) (string, error) {
	g, e := s.namespaceImpAt(574)
	if e != nil {
		return "", e
	}
	return g.GetValueName(name)
}
func (s *GlobalParamsService) SourceGetValueSymbol(name Symbol) (string, error) {
	g, e := s.namespaceImpAt(579)
	if e != nil {
		return "", e
	}
	enabled, e := s.NamespaceAssertionsEnabled()
	if e != nil {
		return "", e
	}
	if enabled && name.IsNumerical() {
		if e = s.NamespaceAssertionFailure("src/util/symbol.h", 96, "!is_numerical()"); e != nil {
			return "", e
		}
	}
	if name.IsNumerical() {
		return g.GetValueName("")
	}
	return g.GetValueSymbol(name)
}
func (s *GlobalParamsService) SourceRegisterGlobal(d *ParamDescrs) error {
	g, e := s.namespaceImpAt(584)
	if e != nil {
		return e
	}
	return g.RegisterGlobal(d)
}
func (s *GlobalParamsService) SourceRegisterModule(name string, d *ParamDescrs) error {
	g, e := s.namespaceImpAt(589)
	if e != nil {
		return e
	}
	return g.RegisterModule(name, d)
}
func (s *GlobalParamsService) SourceRegisterModuleDescription(name string, r *arena.BytesRegion, ref arena.BytesRegionRef, offset uint32) error {
	g, e := s.namespaceImpAt(594)
	if e != nil {
		return e
	}
	return g.RegisterModuleDescription(name, r, ref, offset)
}
func (s *GlobalParamsService) SourceGetModuleName(name string) (ParamsRef, error) {
	if e := s.open(); e != nil {
		return ParamsRef{}, e
	}
	key, e := s.Symbols.Intern(name)
	if e != nil {
		return ParamsRef{}, e
	}
	return s.SourceGetModuleSymbol(key)
}
func (s *GlobalParamsService) SourceGetModuleSymbol(name Symbol) (ParamsRef, error) {
	g, e := s.namespaceImpAt(603)
	if e != nil {
		return ParamsRef{}, e
	}
	return g.GetModule(name)
}
func (s *GlobalParamsService) SourceGetRef() (*ParamsRef, error) {
	g, e := s.namespaceImpAt(609)
	if e != nil {
		return nil, e
	}
	return g.GetRef()
}
func (s *GlobalParamsService) SourceDisplayInto(out *StringBuffer64, indent uint32, smt2, includeDescr bool) error {
	g, e := s.namespaceImpAt(614)
	if e != nil {
		return e
	}
	return g.DisplayInto(out, indent, smt2, includeDescr)
}
func (s *GlobalParamsService) SourceDisplayModulesInto(out *StringBuffer64) error {
	g, e := s.namespaceImpAt(619)
	if e != nil {
		return e
	}
	return g.DisplayModulesInto(out)
}
func (s *GlobalParamsService) SourceDisplayModuleInto(out *StringBuffer64, name string) error {
	g, e := s.namespaceImpAt(624)
	if e != nil {
		return e
	}
	key, e := s.Symbols.Intern(name)
	if e != nil {
		return e
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if e = g.open(); e != nil {
		return e
	}
	return g.displayModuleKey(out, key)
}
func (s *GlobalParamsService) SourceDisplayParameterInto(out, debugOut *StringBuffer64, name string) error {
	g, e := s.namespaceImpAt(629)
	if e != nil {
		return e
	}
	return g.DisplayParameterInto(out, debugOut, name)
}
