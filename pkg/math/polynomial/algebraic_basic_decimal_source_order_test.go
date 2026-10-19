// Source: algebraic_numbers.cpp:2733; mpq.cpp:157-161 raw declaration/getter order.
package polynomial

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBasicDecimalEntersRawScopeBeforeComponentReads(t *testing.T) {
	for _, negative := range []bool{false, true} {
		t.Run(fmt.Sprint("negative=", negative), func(t *testing.T) {
			for parts := 0; parts <= 2; parts++ {
				t.Run(fmt.Sprint(parts), func(t *testing.T) {
					c, _, _ := algebraicCompareFixtureProfile(t, true)
					q, z := c.config.Rationals, c.config.Rationals.Integers()
					var n unumeral.Int
					text := "18446744073709551617"
					if negative {
						text = "-" + text
					}
					serviceMust(t, z.SetString(&n, text))
					defer func() { serviceMust(t, z.Del(&n)) }()
					// This view owns its raw header only; the genuinely constructed NUM is
					// independently owned by n. No source cell or member construction is claimed.
					a := algebraicQueryRawCell(t, c, 0)
					l, e := c.location(&a)
					serviceMust(t, e)
					words := []uint32{uint32(n.Val), uint32(n.Cell), uint32(uint64(n.Cell) >> 32), 3, 0, 0}
					for i, w := range words {
						if i >= parts*3 {
							break
						}
						serviceMust(t, c.config.Allocator.WriteUint32(l.raw, uint64(4*i), w))
					}
					before := z.Cells.Stats().Live
					var out bytes.Buffer
					body := c.DisplayDecimal(&out, &a, 1)
					if parts < 2 {
						if !errors.Is(body, umemory.ErrRuntimeUninitialized) || out.Len() != 0 {
							t.Error("actual component guard/output prefix", body, out.String())
						}
						live := before
						if parts == 1 {
							live++
						}
						if z.Cells.Stats().Live != live {
							t.Error("NUM SET did not precede DEN read", z.Cells.Stats().Live, live)
						}
						zero := unumeral.NewRational()
						var retry bytes.Buffer
						e = q.DisplayDecimalInto(algebraicDecimalWriter{&retry}, &zero, 0, false)
						if !errors.Is(e, unumeral.ErrNumericPending) || retry.Len() != 0 {
							t.Error("RAW declarations were not entered before failed source reads", e, retry.String())
						}
						serviceMust(t, q.CleanupDecimalStream())
						if z.Cells.Stats().Live != before {
							t.Error("owning cleanup lost actual NUM copy")
						}
					} else {
						serviceMust(t, body)
						want := "6148914691236517205.6?"
						if negative {
							want = "-" + want
						}
						if out.String() != want {
							t.Error("independent stored rational decimal", out.String())
						}
					}
					serviceMust(t, q.CleanupDecimalStream())
				})
			}
		})
	}

}
