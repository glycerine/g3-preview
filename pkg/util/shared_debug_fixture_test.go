package util

import (
	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	"testing"
)

func debugFixture(t *testing.T) (*udiag.DebugContext, *arena.BytesRegion, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	d, e := udiag.NewDebugContext(r, b)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, d.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("remaining debug graph", b.Used())
		}
	})
	return d, r, b
}
