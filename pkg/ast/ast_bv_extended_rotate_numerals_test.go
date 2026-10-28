package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:2020-2038;
// src/util/rational.h:101,202-206.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"testing"
)

func TestASTBitVectorExtendedRotationEveryActualReservation(t *testing.T) {
	for _, tc := range []struct {
		text  string
		width uint32
	}{{"73786976294838206467", 65}, {"18446744073709551617", 257}, {"0", 65}} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d/%d", tc.text, tc.width, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(tc.width)
				astTestMust(t, e)
				arg := bvRawOffsetTestNumeral(t, m, fid, sort, tc.text, tc.width)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorExtendedRotateNumerals(m)
				var width, shift uint32
				if body == nil {
					_, body = n.Read(arg, &width)
				}
				if body == nil {
					shift, body = n.Shift(width)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					value, _ := new(big.Int).SetString(tc.text, 10)
					want := new(big.Int).Rem(value, new(big.Int).SetUint64(uint64(tc.width))).Uint64()
					text, e := m.numerals.String(&n.values[0])
					astTestMust(t, e)
					if width != tc.width || uint64(shift) != want || text != tc.text || !n.placed[0] || n.placed[1] || n.placed[2] {
						t.Fatal("raw read/remainder/lifetimes", width, shift, want, text, n.placed)
					}
					astTestMust(t, n.SourceEnd())
				} else {
					failures++
					var f umemory.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
						t.Fatal(body)
					}
				}
				if n != nil {
					copy := *n
					if e = copy.Close(); e != ErrASTHandle {
						t.Fatal("copied owner", e)
					}
					beforeBody, beforeSource, _ := n.scalar.Causes()
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					afterBody, afterSource, _ := n.scalar.Causes()
					if beforeBody != afterBody || beforeSource != afterSource {
						t.Fatal("own cleanup rewrote source history")
					}
				}
			})
		}
		if !complete || failures == 0 {
			t.Fatal("actual reservation campaign incomplete", tc, complete, failures)
		}
	}
}

func TestASTBitVectorExtendedRotationSignedRemainderThenUnsignedRead(t *testing.T) {
	if ASTSourceDebug {
		debug, _, _ := debugDiagnosticFixture(t)
		previous, err := diagnostic.BindProcessDebugContext(debug)
		astTestMust(t, err)
		t.Cleanup(func() {
			_, err := diagnostic.BindProcessDebugContext(previous)
			astTestMust(t, err)
		})
	}
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(65)
	astTestMust(t, e)
	arg := bvRawOffsetTestNumeral(t, m, fid, sort, "-3", 65)
	n, e := NewBitVectorExtendedRotateNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	var width uint32
	yes, e := n.Read(arg, &width)
	astTestMust(t, e)
	if !yes {
		t.Fatal("raw negative metadata")
	}
	shift, body := n.Shift(width)
	if ASTSourceDebug {
		if !errors.Is(body, diagnostic.ErrDebugAssertion) || !n.placed[1] || !n.placed[2] {
			t.Fatal("original unsigned assertion before expression end", body, n.placed)
		}
		text, e := m.numerals.String(&n.values[2])
		astTestMust(t, e)
		if text != "-3" {
			t.Fatal("remainder changed to Euclidean modulo", text)
		}
	} else {
		astTestMust(t, body)
		signed := int64(-3)
		want := uint32(uint64(signed) % 65)
		if shift != want {
			t.Fatal("original small unsigned cast", shift, want)
		}
		astTestMust(t, n.SourceEnd())
	}
}
