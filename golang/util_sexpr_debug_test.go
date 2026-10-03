//go:build g3debug

// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"strings"
	"testing"
)

func TestSexprDebugBVInvariantAndShortCircuit(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	for _, tc := range []struct {
		c     string
		base  int64
		valid bool
	}{{"0", 16, true}, {"15", 16, true}, {"16", 16, false}, {"-1", 16, false}, {"0", 2, true}, {"1", 2, true}, {"2", 2, false}} {
		c := sxNumber(t, q, tc.c)
		e := m.sexprBVDiagnostic(&c, tc.base)
		if tc.valid {
			sxMust(t, e)
		} else if !errors.Is(e, ErrSexprBVInvariant) {
			t.Fatal(tc, e)
		}
		sxMust(t, q.Del(&c))
	}
	negative := sxNumber(t, q, "-1/1684996666696914987166688442938726917102321526408785780068975640579")
	before, e := q.Stats()
	sxMust(t, e)
	m.config.Budget.FailAfter(0)
	e = m.sexprBVDiagnostic(&negative, 16)
	if !errors.Is(e, ErrSexprBVInvariant) {
		t.Fatal("lower inequality must short circuit", e)
	}
	after, err := q.Stats()
	sxMust(t, err)
	if before.LiveCells != after.LiveCells || before.LivePayloads != after.LivePayloads {
		t.Fatal(before, after)
	}
	m.config.Budget.FailAfter(-1)
	sxMust(t, q.Del(&negative))
}
func TestSexprDebugBVComparisonFailureRoots(t *testing.T) {
	failures := 0
	success := false
	for prefix := int64(0); prefix < 256; prefix++ {
		b := arena.NewBudget(1 << 26)
		m, _, q := sxOwner(t, b)
		text := "1" + strings.Repeat("0", 400)
		den := "3" + strings.Repeat("0", 399) + "1"
		c := sxNumber(t, q, text+"/"+den)
		var filler [16]mpnum.OwnedRational
		for i := range filler {
			filler[i], _ = q.NewValue()
			sxMust(t, q.Copy(&filler[i], &c))
		}
		before, e := q.Stats()
		sxMust(t, e)
		b.FailAfter(prefix)
		e = m.sexprBVDiagnostic(&c, 16)
		after, err := q.Stats()
		sxMust(t, err)
		if before.LiveCells != after.LiveCells || before.LivePayloads != after.LivePayloads {
			t.Fatal("debug comparison temp leak", prefix, before, after)
		}
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			failures++
		} else {
			success = true
		}
		b.FailAfter(-1)
		for i := range filler {
			sxMust(t, q.Del(&filler[i]))
		}
		sxMust(t, q.Del(&c))
		sxMust(t, m.Close())
		sxMust(t, q.Close())
		sxMust(t, m.config.Symbols.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal(success, failures)
	}
}
