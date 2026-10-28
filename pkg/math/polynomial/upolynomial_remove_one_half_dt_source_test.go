package polynomial

// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Qualified signed-host TLS source numeral destruction fixture.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialRemoveOneHalfRootActualAmSourceStop(t *testing.T) {
	c, p, rt := algebraicNegTLSFixture(t, nil)
	z := p.config.Integer
	var big, twice, negative unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
	serviceMust(t, z.Mul(&big, &[]unumeral.Int{unumeral.NewInt(2)}[0], &twice))
	serviceMust(t, z.Copy(&negative, &big))
	serviceMust(t, z.Neg(&negative))
	input, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(input, []unumeral.Int{negative, twice}))
	serviceMust(t, z.Del(&big))
	serviceMust(t, z.Del(&twice))
	serviceMust(t, z.Del(&negative))
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	v, e := p.wholeView(input)
	serviceMust(t, e)
	oldCheckpoint := p.config.Checkpoint
	var lane umemory.Block
	defer func() {
		p.config.Checkpoint = oldCheckpoint
		contextTLSFinish(t, rt, func() error { return p.cleanupRemoveOneHalfRoot(false) })
		contextTLSFinish(t, rt, func() error {
			if lane == (umemory.Block{}) {
				return nil
			}
			return rt.Deallocate(&lane)
		})
		contextTLSFinish(t, rt, func() error {
			if input == (UPolynomial{}) {
				return nil
			}
			return p.DelPolynomial(&input)
		})
		contextTLSFinish(t, rt, func() error {
			if out == (UPolynomial{}) {
				return nil
			}
			return p.DelPolynomial(&out)
		})
		contextTLSFinish(t, rt, func() error { return p.cleanupRemoveOneHalfRoot(true) })
	}()
	checkpoints := 0
	var beforeBlocks uint32
	p.config.Checkpoint = func() bool {
		checkpoints++
		if checkpoints == 2 {
			q := &p.removeOneHalfRoot
			if q.constructed != 1 || (q.values[0]).Cell == 0 {
				t.Fatal("not actual published am")
			}
			contextTLSNearMinimum(t, rt)
			var e error
			lane, e = rt.Allocate(8)
			serviceMust(t, e)
			delta := int64(-1000000)
			if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(e)
			}
			beforeBlocks = rt.Stats().LiveBlocks
		}
		return oldCheckpoint()
	}
	cause := p.RemoveOneHalfRoot(v, out)
	if checkpoints != 2 || !errors.Is(cause, arena.ErrBounds) {
		t.Fatal("actual am source DT discarded", checkpoints, cause)
	}
	q := &p.removeOneHalfRoot
	if q.sourceRemaining != 1 || q.sourceCause == nil || !q.active || rt.Stats().LiveBlocks >= beforeBlocks {
		t.Fatal("actual consumed am/source cursor", q.sourceRemaining, q.sourceCause)
	}
	if (q.values[0]).Cell == 0 || (!z.Cell((q.values[0]).Cell).Released && !z.Cell((q.values[0]).Cell).CleanupPending) {
		t.Fatal("lost actual consumed am descriptor")
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{input, out}, []*unumeral.Int{&q.values[0]}, []umemory.Block{lane})
	before := q.sourceRemaining
	physical := rt.Stats()
	used := z.Budget().Used()
	if e = p.SourceFinalize(); e != q.sourceCause || q.sourceRemaining != before || rt.Stats() != physical || z.Budget().Used() != used {
		t.Fatal("source replay crossed older class", e)
	}
	if e = p.DelPolynomial(&input); e != ErrUPolynomialCleanupPending {
		t.Fatal("caller input DT crossed younger am", e)
	}
	r := p.removeOneHalfRootReceipt()
	oldBody, oldSource := q.bodyCause, q.sourceCause
	contextTLSFinish(t, rt, func() error { return p.cleanupRemoveOneHalfRootWithReceipt(&r, false) })
	contextTLSFinish(t, rt, func() error {
		if input == (UPolynomial{}) {
			return nil
		}
		return p.DelPolynomial(&input)
	})
	contextTLSFinish(t, rt, func() error {
		if out == (UPolynomial{}) {
			return nil
		}
		return p.DelPolynomial(&out)
	})
	contextTLSFinish(t, rt, func() error { return p.cleanupRemoveOneHalfRootWithReceipt(&r, true) })
	if q.sourceCause != oldSource || q.bodyCause != oldBody || !errors.Is(cause, oldSource) {
		t.Fatal("own rescue erased original source history")
	}
}
