// Source: src/cmd_context/context_params.cpp:198-209;
// src/ast/ast.cpp:1297-1334; src/smt/params/qi_params.cpp:43-64.
package smtcore

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

// Concrete call-local filename reader: borrowed c_str address identity remains
// distinct from owned string/Params storage. No copied path is retained here.

type astTraceFilename struct {
	params    util.ParamsStringView
	owned     QITextView
	ownedMode bool
}

func (n astTraceFilename) Byte(i uint64) (byte, error) {
	if !n.ownedMode {
		return n.params.Byte(i)
	}
	v := n.owned
	if v.Owner == nil || v.Owner.Self != v.Owner || v.Owner.Closed || !v.Owner.Ready || v.Generation != v.Owner.Generation {
		return 0, arena.ErrInvalidSlot
	}
	if i > v.Length {
		return 0, arena.ErrBounds
	}
	return v.Owner.CStringByte(i)
}
func (n astTraceFilename) Length() (uint64, error) {
	if !n.ownedMode {
		return n.params.Length()
	}
	for i := uint64(0); ; i++ {
		b, e := n.Byte(i)
		if e != nil {
			return 0, e
		}
		if b == 0 {
			return i, nil
		}
	}
}

// MkASTManager returns the actual placement-new failed owner on constructor
// failure. The caller may abort that exact owner; no successful source setters
// run before the constructor truly completes.
func (s *ContextParams) MkASTManager(c ast.ASTManagerConfig) (*ast.ASTManager, error) {
	if e := s.sourceCallReady(); e != nil {
		return nil, e
	}
	if c.Symbols != s.context.Symbols() || c.Numerals != s.context.Numerals() {
		return nil, util.ErrParamsOwner
	}
	c.ProofMode = ast.ASTProofDisabled
	if s.state.Proof {
		c.ProofMode = ast.ASTProofEnabled
	}
	var filename ast.TraceFilename
	if s.state.Trace {
		n := astTraceFilename{owned: s.traceFile.View(), ownedMode: true}
		filename = n
	}
	s.executing = true
	defer func() { s.executing = false }()
	m, e := ast.NewSourceASTManagerBorrowed(c, filename)
	if e != nil {
		return m, e
	}
	if s.state.SMTLib2Compliant {
		if e = m.EnableIntRealCoercions(false); e != nil {
			return m, e
		}
	}
	if s.state.DebugRefCount {
		if e = m.EnableDebugRefCount(); e != nil {
			return m, e
		}
	}
	return m, nil
}
