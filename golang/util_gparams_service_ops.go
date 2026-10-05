// Source: src/util/gparams.cpp:557-631; src/util/gparams.h:36-143;
// src/util/symbol.h:96; src/util/debug.h:60-65.
// Author: Leonardo de Moura (gparams: 2012-11-29; symbol: 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
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
func (s *GlobalParamsService) namespaceAssertion(value bool) error {
	if !globalParamsSourceDebug {
		return nil
	}
	enabled := true
	if s.namespaceAssertions != nil {
		var e error
		enabled, e = s.namespaceAssertions.AssertionsEnabled()
		if e != nil {
			return e
		}
	}
	if enabled && !value {
		return ErrGlobalParamsAssertion
	}
	return nil
}
func (s *GlobalParamsService) namespaceImp() (*GlobalParams, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	if e := s.namespaceAssertion(s.current != nil); e != nil {
		return nil, e
	}
	// A null source pointer with SASSERT erased/disabled is an undefined native
	// dereference domain. The typed adapter stops at that reached pointer access.
	if s.current == nil {
		return nil, ErrGlobalParamsUninitialized
	}
	return s.current, nil
}
func (s *GlobalParamsService) SourceReset() error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.Reset()
}
func (s *GlobalParamsService) SourceSetName(name, value string) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.SetName(name, value)
}
func (s *GlobalParamsService) SourceSetSymbol(name Symbol, value string) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	if e = s.namespaceAssertion(!name.IsNumerical()); e != nil {
		return e
	}
	if name.IsNumerical() {
		return g.SetName("", value)
	} // actual bare_str ternary
	return g.SetSymbol(name, value)
}
func (s *GlobalParamsService) SourceGetValueName(name string) (string, error) {
	g, e := s.namespaceImp()
	if e != nil {
		return "", e
	}
	return g.GetValueName(name)
}
func (s *GlobalParamsService) SourceGetValueSymbol(name Symbol) (string, error) {
	g, e := s.namespaceImp()
	if e != nil {
		return "", e
	}
	if e = s.namespaceAssertion(!name.IsNumerical()); e != nil {
		return "", e
	}
	if name.IsNumerical() {
		return g.GetValueName("")
	}
	return g.GetValueSymbol(name)
}
func (s *GlobalParamsService) SourceRegisterGlobal(d *ParamDescrs) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.RegisterGlobal(d)
}
func (s *GlobalParamsService) SourceRegisterModule(name string, d *ParamDescrs) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.RegisterModule(name, d)
}
func (s *GlobalParamsService) SourceRegisterModuleDescription(name string, r *arena.BytesRegion, ref arena.BytesRegionRef, offset uint32) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.RegisterModuleDescription(name, r, ref, offset)
}
func (s *GlobalParamsService) SourceGetModuleName(name string) (ParamsRef, error) {
	if e := s.open(); e != nil {
		return ParamsRef{}, e
	}
	key, e := s.symbols.Intern(name)
	if e != nil {
		return ParamsRef{}, e
	}
	return s.SourceGetModuleSymbol(key)
}
func (s *GlobalParamsService) SourceGetModuleSymbol(name Symbol) (ParamsRef, error) {
	g, e := s.namespaceImp()
	if e != nil {
		return ParamsRef{}, e
	}
	return g.GetModule(name)
}
func (s *GlobalParamsService) SourceGetRef() (*ParamsRef, error) {
	g, e := s.namespaceImp()
	if e != nil {
		return nil, e
	}
	return g.GetRef()
}
func (s *GlobalParamsService) SourceDisplayInto(out *StringBuffer64, indent uint32, smt2, includeDescr bool) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.DisplayInto(out, indent, smt2, includeDescr)
}
func (s *GlobalParamsService) SourceDisplayModulesInto(out *StringBuffer64) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.DisplayModulesInto(out)
}
func (s *GlobalParamsService) SourceDisplayModuleInto(out *StringBuffer64, name string) error {
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	key, e := s.symbols.Intern(name)
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
	g, e := s.namespaceImp()
	if e != nil {
		return e
	}
	return g.DisplayParameterInto(out, debugOut, name)
}
