// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

const monomialHeaderBytes uint64 = 20

func monomialObjectSize(n uint32) uint32         { return uint32(20) + n*8 }
func monomialRefIncrement(n uint32) uint32       { return n + 1 }
func monomialExponentProduct(a, b uint32) uint32 { return a * b }

// Scratch owns only scalar powers. The hash is the project's cached interning
// hash, not a container header or a reference-counted monomial object.
type monomialTemp struct {
	powers []MonomialPower
	hash   uint32
}

func (m *MonomialManager) tempSize(t *monomialTemp) (uint32, error) {
	return uint32(len(t.powers)), nil
}
func (m *MonomialManager) tempSetSize(t *monomialTemp, n uint32) error {
	if uint64(n) > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if int(n) < len(t.powers) {
		clear(t.powers[n:])
		t.powers = t.powers[:n]
	} else {
		t.powers = append(t.powers, make([]MonomialPower, int(n)-len(t.powers))...)
	}
	return nil
}
func (m *MonomialManager) tempPower(t *monomialTemp, i uint32) (MonomialPower, error) {
	if uint64(i) >= uint64(len(t.powers)) {
		return MonomialPower{}, arena.ErrBounds
	}
	return t.powers[i], nil
}
func (m *MonomialManager) tempSetPower(t *monomialTemp, i uint32, p MonomialPower) error {
	if uint64(i) >= uint64(len(t.powers)) {
		if i == ^uint32(0) {
			return arena.ErrBounds
		}
		if e := m.tempSetSize(t, i+1); e != nil {
			return e
		}
	}
	t.powers[i] = p
	return nil
}
func (m *MonomialManager) tempInit(t *monomialTemp, p []MonomialPower) error {
	if uint64(len(p)) > uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	t.powers = append(t.powers[:0], p...)
	return nil
}

func (m *MonomialManager) tempHash(t *monomialTemp) (uint32, error) {
	t.hash = util.WordReader(uint32(len(t.powers))*8, 11, func(i uint32) uint32 {
		p := t.powers[i/2]
		if i%2 == 0 {
			return p.Var
		}
		return p.Degree
	})
	return t.hash, nil
}
func (m *MonomialManager) tempCleanup(t *monomialTemp) error { *t = monomialTemp{}; return nil }
