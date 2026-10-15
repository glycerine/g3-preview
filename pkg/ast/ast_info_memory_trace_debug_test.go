//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTInfoMemoryWrapperRegisteredSortAndDecl(t *testing.T) {
	sy, q, b := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b, TraceTags: tags})
	astTestMust(t, err)
	tags.Enable("memory")
	si, err := m.NewSortInfo(0, 71, nil, FiniteSortSize(11), false)
	astTestMust(t, err)
	s, err := m.MkSort(astTestSymbol(t, m, "trace-info-sort"), si)
	astTestMust(t, err)
	astTestMust(t, m.IncRef(s))
	sr := m.sortAt(s.slot).info
	sortRecord := m.infoAt(sr.slot)
	astTestMust(t, m.DelSortInfo(&si))
	fi, err := m.NewFuncDeclInfo(NullFamilyID, 42, nil, ASTFuncFlags{}, false)
	astTestMust(t, err)
	d, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "trace-info-decl"), nil, s, fi)
	astTestMust(t, err)
	astTestMust(t, m.IncRef(d))
	dr := m.declAt(d.slot).info
	declRecord := m.infoAt(dr.slot)
	astTestMust(t, m.DelFuncDeclInfo(&fi))
	astTestMust(t, m.DecRef(d))
	astTestMust(t, m.DecRef(s))
	size := strconv.FormatUint(uint64(reflect.TypeOf(astInfoRecord{}).Size()), 10)
	alloc := func(r *astInfoRecord, line int, name string) string {
		return "-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\n" +
			"alloc " + fmt.Sprintf("%p", r) + " src/ast/ast.cpp:" + strconv.Itoa(line) + " " + name + " " + size + "\n" +
			"------------------------------------------------\n"
	}
	dealloc := func(r *astInfoRecord, line int) string {
		return "-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\n" +
			"dealloc " + fmt.Sprintf("%p", r) + " src/ast/ast.cpp:" + strconv.Itoa(line) + "\n" +
			"------------------------------------------------\n"
	}
	want := alloc(sortRecord, 1702, "sort_info") + alloc(declRecord, 1708, "func_decl_info") +
		dealloc(declRecord, 1820) + dealloc(sortRecord, 1813)
	if got := output.String(); got != want || sortRecord.memoryTraceAllocated || declRecord.memoryTraceAllocated {
		t.Fatalf("source info records:\n got %q\nwant %q", got, want)
	}
	tags.Disable("memory")
	astTestMust(t, m.Close())
}

func TestASTInfoMemoryWrapperDebugRetainsSourceAllocation(t *testing.T) {
	sy, q, b := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b, TraceTags: tags})
	astTestMust(t, err)
	astTestMust(t, m.EnableDebugRefCount())
	tags.Enable("memory")
	si, err := m.NewSortInfo(0, 71, nil, FiniteSortSize(11), false)
	astTestMust(t, err)
	s, err := m.MkSort(astTestSymbol(t, m, "retained-info-sort"), si)
	astTestMust(t, err)
	astTestMust(t, m.IncRef(s))
	r := m.infoAt(m.sortAt(s.slot).info.slot)
	astTestMust(t, m.DelSortInfo(&si))
	astTestMust(t, m.DecRef(s))
	if !r.retainedByDebug || !r.memoryTraceAllocated || bytes.Count(output.Bytes(), []byte("alloc ")) != 1 || bytes.Contains(output.Bytes(), []byte("dealloc ")) {
		t.Fatalf("debug metadata source retention: %q", output.String())
	}
	astTestMust(t, m.Close())
	if bytes.Contains(output.Bytes(), []byte("dealloc "+fmt.Sprintf("%p", r)+" ")) {
		t.Fatalf("owning close invented retained info source dealloc: %q", output.String())
	}
}
