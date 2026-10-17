package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicManagerStoppedYoungestMemberProtectsOlderOwners(t *testing.T) {
	cfg, _, _ := algebraicManagerFixture(t)
	m, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	z := cfg.Rationals.Manager.Integers()
	n := unumeral.NewInt(0)
	serviceMust(t, z.SetLimbs(&n, []uint32{1, 0, 1}))
	serviceMust(t, m.univariate.SetInts(m.addTmp, []unumeral.Int{n}))
	serviceMust(t, z.Del(&n))
	view, e := m.univariate.wholeView(m.addTmp)
	serviceMust(t, e)
	value, e := m.univariate.read(view, 0)
	serviceMust(t, e)
	ref := value.Cell
	cell := z.Cell(ref)
	requested := cell.Backing.Requested
	// This malformed actual allocation extent reaches the real guarded release;
	// it is deliberately a source bounds fixture, not a valid malloc refusal.
	cell.Backing.Requested = uint64(1) << 32
	z.Store(ref, cell)
	defer func() {
		current := z.Cell(ref)
		current.Backing.Requested = requested
		z.Store(ref, current)
		serviceMust(t, m.Close())
	}()
	coreHeader, ownerHeader := m.AlgebraicRootCore.header, m.header
	older := []UPolynomial{m.evalSignTmp, m.tmp3, m.tmp2, m.isolateTmp1}
	up, pm, binary, intervals := m.univariate, m.polynomial, m.binary, m.intervals
	e = m.Close()
	if !errors.Is(e, arena.ErrBounds) || m.privateFinalized || m.closed {
		t.Fatal("youngest member did not stop destruction", e)
	}
	if m.AlgebraicRootCore.header != coreHeader || m.header != ownerHeader || m.univariate != up || m.polynomial != pm || m.binary != binary || m.intervals != intervals {
		t.Fatal("source stop retired older owner headers")
	}
	for _, p := range older {
		if _, e = up.Size(p); e != nil {
			t.Fatal("older polynomial died before add_tmp", e)
		}
	}
	serviceMust(t, binary.Open())
	if _, e = intervals.Bounds(); e != nil {
		t.Fatal("older interval manager closed", e)
	}
	if e = m.polynomial.ready(); e != nil {
		t.Fatal("older polynomial manager closed", e)
	}
}
