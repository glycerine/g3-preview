package ast

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTNondeterminismTraceFollowsProcessDebugTag(t *testing.T) {
	m := astTestManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m.traceTags = tags
	budget := arena.NewBudget(^uint64(0))
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	astTestMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	astTestMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(debug)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		astTestMust(t, e)
		astTestMust(t, debug.Close())
		astTestMust(t, region.Close())
	})
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	existing, err := m.MkVar(17, sort)
	astTestMust(t, err)
	astTestPin(t, m, existing)
	id, err := m.GetASTID(existing)
	astTestMust(t, err)
	m.mu.Lock()
	candidate, err := m.allocateNode(ASTVar)
	if err != nil {
		m.mu.Unlock()
		t.Fatal(err)
	}
	m.varAt(candidate.slot).index = m.varAt(existing.slot).index
	m.varAt(candidate.slot).sort = m.varAt(existing.slot).sort
	hash, err := m.nodeHash(candidate)
	if err != nil {
		m.freeNodeStorage(candidate)
		m.mu.Unlock()
		t.Fatal(err)
	}
	m.nodeAt(candidate.slot).hash = hash
	bucket := m.table.buckets[hash]
	delete(m.table.buckets, hash)
	defer func() {
		m.table.buckets[hash] = bucket
		m.freeNodeStorage(candidate)
		m.mu.Unlock()
	}()
	check := func() error {
		_, e := m.registrationBefore(candidate)
		return e
	}
	if err = check(); err != nil || output.Len() != 0 {
		t.Errorf("disabled debug tag: %v, output %q", err, output.String())
	}
	astTestMust(t, debug.EnableDebug("nondet_bug"))
	if err = check(); ASTSourceDebug && !errors.Is(err, ErrASTDomain) || !ASTSourceDebug && err != nil {
		t.Errorf("enabled assertion without TRACE: %v", err)
	}
	if output.Len() != 0 {
		t.Errorf("disabled trace tag wrote %q", output.String())
	}
	tags.Enable("nondet_bug")
	astTestMust(t, debug.EnableAssertions(false))
	if err = check(); err != nil || output.Len() != 0 {
		t.Errorf("disabled assertion scanned canonical table: %v, output %q", err, output.String())
	}
	astTestMust(t, debug.EnableAssertions(true))
	err = check()
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) || !ASTSourceDebug && err != nil {
		t.Errorf("conditional debug scan: %v", err)
	}
	if ASTSourceDebug && tags.ShouldTrace("nondet_bug") {
		want := "-------- [nondet_bug] slow_not_contains src/ast/ast.cpp:1650 ---------\n" +
			fmt.Sprintf("id1:   %d, id2: 4294967295\nhash1: %d, hash2: %d\n", id, hash, hash) +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Errorf("source nondeterminism trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Errorf("erased nondeterminism trace wrote %q", output.String())
	}
}
