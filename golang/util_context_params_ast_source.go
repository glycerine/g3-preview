// Licensed under the MIT license; see LICENSE.txt.
// Source: src/cmd_context/context_params.cpp:198-209;
// src/ast/ast.cpp:1297-1334; src/smt/params/qi_params.cpp:43-64.
// Source author: Leonardo de Moura.
package g3

import "github.com/glycerine/g3/golang/arena"

// Concrete call-local filename reader: borrowed c_str address identity remains
// distinct from owned string/Params storage. No copied path is retained here.
type astTraceFilename struct {
	params    ParamsStringView
	owned     QITextView
	ownedMode bool
}

func (n astTraceFilename) byte(i uint64) (byte, error) {
	if !n.ownedMode {
		return n.params.Byte(i)
	}
	v := n.owned
	if v.owner == nil || v.owner.self != v.owner || v.owner.closed || !v.owner.ready || v.generation != v.owner.generation {
		return 0, arena.ErrInvalidSlot
	}
	if i > v.length {
		return 0, arena.ErrBounds
	}
	if v.owner.external != nil {
		return v.owner.external[i], nil
	}
	return v.owner.inline[i], nil
}
func (n astTraceFilename) length() (uint64, error) {
	if !n.ownedMode {
		return n.params.length()
	}
	for i := uint64(0); ; i++ {
		b, e := n.byte(i)
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
func (s *ContextParams) MkASTManager(c ASTManagerConfig) (*ASTManager, error) {
	if e := s.sourceCallReady(); e != nil {
		return nil, e
	}
	if c.Symbols != s.context.symbols || c.Numerals != s.context.numerals {
		return nil, ErrParamsOwner
	}
	c.ProofMode = ASTProofDisabled
	if s.state.Proof {
		c.ProofMode = ASTProofEnabled
	}
	var filename *astTraceFilename
	if s.state.Trace {
		n := astTraceFilename{owned: s.traceFile.view(), ownedMode: true}
		filename = &n
	}
	s.executing = true
	defer func() { s.executing = false }()
	m, e := constructSourceASTManager(c, false, filename, nil)
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
