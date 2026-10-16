// Source: src/util/symbol_table.h:26-164; src/parsers/smt2/smt2parser.cpp:45-53.
package smt2

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

var errSMT2LocalScope = errors.New("g3: SMT2 local binding scope is empty")

// smt2Local is the parser's concrete symbol_table<local> value. Term is
// borrowed from the parser's expression stack, just as the source pointer is.
type smt2Local struct {
	term  ast.ASTHandle
	level uint32
}

type smt2LocalUndo struct {
	name     symbol.Symbol
	previous smt2Local
	existed  bool
}

// smt2LocalEnv preserves the source's overwrite and per-insert trail rules.
// Map storage replaces the C++ symbol hashtable; scope rollback consumes the
// trail from its end so repeated writes to one symbol restore in source order.
type smt2LocalEnv struct {
	bindings map[symbol.Symbol]smt2Local
	trail    []smt2LocalUndo
	limits   []int
}

// reset is reached when the parser releases its current command state. It
// discards both bindings and scope history, so a later parse starts fresh.
func (e *smt2LocalEnv) reset() {
	e.bindings = nil
	e.trail = nil
	e.limits = nil
}

func (e *smt2LocalEnv) find(name symbol.Symbol) (smt2Local, bool) {
	v, ok := e.bindings[name]
	return v, ok
}

func (e *smt2LocalEnv) contains(name symbol.Symbol) bool {
	_, ok := e.bindings[name]
	return ok
}

func (e *smt2LocalEnv) insert(name symbol.Symbol, value smt2Local) {
	if e.bindings == nil {
		e.bindings = make(map[symbol.Symbol]smt2Local)
	}
	if len(e.limits) != 0 {
		old, found := e.bindings[name]
		e.trail = append(e.trail, smt2LocalUndo{name: name, previous: old, existed: found})
	}
	e.bindings[name] = value
}

func (e *smt2LocalEnv) beginScope() {
	e.limits = append(e.limits, len(e.trail))
}

func (e *smt2LocalEnv) endScope() error {
	if len(e.limits) == 0 {
		return errSMT2LocalScope
	}
	last := len(e.limits) - 1
	limit := e.limits[last]
	e.limits = e.limits[:last]
	for len(e.trail) > limit {
		last = len(e.trail) - 1
		undo := e.trail[last]
		e.trail[last] = smt2LocalUndo{}
		e.trail = e.trail[:last]
		if undo.existed {
			e.bindings[undo.name] = undo.previous
		} else {
			delete(e.bindings, undo.name)
		}
	}
	return nil
}
