package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMonomialTemporaryKeyPowerAssertionActions(t *testing.T) {
	for _, kind := range []string{"internal", "caller"} {
		t.Run(kind, func(t *testing.T) {
			for _, action := range []string{"T", "C", "disabled"} {
				t.Run(action, func(t *testing.T) {
					choices := action
					if action == "disabled" {
						choices = ""
					}
					output := polynomialAssertionActionsFixture(t, choices)
					if polynomialSourceDebug && action == "disabled" {
						serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
					}
					m := monomialFixture(t)
					for i := 0; i < 2; i++ {
						_, e := m.MkVar()
						serviceMust(t, e)
					}
					var caller monomialTemp
					tmp := &m.temps[2]
					if kind == "caller" {
						tmp = &caller
					}
					t.Cleanup(func() { serviceMust(t, m.tempCleanup(tmp)) })
					serviceMust(t, m.tempInit(tmp, []MonomialPower{{0, 2}}))
					// Initialize a real spare physical slot through the source setter while
					// keeping logical size1. Continue at228 can read it without undefined data.
					extra := MonomialPower{1, 7}
					serviceMust(t, m.tempSetPower(tmp, 1, extra))
					var got MonomialPower
					var body error
					if kind == "internal" {
						got, body = m.keyPower(monomialKey{monomialTemporaryKey, 2}, 1)
					} else {
						got, body = m.temporaryKeyPower(tmp, 1)
					}
					if polynomialSourceDebug && action == "T" {
						if !errors.Is(body, ErrMonomialAssertion) || got != (MonomialPower{}) {
							t.Fatal(got, body)
						}
						_, e := output.Seek(0, 0)
						serviceMust(t, e)
						text, e := io.ReadAll(output)
						serviceMust(t, e)
						if !strings.Contains(string(text), "Line: 228\nidx < size()") {
							t.Fatal(string(text))
						}
					} else if body != nil || got != extra {
						t.Fatal(got, body)
					}
					if tmp.size != 1 || tmp.capacity != monomialTempInitialCapacity || tmp.powers[1] != extra {
						t.Fatal("accessor changed temporary")
					}
				})
			}
		})
	}
}

func TestMonomialTemporaryKeyClosedAssertionOwner(t *testing.T) {
	m := monomialFixture(t)
	tmp := &m.temps[2]
	original := MonomialPower{0, 2}
	serviceMust(t, m.tempInit(tmp, []MonomialPower{original}))
	polynomialClosedCellAssertionContext(t)
	write := m.tempSetPower(tmp, 0, MonomialPower{1, 9})
	got, read := m.keyPower(monomialKey{monomialTemporaryKey, 2}, 0)
	if polynomialSourceDebug {
		if !errors.Is(write, arena.ErrClosed) || !errors.Is(read, arena.ErrClosed) || got != (MonomialPower{}) || tmp.powers[0] != original {
			t.Fatal(write, read, got, tmp.powers[0])
		}
	} else {
		serviceMust(t, write)
		serviceMust(t, read)
		if got != (MonomialPower{1, 9}) {
			t.Fatal(got)
		}
	}
	// get_ptr's raw hashing contains no get_power assertion of its own.
	_, e := m.tempHash(tmp)
	serviceMust(t, e)
}

func TestMonomialTemporaryGrowContinuedSourceBody(t *testing.T) {
	for _, preserve := range []bool{false, true} {
		name := "destructive"
		if preserve {
			name = "preserving"
		}
		t.Run(name, func(t *testing.T) {
			for _, action := range []string{"T", "C", "disabled"} {
				t.Run(action, func(t *testing.T) {
					choices := action
					if action == "disabled" {
						choices = ""
					}
					polynomialAssertionActionsFixture(t, choices)
					if polynomialSourceDebug && action == "disabled" {
						serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
					}
					m := monomialFixture(t)
					var tmp monomialTemp
					t.Cleanup(func() { serviceMust(t, m.tempCleanup(&tmp)) })
					original := MonomialPower{0, 3}
					serviceMust(t, m.tempInit(&tmp, []MonomialPower{original}))
					line := int32(521)
					if preserve {
						line = 528
					}
					body := m.tempGrow(&tmp, 64, preserve, line)
					if polynomialSourceDebug && action == "T" {
						if !errors.Is(body, ErrMonomialAssertion) || tmp.capacity != 128 || tmp.size != 1 || tmp.powers[0] != original {
							t.Fatal(body, tmp.capacity, tmp.size)
						}
					} else {
						serviceMust(t, body)
						expected := uint32(0)
						if preserve {
							expected = 1
						}
						if tmp.capacity != 64 || tmp.size != expected {
							t.Fatal(tmp.capacity, tmp.size)
						}
						if preserve && tmp.powers[0] != original {
							t.Fatal(tmp.powers[0])
						}
					}
				})
			}
		})
	}
}

func TestMonomialTemporaryZeroCapacityContinuedConstructor(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choices := "CCCC"
			if action == "T" {
				choices = "T"
			}
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			m := monomialFixture(t)
			var tmp monomialTemp
			t.Cleanup(func() { serviceMust(t, m.tempCleanup(&tmp)) })
			serviceMust(t, m.tempInit(&tmp, nil))
			body := m.tempGrow(&tmp, 0, false, 521)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(body, ErrMonomialAssertion) || tmp.capacity != 128 {
					t.Fatal(body, tmp.capacity)
				}
				return
			}
			serviceMust(t, body)
			// A constructed zero-capacity Go slice differs from an unconstructed nil
			// slice. Queries must not recreate the default128-capacity constructor.
			n, e := m.tempSize(&tmp)
			serviceMust(t, e)
			_, e = m.tempHash(&tmp)
			serviceMust(t, e)
			if n != 0 || tmp.capacity != 0 || tmp.powers == nil {
				t.Fatal(n, tmp.capacity, tmp.powers == nil)
			}
			if e = m.tempSetPower(&tmp, 0, MonomialPower{0, 1}); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(e)
			}
			if e = m.tempPushBack(&tmp, MonomialPower{0, 1}); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(e)
			}
			if tmp.size != 0 || tmp.capacity != 0 {
				t.Fatal("source overrun was replaced with growth", tmp.size, tmp.capacity)
			}
		})
	}
}
