package numeral

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

type parserScopeBoundaryWriter struct {
	t          *testing.T
	runtime    *umemory.Runtime
	lane       umemory.Block
	injected   bool
	setupErr   error
	targetPart string
	seenTarget bool
}

func (w *parserScopeBoundaryWriter) Write(p []byte) (int, error) {
	if w.targetPart != "" && string(p) == w.targetPart {
		w.seenTarget = true
	}
	if !w.injected && (w.targetPart == "" || w.seenTarget) && string(p) == "------------------------------------------------\n" {
		w.injected = true
		contextTLSNearMinimum(w.t, w.runtime)
		w.lane, w.setupErr = w.runtime.Allocate(8)
		if w.setupErr == nil {
			delta := int64(-100001) - w.runtime.State().LocalSize
			if e := w.runtime.Reallocate(&w.lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
				w.setupErr = e
				if e == nil {
					w.setupErr = errors.New("TLS boundary reallocation unexpectedly succeeded")
				}
			}
		}
	}
	return len(p), nil
}

func TestRationalParserActualOuterScopedDeletionRefusal(t *testing.T) {
	gate := diagnostic.NewTraceTags()
	gate.Enable("mpq_set")
	if !gate.ShouldTrace("mpq_set") {
		t.Skip("source trace call site is compiled out")
	}
	budget := arena.NewBudget(1 << 26)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{
		Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: budget, PageSize: 1},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	serviceMust(t, e)
	w := &parserScopeBoundaryWriter{t: t, runtime: runtime}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpq_set")
	serviceMust(t, m.BindTrace(tags))
	a := NewRational()
	input := strings.Repeat("9", 600)
	e = m.SetString(&a, input)
	if !w.injected || w.setupErr != nil || !errors.Is(e, arena.ErrBounds) || !m.parseOuterPending || m.parseOuterTemporary.Cell == 0 || m.parseSuffixPending || m.NTmp != NewInt(1) {
		t.Fatal("actual parser outer destructor refusal", w.injected, w.setupErr, e, m.parseOuterPending, m.parseOuterTemporary, m.NTmp, runtime.State())
	}
	parsed, e := m.Z.String(&a.Num)
	if e != nil || parsed != input || a.Den != NewInt(1) {
		t.Fatal("parser body must complete before scoped deletion", len(parsed), e, a.Den)
	}
	if e = m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatal("source finalization discarded stopped parser temporary", e)
	}
	recovery, e := runtime.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupParserTemporaries())
	serviceMust(t, m.Del(&a))
	serviceMust(t, runtime.Deallocate(&w.lane))
	state := runtime.State()
	delta := int64(-500000) - state.GlobalSize - state.LocalSize
	if e = runtime.Reallocate(&recovery, uint64(200000)+uint64(delta)); e == nil || runtime.State().GlobalSize != -500000 || runtime.State().LocalSize != 0 {
		t.Fatal("restore TLS accounting range before owner teardown", e, runtime.State())
	}
	serviceMust(t, runtime.Deallocate(&recovery))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, runtime.Close())
	if budget.Used() != 0 {
		t.Fatal("actual parser scope leaked ownership", budget.Used())
	}
}

func TestRationalParserActualExponentScopedDeletionRefusal(t *testing.T) {
	gate := diagnostic.NewTraceTags()
	gate.Enable("mpq_set")
	if !gate.ShouldTrace("mpq_set") {
		t.Skip("source trace call site is compiled out")
	}
	budget := arena.NewBudget(1 << 26)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{
		Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: budget, PageSize: 1},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	serviceMust(t, e)
	w := &parserScopeBoundaryWriter{t: t, runtime: runtime, targetPart: "260"}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpq_set")
	serviceMust(t, m.BindTrace(tags))
	a := NewRational()
	e = m.SetString(&a, "0e600")
	if !w.injected || w.setupErr != nil || !errors.Is(e, arena.ErrBounds) || !m.parseExp.pending || m.parseExp.value.Num.Cell == 0 || m.parseTen.pending || m.parseB.pending || m.parseSuffixPending || m.parseOuterPending {
		t.Fatal("actual parser _exp destructor refusal", w.injected, w.setupErr, e, m.parseExp, m.parseTen, m.parseB, runtime.State())
	}
	if a.Num != NewInt(0) || a.Den != NewInt(1) {
		t.Fatal("zero scaling must finish before _exp cleanup", a)
	}
	if e = m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatal("source finalization discarded stopped _exp", e)
	}
	recovery, e := runtime.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupParserTemporaries())
	serviceMust(t, m.Del(&a))
	serviceMust(t, runtime.Deallocate(&w.lane))
	state := runtime.State()
	delta := int64(-500000) - state.GlobalSize - state.LocalSize
	if e = runtime.Reallocate(&recovery, uint64(200000)+uint64(delta)); e == nil || runtime.State().GlobalSize != -500000 || runtime.State().LocalSize != 0 {
		t.Fatal("restore TLS accounting range before owner teardown", e, runtime.State())
	}
	serviceMust(t, runtime.Deallocate(&recovery))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, runtime.Close())
	if budget.Used() != 0 {
		t.Fatal("actual exponent scope leaked ownership", budget.Used())
	}
}
