// Source: src/util/mpz.cpp:1432-1435.
package util

import (
	utext "github.com/glycerine/g3/pkg/util/text"
	"strings"
	"testing"
)

func TestOwnedStringStreamSourceIntegerEveryColdReservation(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, text := range []string{"-2147483648", "123456789012345678901234567890123456789", "-" + strings.Repeat("9", 600)} {
			success := false
			for fail := int64(0); fail < 100; fail++ {
				p, b := mpqInfFixture(t, sync)
				x := mpqInfValue(t, p, text, "0")
				var out utext.OwnedStringStream
				serviceMust(t, out.Construct(b))
				b.FailAfter(fail)
				var result utext.OwnedString
				e := p.m.Z.insertSourceInteger(&out, &x.First.Num)
				if e == nil {
					e = out.Str(&result)
				}
				qopsAuditLiveOwnership(t, p.m, []*Int{&x.First.Num, &x.First.Den, &x.Second.Num, &x.Second.Den})
				actual := p.header + p.m.HeaderBytes + integerActualOwnedBytes(p.m.Z) + uint64(out.Buffer.String.StorageBytes()+result.StorageBytes())
				if b.Used() != actual {
					t.Fatal(sync, len(text), fail, "immediate ownership", b.Used(), actual)
				}
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(sync, len(text), fail, e)
					}
				} else {
					got, err := result.View().String()
					serviceMust(t, err)
					if got != text {
						t.Fatal("source integer result", got, text)
					}
					success = true
				}
				b.FailAfter(-1)
				serviceMust(t, p.m.Z.Natural.cleanupOutputFormatting())
				result.OwnAbort()
				out.OwnAbort()
				serviceMust(t, p.Del(&x))
				serviceMust(t, p.Close())
				if b.Used() != 0 {
					t.Fatal("source integer refund", b.Used())
				}
				if success {
					break
				}
			}
			if !success {
				t.Fatal("no successful source integer", sync, len(text))
			}
		}
	}
}
