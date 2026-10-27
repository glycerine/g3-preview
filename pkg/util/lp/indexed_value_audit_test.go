// Source: src/util/lp/indexed_value.h:34-55; actual concrete sparse-matrix helpers.
package lp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPIndexedValueRationalConstructorAndAssignmentPublication(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		for _, operation := range []string{"constructor", "assignment"} {
			failures, succeeded := 0, false
			for ordinal := int64(0); ordinal < 64 && !succeeded; ordinal++ {
				if !t.Run(fmt.Sprintf("%v/%s/%d", direct, operation, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					p := h.pair(strings.Repeat("1234567890", 100)+"/"+strings.Repeat("9876543210", 30)+"7", "0")
					// The concrete matrix header provides the same owning storage and call
					// receipts as the real caller; no matrix algorithm reads these records.
					m := &RationalSparseMatrix{manager: h.m, Rows: make([][]RationalIndexedValue, 1)}
					if operation == "constructor" {
						m.work = NewIndexedRationalVector(h.m, 3)
					} else {
						m.Rows[0] = []RationalIndexedValue{{Value: numeral.NewRational(), Index: 11, Other: 17}, {Value: numeral.NewRational(), Index: 23, Other: 29}}
						h.must(h.m.Copy(&m.Rows[0][0].Value, &p.X))
						h.must(h.m.SetInt32(&m.Rows[0][1].Value, 31))
					}
					t.Cleanup(func() { h.budget.FailAfter(-1); h.must(m.Close()) })
					before, e := m.calls.begin(h.m)
					h.must(e)
					h.budget.FailAfter(ordinal)
					if operation == "constructor" {
						e = m.appendIndexed(&m.Rows[0], &p.X, 41, 43)
					} else {
						e = m.assignIndexed(&m.Rows[0][1], &m.Rows[0][0])
					}
					h.budget.FailAfter(-1)
					e = m.calls.after(h.m, before, e)
					if e != nil {
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						failures++
					} else {
						succeeded = true
					}
					if operation == "constructor" {
						if e != nil && len(m.Rows[0]) != 0 {
							t.Fatal("published unfinished element")
						}
						if e == nil {
							v := &m.Rows[0][0]
							if v.Index != 41 || v.Other != 43 {
								t.Fatal(v)
							}
							equal, e := h.m.Equal(&v.Value, &p.X)
							h.must(e)
							if !equal {
								t.Fatal("constructor value")
							}
						}
					} else {
						v := &m.Rows[0][1]
						if e != nil && (v.Index != 23 || v.Other != 29) {
							t.Fatal("metadata published before value copy", v)
						}
						if e == nil {
							if v.Index != 11 || v.Other != 17 {
								t.Fatal(v)
							}
							equal, e := h.m.Equal(&v.Value, &p.X)
							h.must(e)
							if !equal {
								t.Fatal("assignment value")
							}
						}
						equal, e := h.m.Equal(&m.Rows[0][0].Value, &p.X)
						h.must(e)
						if !equal {
							t.Fatal("mutated borrowed source")
						}
					}
					h.must(m.Close())
					lpExactVectorNamedCensus(t, h, nil)
				}) {
					t.FailNow()
				}
			}
			if !succeeded || failures == 0 {
				t.Fatal("incomplete indexed-value physical campaign", direct, operation, failures, succeeded)
			}
		}
	}
}

func TestLPIndexedValueByValueSetterAndAliasedArgument(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			failures, succeeded := 0, false
			for ordinal := int64(0); ordinal < 64 && !succeeded; ordinal++ {
				if !t.Run(fmt.Sprintf("direct=%v/alias=%v/%d", direct, alias, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					old := h.pair(strings.Repeat("9876543210", 70)+"/17", "0")
					value := h.pair(strings.Repeat("1234567890", 100)+"/19", "0")
					m := &RationalSparseMatrix{manager: h.m, Rows: [][]RationalIndexedValue{{{Value: numeral.NewRational(), Index: 37, Other: 41}}}, work: NewIndexedRationalVector(h.m, 1)}
					h.must(h.m.Copy(&m.Rows[0][0].Value, &old.X))
					src := &value.X
					want := value
					if alias {
						src = &m.Rows[0][0].Value
						want = old
					}
					t.Cleanup(func() { h.budget.FailAfter(-1); h.must(m.Close()) })
					before, e := m.calls.begin(h.m)
					h.must(e)
					h.budget.FailAfter(ordinal)
					e = m.setIndexedValue(&m.Rows[0][0].Value, src, 0)
					h.budget.FailAfter(-1)
					e = m.calls.after(h.m, before, e)
					if e != nil {
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						failures++
					} else {
						succeeded = true
						equal, e := h.m.Equal(&m.Rows[0][0].Value, &want.X)
						h.must(e)
						if !equal {
							t.Fatal("setter/alias value")
						}
					}
					if m.Rows[0][0].Index != 37 || m.Rows[0][0].Other != 41 {
						t.Fatal("value setter changed metadata")
					}
					h.must(m.Close())
					lpExactVectorNamedCensus(t, h, nil)
				}) {
					t.FailNow()
				}
			}
			if !succeeded || failures == 0 {
				t.Fatal("incomplete setter physical campaign", direct, alias, failures, succeeded)
			}
		}
	}
}
