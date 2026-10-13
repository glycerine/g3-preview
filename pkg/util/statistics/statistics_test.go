package statistics

import (
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestContributionsCopyResetAndRawAccess(t *testing.T) {
	var source Statistics
	source.UpdateUint("ignored", 0)
	source.UpdateDouble("ignored", math.Copysign(0, -1))
	source.UpdateUint("first", 7)
	source.UpdateDouble("second", 2.5)
	if source.Size() != 2 || !source.IsUint(0) || source.IsUint(1) {
		t.Fatal("raw domains", source.Size())
	}
	if source.Key(0) != "first" || source.UintValue(0) != 7 || source.Key(1) != "second" || source.DoubleValue(1) != 2.5 {
		t.Fatal("raw values")
	}
	var copy Statistics
	copy.Copy(source)
	copy.Copy(source)
	if copy.Size() != 4 || copy.UintValue(1) != 7 || copy.DoubleValue(3) != 2.5 {
		t.Fatal("append copy")
	}
	copy.Reset()
	if copy.Size() != 0 {
		t.Fatal("reset")
	}
}

func TestRenderingAggregationOrderingAndShadow(t *testing.T) {
	var s Statistics
	s.UpdateUint(":z key", 2)
	s.UpdateUint("alpha", ^uint32(0))
	s.UpdateUint("alpha", 2)
	s.UpdateDouble("beta", 1.25)
	s.UpdateDouble("alpha", 9.5)
	s.UpdateDouble(":z key", 0.75)

	var normal strings.Builder
	if err := s.Display(&normal); err != nil {
		t.Fatal(err)
	}
	const wantNormal = "z key: 2\nz key: 2\nalpha: 1\nalpha: 1\nbeta:  1.25\n"
	if normal.String() != wantNormal {
		t.Fatalf("normal:\n%q\nwant:\n%q", normal.String(), wantNormal)
	}

	var smt2 strings.Builder
	if err := s.DisplaySMT2(&smt2); err != nil {
		t.Fatal(err)
	}
	const wantSMT2 = "(:z-key 2\n :z-key 2\n :alpha 1\n :alpha 1\n :beta  1.25)\n"
	if smt2.String() != wantSMT2 {
		t.Fatalf("smt2:\n%q\nwant:\n%q", smt2.String(), wantSMT2)
	}

	var internal strings.Builder
	if err := s.DisplayInternal(&internal); err != nil {
		t.Fatal(err)
	}
	const wantInternal = "Z_KEY 2\nALPHA 1\nBETA 1.25\nALPHA 9.5\nZ_KEY 0.75\n"
	if internal.String() != wantInternal {
		t.Fatalf("internal:\n%q\nwant:\n%q", internal.String(), wantInternal)
	}
}

func TestUint64Threshold(t *testing.T) {
	var s Statistics
	AddResourceLimit(&s, uint64(^uint32(0)))
	s.UpdateUint64("large", uint64(^uint32(0))+1)
	if s.Size() != 2 || s.Key(0) != "rlimit count" || !s.IsUint(0) || s.IsUint(1) || s.UintValue(0) != ^uint32(0) || s.DoubleValue(1) != float64(uint64(^uint32(0))+1) {
		t.Fatal("uint64 domains")
	}
}

func TestMemoryStatisticsSourceCounterDomains(t *testing.T) {
	var s Statistics
	addMemory(&s, 2*1024*1024+1024*1024/2, 4*1024*1024, uint64(^uint32(0))+1)
	if s.Size() != 3 || s.Key(2) != "num allocs" || s.IsUint(2) || s.DoubleValue(2) != float64(uint64(^uint32(0))+1) {
		t.Fatal("allocation count domain")
	}
	if s.Key(0) != "max memory" || s.DoubleValue(0) != 4 || s.Key(1) != "memory" || s.DoubleValue(1) != 2.5 {
		t.Fatal("memory conversion")
	}
	if memoryMegabytes(^uint64(0)) != float64((^uint64(0)-99)/(1024*1024))/100 {
		t.Fatal("unsigned source conversion overflow")
	}
}

func TestMemoryStatisticsReadsSharedSourceProcess(t *testing.T) {
	p := memory.NewProcessAccounting()
	firstBudget, secondBudget := arena.NewBudget(1<<22), arena.NewBudget(1<<22)
	first, err := memory.NewRuntime(memory.RuntimeConfig{Budget: firstBudget, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	second, err := memory.NewRuntime(memory.RuntimeConfig{Budget: secondBudget, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	a, err := first.Allocate(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.Allocate(1 << 19)
	if err != nil {
		t.Fatal(err)
	}
	if p.AllocationSize() != (1<<20)+(1<<19)+16 || p.MaximumUsed() != p.AllocationSize() || p.AllocationCount() != 2 {
		t.Fatal("source global counters", p.AllocationSize(), p.MaximumUsed(), p.AllocationCount())
	}
	var stats Statistics
	if err := AddMemory(&stats, p); err != nil || stats.Size() != 3 || stats.Key(0) != "num allocs" || stats.UintValue(0) != 2 || stats.Key(1) != "max memory" || stats.DoubleValue(1) != 1.5 || stats.Key(2) != "memory" || stats.DoubleValue(2) != 1.5 {
		t.Fatal("shared process stats", err, stats)
	}
	if err := first.Deallocate(&a); err != nil {
		t.Fatal(err)
	}
	stats.Reset()
	if err := AddMemory(&stats, p); err != nil || stats.Size() != 3 || stats.DoubleValue(1) != 1.5 || stats.DoubleValue(2) != 0.5 || stats.UintValue(0) != 2 {
		t.Fatal("source peak survives release", err, stats)
	}
	if err := second.Deallocate(&b); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if firstBudget.Used() != 0 || secondBudget.Used() != 0 {
		t.Fatal("retained process budgets", firstBudget.Used(), secondBudget.Used())
	}
	if err := AddMemory(nil, p); err != arena.ErrConfig {
		t.Fatal("missing statistics", err)
	}
	if err := AddMemory(&stats, nil); err != arena.ErrConfig {
		t.Fatal("missing process", err)
	}
}
