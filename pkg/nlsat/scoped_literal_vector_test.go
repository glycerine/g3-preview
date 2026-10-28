package nlsat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func scopedLiteralFixture(t *testing.T) (*Evaluator, *searchState, *scopedLiteralVector) {
	t.Helper()
	e, r := searchStateFixture(t)
	v := &scopedLiteralVector{factory: r.store.atoms}
	t.Cleanup(func() { assignmentMust(t, v.close()) })
	return e, r, v
}
func scopedFreshLiteral(t *testing.T, e *Evaluator, r *searchState, power uint32) sat.Literal {
	t.Helper()
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, power})
	b, err := r.store.atoms.mkInequality(AtomGT, []polynomial.Polynomial{p}, []bool{false})
	assignmentMust(t, err)
	return sat.MakeLiteral(b, false)
}
func TestScopedLiteralVectorSetShrinkAndMultiplicity(t *testing.T) {
	e, r, v := scopedLiteralFixture(t)
	l := scopedFreshLiteral(t, e, r, 1)
	for i := 0; i < 3; i++ {
		assignmentMust(t, v.push(l))
	}
	assignmentMust(t, v.set(0, l.Negated()))
	if arithmeticAtomHeader(r.store.atoms.bools.atoms[l.Variable()]).refs != 3 {
		t.Fatal("SET/self-association leaked an occurrence")
	}
	assignmentMust(t, v.shrink(1))
	if !reflect.DeepEqual(v.result, []sat.Literal{l.Negated()}) || arithmeticAtomHeader(r.store.atoms.bools.atoms[l.Variable()]).refs != 1 {
		t.Fatal("SHRINK multiplicity", v.result)
	}
	assignmentMust(t, v.reset())
	if !r.store.atoms.bools.dead[l.Variable()] {
		t.Fatal("last vector lease retained atom")
	}
}
func TestScopedLiteralVectorSetBoundsOwnsOnlyIncoming(t *testing.T) {
	e, r, v := scopedLiteralFixture(t)
	assignmentMust(t, v.push(TrueLiteral))
	l := scopedFreshLiteral(t, e, r, 1)
	err := v.set(7, l)
	if !errors.Is(err, arena.ErrBounds) || !v.setActive || !v.incomingOwned || v.setOldRead || arithmeticAtomHeader(r.store.atoms.bools.atoms[l.Variable()]).refs != 1 {
		t.Fatal("SET did not acquire before bounds read", err)
	}
	assignmentMust(t, v.cleanup())
	if !reflect.DeepEqual(v.result, []sat.Literal{TrueLiteral}) || !r.store.atoms.bools.dead[l.Variable()] || v.setActive {
		t.Fatal("SET abort adopted older output or lost incoming lease")
	}
}
func TestScopedLiteralVectorStoppedSetAndShrinkContinueExactDEC(t *testing.T) {
	for _, set := range []bool{false, true} {
		t.Run(map[bool]string{false: "shrink", true: "set"}[set], func(t *testing.T) {
			atomDebugFixture(t)
			e, r, v := scopedLiteralFixture(t)
			a := scopedFreshLiteral(t, e, r, 1)
			b := scopedFreshLiteral(t, e, r, 2)
			c := scopedFreshLiteral(t, e, r, 3)
			if set {
				assignmentMust(t, v.push(a))
			} else {
				for _, l := range []sat.Literal{c, a, b} {
					assignmentMust(t, v.push(l))
				}
			}
			// Qualified invalid Boolean-watch state reaches the actual source guard.
			r.store.atoms.bools.watches[a.Variable()] = []*Clause{r.store.originals[0]}
			var err error
			if set {
				err = v.set(0, b)
			} else {
				err = v.shrink(1)
			}
			want := ErrAtomCacheLive
			if diagnostic.DebugBuild {
				want = diagnostic.ErrDebugAssertion
			}
			if !errors.Is(err, want) || arithmeticAtomHeader(r.store.atoms.bools.atoms[a.Variable()]).refs != 0 {
				t.Fatal("did not retain consumed last DEC", err)
			}
			if set {
				if !v.setOldDone || v.setRetiring == nil || v.result[0] != a || arithmeticAtomHeader(r.store.atoms.bools.atoms[b.Variable()]).refs != 1 {
					t.Fatal("SET lost unpublished incoming prefix")
				}
			} else if !v.shrinking || v.resetIndex != 2 || len(v.result) != 3 {
				t.Fatal("SHRINK published its extent before tail destruction")
			}
			r.store.atoms.bools.watches[a.Variable()] = nil
			assignmentMust(t, v.cleanup())
			if !r.store.atoms.bools.dead[a.Variable()] || !r.store.atoms.bools.dead[b.Variable()] {
				t.Fatal("owning continuation replayed or omitted a drop")
			}
			if set {
				if !reflect.DeepEqual(v.result, []sat.Literal{sat.DefaultLiteral()}) {
					t.Fatal("aborted SET published incoming", v.result)
				}
			} else if !reflect.DeepEqual(v.result, []sat.Literal{c}) {
				t.Fatal("SHRINK lost surviving prefix", v.result)
			}
		})
	}
}
