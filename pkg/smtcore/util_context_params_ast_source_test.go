// Licensed under the MIT license; see LICENSE.txt.
// Source: src/cmd_context/context_params.cpp:198-209;
// src/ast/ast.cpp:1297-1334,2114-2128; src/ast/ast.h:1507-1508,1600.
// Source author: Leonardo de Moura.
package smtcore

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestContextParamsMkASTManagerActualTraceProofCoercionAndDebugActions(t *testing.T) {
	c, g, _, b := dynAckSourceFixture(t)
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer s.Close()
	name := astTracePath(t)
	for _, kv := range [][2]string{{"trace_file_name", name}, {"trace", "true"}, {"proof", "true"}, {"smtlib2_compliant", "true"}, {"debug_ref_count", "true"}} {
		k, v := kv[0], kv[1]
		paramsMust(t, s.Set(&k, &v))
	}
	ba := arena.NewBudget(1 << 30)
	m, e := s.MkASTManager(ast.ASTManagerConfig{Symbols: c.Symbols(), Numerals: c.Numerals(), Budget: ba})
	paramsMust(t, e)
	defer m.Close()
	mode, e := m.ProofMode()
	paramsMust(t, e)
	coercions, e := m.IntRealCoercions()
	paramsMust(t, e)
	if mode != ast.ASTProofEnabled || coercions || !m.DebugRefCount || m.TraceStream == nil || m.Format.TraceStream != m.TraceStream {
		t.Fatal("literal constructor and subsequent source setter actions", mode, coercions)
	}
	bs, e := m.MkBoolSort()
	paramsMust(t, e)
	v, e := m.MkVar(4, bs)
	paramsMust(t, e)
	paramsMust(t, m.IncRef(v))
	id, e := m.GetASTID(v)
	paramsMust(t, e)
	paramsMust(t, m.DecRef(v))
	if !m.DebugFreeIndices.Contains(id) {
		t.Fatal("debug flag never reached actual deletion map")
	}
	paramsMust(t, m.SourceFinalize())
	text, e := os.ReadFile(name)
	paramsMust(t, e)
	if !strings.Contains(string(text), "[mk-var] #") || !strings.HasSuffix(string(text), "[eof]\n") {
		t.Fatal("named caller actual trace body and source destructor", string(text))
	}
	if ba.Used() != m.Stats().AllocatedBytes {
		t.Fatal("source-completed named manager graph", ba.Used(), m.Stats())
	}
	paramsMust(t, m.Close())
	if ba.Used() != 0 {
		t.Fatal("explicit exact owned close", ba.Used())
	}
}

func TestContextParamsMkASTManagerEveryReachedConstructorPrefix(t *testing.T) {
	c, g, _, b := dynAckSourceFixture(t)
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer s.Close()
	name := astTracePath(t)
	for _, kv := range [][2]string{{"trace_file_name", name}, {"trace", "true"}, {"proof", "true"}, {"debug_ref_count", "true"}} {
		k, v := kv[0], kv[1]
		paramsMust(t, s.Set(&k, &v))
	}
	var sawFile, sawNested, success bool
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		ba := arena.NewBudget(1 << 30)
		ba.FailAfter(ordinal)
		m, e := s.MkASTManager(ast.ASTManagerConfig{Symbols: c.Symbols(), Numerals: c.Numerals(), Budget: ba})
		if m == nil {
			if e == nil || ba.Used() != 0 {
				t.Fatal("unnamed placement header", ordinal, e, ba.Used())
			}
		} else {
			state := m.ConstructorState()
			if ba.Used() != m.Stats().AllocatedBytes {
				t.Fatal("immediate caller constructor graph", ordinal, ba.Used(), m.Stats())
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) || state.Complete || m.DebugRefCount {
					t.Fatal("failed constructor reached caller setters", ordinal, e, state)
				}
				if m.TraceFile != nil {
					sawFile = true
					if m.TraceFile.Closed {
						t.Fatal("body stream prematurely destroyed")
					}
				}
				if m.Format != nil {
					sawNested = true
					if m.Format.Closed || m.Format.TraceStream != m.TraceStream {
						t.Fatal("actual completed nested borrow lost")
					}
				}
			} else {
				success = true
				if !state.Complete || !m.DebugRefCount || m.Format.TraceStream != m.TraceStream {
					t.Fatal("completed source call prefix")
				}
			}
			ba.FailAfter(-1)
			if e != nil {
				paramsMust(t, m.AbortConstruction())
			} else {
				paramsMust(t, m.Close())
			}
			if ba.Used() != 0 {
				t.Fatal("actual caller failed-root abort", ordinal, ba.Used())
			}
		}
		if success {
			break
		}
	}
	if !success || !sawFile || !sawNested {
		t.Fatal("public source constructor horizon", success, sawFile, sawNested)
	}
}
