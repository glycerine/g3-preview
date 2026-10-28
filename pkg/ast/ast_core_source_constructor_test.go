package ast

// Source: src/ast/ast.cpp:1297-1334; src/ast/ast.h:1454-1490.

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTSourceConstructorEveryPrefixRetainsBodyFilesAndUnwindsOnlyCompletedMembers(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	file, e := os.CreateTemp(".", "g3-ast-constructor-*.log")
	astTestMust(t, e)
	name, e := filepath.Abs(file.Name())
	astTestMust(t, e)
	astTestMust(t, file.Close())
	defer os.Remove(name)
	sawFile, sawFormat, succeeded := false, false, false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 30)
		b.FailAfter(ordinal)
		filename := uparams.LiteralParamsStringView(name)
		m, e := NewSourceASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b}, &filename)
		if m == nil {
			if e == nil || b.Used() != 0 {
				t.Fatal("failed wrapper has an unnamed acquisition", ordinal, e, b.Used())
			}
		} else {
			state := m.ConstructorState()
			if b.Used() != m.Stats().AllocatedBytes {
				t.Fatal("immediate constructor named graph", ordinal, b.Used(), m.Stats())
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) || state.Complete || state.BodyCause == nil {
					t.Fatal("failed constructor published", ordinal, e, state)
				}
				if m.TraceFile != nil {
					sawFile = true
					if m.TraceFile.Closed || m.TraceFile.stream.File == nil {
						t.Fatal("automatic member DT destroyed body-acquired fstream")
					}
				}
				if m.Format != nil {
					sawFormat = true
					if m.Format.Closed {
						t.Fatal("automatic member DT destroyed completed body-acquired format manager")
					}
				}
				if _, e2 := m.MkBoolSort(); e2 == nil {
					t.Fatal("incomplete source object became callable")
				}
			} else {
				succeeded = true
				if !state.Complete || m.Format == nil || m.TraceStream != m.Format.TraceStream || m.Format.traceOwner {
					t.Fatal("completed nested constructor borrow", state)
				}
			}
			b.FailAfter(-1)
			if e != nil {
				astTestMust(t, m.AbortConstruction())
			} else {
				astTestMust(t, m.Close())
			}
			if b.Used() != 0 {
				t.Fatal("constructor owning abort lost actual roots", ordinal, b.Used())
			}
		}
		if succeeded {
			break
		}
	}
	if !succeeded || !sawFile || !sawFormat {
		t.Fatal("actual source prefix horizon", succeeded, sawFile, sawFormat)
	}
}

func TestASTSourceConstructorCopyAndBorrowedNamespaceGuardsPreserveActualOwners(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	b := arena.NewBudget(1 << 28)
	filename := uparams.LiteralParamsStringView(astTracePath(t))
	m, e := NewSourceASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b}, &filename)
	paramsMust(t, e)
	defer m.Close()
	var copied ASTManager
	reflect.ValueOf(&copied).Elem().Set(reflect.ValueOf(m).Elem())
	before := b.Used()
	fd := m.TraceFile.stream.File
	for _, call := range []func() error{copied.Close, copied.SourceFinalize, copied.AbortConstruction} {
		if e = call(); !errors.Is(e, ErrASTHandle) {
			t.Fatal("accidental copied wrapper gained source/own authority", e)
		}
		if b.Used() != before || m.TraceFile.Closed || m.TraceFile.stream.File != fd {
			t.Fatal("copied guard altered actual named file or graph")
		}
	}
	var fileCopy astTraceFile
	reflect.ValueOf(&fileCopy).Elem().Set(reflect.ValueOf(m.TraceFile).Elem())
	if e = fileCopy.Close(); !errors.Is(e, ErrASTHandle) || b.Used() != before {
		t.Fatal("copied fd/header cleanup authority", e)
	}
	paramsMust(t, m.SourceFinalize())
	paramsMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("actual owner not consumed exactly once", b.Used())
	}
	if _, e = m.MkVar(0, ASTHandle{}); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source closed lifetime", e)
	}
}
