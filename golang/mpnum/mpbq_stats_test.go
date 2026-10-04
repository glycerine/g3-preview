// Licensed under the MIT license.
// Source: src/util/mpbq.cpp, src/util/mpbq.h.
package mpnum

import (
	"math"
	"testing"
)

func TestBinaryStatsFailedScratchNotLive(t *testing.T) {
	z, budget := mpzPublicationDirectOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(m.Close)
	var src Int
	serviceMust(t, z.SetUint64(&src, math.MaxUint64))
	t.Cleanup(func() { serviceMust(t, z.Del(&src)) })
	budget.FailAfter(0)
	e = z.Copy(&m.tmp, &src)
	integerPhysicalAudit(t, z, &src, &m.tmp)
	stats, statsErr := m.Stats()
	budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || m.tmp.cell != z.failedNullRef {
		t.Fatal("real scratch copy refusal", e, m.tmp)
	}
	if statsErr != nil || stats.LiveScratchNumerators != 0 {
		t.Fatal("failed scratch token counted as live", stats, statsErr)
	}
	serviceMust(t, z.Del(&m.tmp))
	serviceMust(t, z.Copy(&m.tmp, &src))
	stats, statsErr = m.Stats()
	if statsErr != nil || stats.LiveScratchNumerators != 1 {
		t.Fatal("live scratch omitted", stats, statsErr)
	}
}
