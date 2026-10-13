//go:build g3debug

package euclid

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestEuclideanSolvedQueueBorrowsHeapDebugAcrossReset(t *testing.T) {
	m := euclidNumeralManager(t)
	b := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	euclidMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, b)
	euclidMust(t, err)
	t.Cleanup(func() {
		euclidMust(t, debug.Close())
		euclidMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("borrowed debug owner leak", b.Used())
		}
	})
	euclidMust(t, debug.EnableDebug("heap"))
	s, err := NewSolverWithDebug(m, debug)
	euclidMust(t, err)
	defer func() { euclidMust(t, s.Close()) }()
	check := func() {
		t.Helper()
		for i := uint32(0); i < 2; i++ {
			id, e := s.MkVar()
			euclidMust(t, e)
			if id != i {
				t.Fatal("variable ID after setup reset", id, i)
			}
		}
		s.setup.solved[0], s.setup.solved[1] = 0, 1
		euclidMust(t, s.setup.queue.Insert(0))
		euclidMust(t, s.setup.queue.Insert(1))
		s.setup.solved[0], s.setup.solved[1] = 2, 1
		if err := s.setup.queue.Reserve(16); !errors.Is(err, diagnostic.ErrDebugNotificationBinding) {
			t.Fatal("Euclidean queue did not reach source heap CASSERT", err)
		}
		s.setup.solved[0], s.setup.solved[1] = 0, 1
	}
	check()
	euclidMust(t, s.Reset())
	check()
}
