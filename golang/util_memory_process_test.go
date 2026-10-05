// Source: src/util/memory_manager.cpp:89-113,153-160;
// scripts/mk_genfile_common.py:742-802; src/test/main.cpp:140.
// Source author: Nikolaj Bjorner (2007-07-24).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
	"testing"
)

func memoryProcessFixture(t *testing.T, profile ...uint8) (*MemoryProcess, *arena.Budget) {
	t.Helper()
	var q *RationalGlobalService
	var b *arena.Budget
	var r *memory.Runtime
	if len(profile) == 0 {
		q, b, r = rationalGlobalFixture(t)
	} else {
		b = arena.NewBudget(1 << 26)
		var e error
		r, e = memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: profile[0]})
		paramsMust(t, e)
		q, e = NewRationalGlobalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: profile[0], Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
		paramsMust(t, e)
		t.Cleanup(func() {
			b.FailAfter(-1)
			rationalGlobalFinish(t, q)
			paramsMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("actual profile roots", b.Used())
			}
		})
	}
	symbols, e := NewSymbolService(SymbolConfig{Budget: b})
	paramsMust(t, e)
	region := traceRegion(t, b)
	debug, e := NewDebugContext(region, b)
	paramsMust(t, e)
	trace, e := NewTraceContext(region, b)
	paramsMust(t, e)
	primes, e := NewDefaultPrimeService(b)
	paramsMust(t, e)
	cfg := MemoryProcessConfig{Runtime: r, Rationals: q, Symbols: symbols, Debug: debug, Trace: trace, Primes: primes, Budget: b, Params: ParamsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}, Catalog: catalogTestConfig(b), Global: gparamsTestConfig(b)}
	p, e := NewMemoryProcess(cfg)
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, p.Close())
		paramsMust(t, primes.Close())
		paramsMust(t, trace.Close())
		paramsMust(t, debug.Close())
		paramsMust(t, region.Close())
		paramsMust(t, symbols.Close())
	})
	return p, b
}

func TestMemoryProcessFirstInitFlagsAndGeneratedChildren(t *testing.T) {
	p, b := memoryProcessFixture(t)
	if p.initialized || p.fullyInitialized || p.finalizing || p.parameters != nil {
		t.Fatal("cold owner falsely completed source initialization")
	}
	paramsMust(t, p.Initialize(0))
	if !p.initialized || !p.fullyInitialized || p.finalizing || p.config.Rationals.current == nil || p.config.Symbols.manager == nil || p.parameters == nil || p.parameters.current == nil {
		t.Fatal("generated actual children not published")
	}
	memoryProcessColdAudit(t, p)
	before := p.config.Runtime.State()
	used := b.Used()
	actual := p.parameters.current
	b.FailAfter(0)
	paramsMust(t, p.Initialize(math.MaxUint32))
	if p.config.Runtime.State() != before || b.Used() != used || p.parameters.current != actual {
		t.Fatal("fully initialized repeat entered child constructors")
	}
}

func TestMemoryProcessFailedFirstWaitPreservesReachedOwnerPrefix(t *testing.T) {
	p, b := memoryProcessFixture(t)
	b.FailAfter(0)
	e := p.Initialize(0)
	if !errors.Is(e, arena.ErrInjectedFailure) || !p.initialized || p.fullyInitialized || p.config.Rationals.current != nil || p.config.Symbols.manager != nil || p.parameters != nil {
		t.Fatal("first failure flag/pointer prefix", e)
	}
	memoryProcessColdAudit(t, p)
	before := p.config.Runtime.State()
	used := b.Used()
	b.FailAfter(-1)
	if e = p.Initialize(math.MaxUint32); !errors.Is(e, ErrMemoryProcessInitializationWait) {
		t.Fatal("failed-first literal wait must not retry children", e)
	}
	if p.config.Runtime.State() != before || b.Used() != used {
		t.Fatal("wait adapter repaired source state")
	}
}

func TestMemoryProcessFinalizeRetainsFullyFlagAndBorrowedOwnerIdentities(t *testing.T) {
	p, _ := memoryProcessFixture(t)
	paramsMust(t, p.Initialize(0))
	q, s, d, tr, prime := p.config.Rationals, p.config.Symbols, p.config.Debug, p.config.Trace, p.config.Primes
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, tr.EnableAllTrace(true))
	paramsMust(t, p.Finalize())
	if p.initialized || !p.fullyInitialized || p.finalizing || q.current != nil || s.manager != nil || prime.generator.Stats().Count != 0 || d.tags != nil || tr.tags != nil || !tr.all {
		t.Fatal("literal heterogeneous finalizer state")
	}
	paramsMust(t, p.Close())
	if q.closed || s.closed || d.closed || tr.closed || prime.closed {
		t.Fatal("parent owning close adopted borrowed service headers")
	}
	enabled, e := d.AssertionsEnabled()
	paramsMust(t, e)
	if enabled {
		t.Fatal("debug finalize reset assertion switch")
	}
}

func TestMemoryProcessEveryActualInitializationReservationAndOwnedAbort(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		p, b := memoryProcessFixture(t)
		tag := traceKey(t, p.config.Trace.region, "caller-kept-tag")
		paramsMust(t, p.config.Trace.EnableTrace(tag))
		primeBefore := p.config.Primes.Stats()
		b.FailAfter(ordinal)
		e := p.Initialize(0)
		memoryProcessColdAudit(t, p)
		if !p.initialized {
			t.Fatal("source flag must precede actual constructor refusal")
		}
		if e == nil {
			if !p.fullyInitialized || p.initializationStage != 3 {
				t.Fatal("early full publication")
			}
			t.Log("first complete generated initialization ordinal", ordinal)
			return
		}
		var f memory.Failure
		if !errors.Is(e, arena.ErrInjectedFailure) && (!errors.As(e, &f) || f.Kind != memory.FailureMemout || f.ErrorCode() != 101 || !p.config.Runtime.State().OutOfMemory) {
			t.Fatal("actual refusal category", ordinal, e)
		}
		if p.fullyInitialized {
			t.Fatal("failed child published fully-initialized")
		}
		before := p.config.Runtime.State()
		used := b.Used()
		b.FailAfter(-1)
		if e = p.Initialize(math.MaxUint32); !errors.Is(e, ErrMemoryProcessInitializationWait) {
			t.Fatal("wait retried failed source initialization", ordinal, e)
		}
		if p.config.Runtime.State() != before || b.Used() != used {
			t.Fatal("wait mutated actual prefix")
		}
		q, s := p.config.Rationals.current, p.config.Symbols.manager
		paramsMust(t, p.Close())
		memoryProcessColdAudit(t, p)
		if p.config.Rationals.current != q || p.config.Symbols.manager != s || p.config.Primes.Stats() != primeBefore {
			t.Fatal("owning parent abort mutated borrowed providers")
		}
		enabled, e := p.config.Trace.IsTraceEnabled(tag)
		paramsMust(t, e)
		if !enabled {
			t.Fatal("unrelated actual caller trace root destroyed")
		}
	}
	t.Fatal("actual initialization horizon did not reach success")
}

func TestMemoryProcessFirstCallClearsRealOOMBeforeFailedChild(t *testing.T) {
	p, b := memoryProcessFixture(t)
	b.FailAfter(0)
	block, e := p.config.Runtime.Allocate(16)
	if block != (memory.Block{}) || e == nil || !p.config.Runtime.State().OutOfMemory {
		t.Fatal("real prior physical refusal did not publish OOM", block, e)
	}
	e = p.Initialize(0)
	if !errors.Is(e, arena.ErrInjectedFailure) || p.config.Runtime.State().OutOfMemory {
		t.Fatal("first branch did not clear actual OOM before constructor", e)
	}
	memoryProcessColdAudit(t, p)
}

func TestMemoryProcessReinitRetainsFullyTrueAfterEndedStaticPublicationGuard(t *testing.T) {
	p, b := memoryProcessFixture(t)
	paramsMust(t, p.Initialize(0))
	paramsMust(t, p.Finalize())
	e := p.Initialize(0)
	if !errors.Is(e, mpnum.ErrRationalStaticLifetime) || !p.initialized || !p.fullyInitialized || p.finalizing || p.config.Rationals.current == nil {
		t.Fatal("actual reinit flag/manager publication prefix", e)
	}
	published := p.config.Rationals.current
	used := b.Used()
	state := p.config.Runtime.State()
	b.FailAfter(0)
	paramsMust(t, p.Initialize(7))
	if p.config.Runtime.State() != state || b.Used() != used || p.config.Rationals.current != published {
		t.Fatal("fully-true repeat retried failed child")
	}
	// The setter executes before the flag shortcut. Prove the reached limit with
	// a real allocation rather than reading a private accounting field.
	b.FailAfter(-1)
	// The TLS profile checks the limit only at the actual 100000-byte sync.
	block, limitErr := p.config.Runtime.Allocate(100008)
	if block == (memory.Block{}) || limitErr == nil || !p.config.Runtime.State().OutOfMemory {
		if block != (memory.Block{}) {
			paramsMust(t, p.config.Runtime.Deallocate(&block))
		}
		t.Fatal("non-sentinel repeat did not update actual max size", block, limitErr)
	}
	paramsMust(t, p.config.Runtime.Deallocate(&block))
	paramsMust(t, p.config.Runtime.SetMaxSize(0))
	e = p.Finalize()
	if !errors.Is(e, mpnum.ErrRationalStaticLifetime) || !p.initialized || !p.fullyInitialized || !p.finalizing || p.finalizationStage != 3 || p.sourceCause != e || p.config.Rationals.current != published {
		t.Fatal("repeated ended Inf finalizer prefix", e)
	}
	if again := p.Finalize(); again != e {
		t.Fatal("stopped source cause changed", again)
	}
	memoryProcessColdAudit(t, p)
}

func TestMemoryProcessSizeTConversionBeforeSentinelAndWait(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		t.Run(fmt.Sprint(bits), func(t *testing.T) {
			p, b := memoryProcessFixture(t, bits)
			paramsMust(t, p.config.Runtime.SetMaxSize(5))
			e := p.Initialize((uint64(1) << 32) + math.MaxUint32)
			if bits == 32 {
				var f memory.Failure
				if !errors.As(e, &f) || f.Kind != memory.FailureMemout || f.ErrorCode() != 101 || !p.initialized || p.fullyInitialized || p.config.Rationals.current != nil {
					t.Fatal("size_t32 sentinel failed to preserve reached old limit", e)
				}
				b.FailAfter(0)
				if e = p.Initialize(uint64(1) << 32); !errors.Is(e, ErrMemoryProcessInitializationWait) {
					t.Fatal("source zero-converted max must precede failed-first wait", e)
				}
				b.FailAfter(-1)
				block, e := p.config.Runtime.Allocate(8)
				paramsMust(t, e)
				paramsMust(t, p.config.Runtime.Deallocate(&block))
			} else {
				paramsMust(t, e)
				if !p.fullyInitialized || p.config.Rationals.current == nil {
					t.Fatal("LP64 non-sentinel wide max did not permit real constructor")
				}
			}
			memoryProcessColdAudit(t, p)
		})
	}
}

func TestMemoryProcessRejectsEachForeignColdChildBudgetBeforeReservation(t *testing.T) {
	for _, kind := range []string{"symbol", "debug", "trace", "prime"} {
		t.Run(kind, func(t *testing.T) {
			p, b := memoryProcessFixture(t)
			foreign := arena.NewBudget(1 << 20)
			cfg := p.config
			var closeChild func() error
			switch kind {
			case "symbol":
				v, e := NewSymbolService(SymbolConfig{Budget: foreign})
				paramsMust(t, e)
				cfg.Symbols = v
				closeChild = v.Close
			case "debug":
				v, e := NewDebugContext(p.config.Debug.region, foreign)
				paramsMust(t, e)
				cfg.Debug = v
				closeChild = v.Close
			case "trace":
				v, e := NewTraceContext(p.config.Trace.region, foreign)
				paramsMust(t, e)
				cfg.Trace = v
				closeChild = v.Close
			case "prime":
				v, e := NewDefaultPrimeService(foreign)
				paramsMust(t, e)
				cfg.Primes = v
				closeChild = v.Close
			}
			defer func() {
				paramsMust(t, closeChild())
				if foreign.Used() != 0 {
					t.Fatal("foreign borrowed child header leaked")
				}
			}()
			before, other, state := b.Used(), foreign.Used(), p.config.Runtime.State()
			b.FailAfter(0)
			candidate, e := NewMemoryProcess(cfg)
			if candidate != nil || !errors.Is(e, arena.ErrConfig) {
				t.Fatal("foreign child reached parent reservation", candidate, e)
			}
			if b.Used() != before || foreign.Used() != other || p.config.Runtime.State() != state || p.config.Rationals.current != nil || p.config.Symbols.manager != nil {
				t.Fatal("foreign cohort validation changed source/cold state")
			}
		})
	}
}

// The actual imp global ParamsRef supplies a retained NUM leaf. Public source
// TLS accounting reaches its signed-overflow guard after physical release;
// no free callback, fake OOM, or source-field corruption creates the stop.
func TestMemoryProcessActualGParamsStopPreservesLaterBorrowedChildren(t *testing.T) {
	p, b := memoryProcessFixture(t)
	paramsMust(t, p.Initialize(0))
	q := p.config.Rationals.service
	m, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("pure generated finalizer checkpoint") }, Cooperation: func() error { panic("pure generated finalizer cooperation") }})
	paramsMust(t, e)
	t.Cleanup(func() { paramsDecFinish(t, m.Close) })
	imp := p.parameters.current
	key, e := p.config.Symbols.manager.Intern("source-stored-num")
	paramsMust(t, e)
	prefix, e := p.config.Symbols.manager.Intern("source-inline-prefix")
	paramsMust(t, e)
	paramsMust(t, p.context.SetUInt(&imp.globalParams, prefix, 42))
	scoped, e := mpnum.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	value, e := scoped.Get()
	paramsMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	paramsMust(t, m.IntegerManager().SetLimbs(value, limbs))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	paramsMust(t, p.context.SetRat(&imp.globalParams, key, &big))
	paramsMust(t, q.Del(&big))
	// An actual unrelated caller numeral survives parent owning abort.
	caller, e := q.NewValue()
	paramsMust(t, e)
	paramsMust(t, q.SetString(&caller, "18446744073709551617"))
	t.Cleanup(func() { paramsDecFinish(t, func() error { return q.Del(&caller) }) })
	tag := traceKey(t, p.config.Debug.region, "later-borrowed-tag")
	paramsMust(t, p.config.Debug.EnableDebug(tag))
	paramsMust(t, p.config.Trace.EnableTrace(tag))
	extras := upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{p.context, &imp.globalParams}}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &caller}}}
	before := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, p.config.Runtime)
	cause := p.Finalize()
	if !errors.Is(cause, arena.ErrBounds) || !p.initialized || !p.fullyInitialized || !p.finalizing || p.finalizationStage != 1 || p.sourceCause != cause || imp.sourceStage != 0 || p.parameters.current != imp {
		t.Fatal("reached parent/imp source stop", cause)
	}
	after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	if before.Named != after.Named || before.Objects != after.Objects+1 {
		t.Fatal("retained source NUM versus consumed physical block", before, after)
	}
	memoryProcessColdAudit(t, p, m)
	if p.config.Debug.tags != nil || p.config.Trace.tags == nil || p.config.Primes.generator.Stats().Count == 0 || p.config.Symbols.manager == nil || p.config.Rationals.current == nil {
		t.Fatal("earlier debug and unread later finalizer prefix")
	}
	got, e := p.context.GetUInt(&imp.globalParams, prefix, 0)
	paramsMust(t, e)
	if got != 42 {
		t.Fatal("live source Reset inline prefix lost")
	}
	b.FailAfter(0)
	if again := p.Finalize(); again != cause || p.finalizationStage != 1 {
		t.Fatal("source-stopped replay changed cause/cursor", again)
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	memoryProcessColdAudit(t, p, m)
	b.FailAfter(-1)
	paramsMust(t, p.Close())
	extras.ParamsValues = nil
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, extras)
	memoryProcessColdAudit(t, p, m)
	text, e := q.String(&caller)
	paramsMust(t, e)
	if text != "18446744073709551617" {
		t.Fatal("parent abort destroyed unrelated caller numeral", text)
	}
	if p.sourceCause != cause || !p.initialized || !p.fullyInitialized || !p.finalizing || p.config.Trace.tags == nil || p.config.Primes.generator.Stats().Count == 0 {
		t.Fatal("owning abort repaired source flags or adopted borrowed children")
	}
}

func TestMemoryProcessFailedFirstFinalizeUsesActualNullGParamsBranch(t *testing.T) {
	p, b := memoryProcessFixture(t)
	b.FailAfter(0)
	if e := p.Initialize(0); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	e := p.Finalize()
	if !errors.Is(e, mpnum.ErrRationalServiceNoManager) || p.parameters != nil || p.finalizationStage != 3 || !p.initialized || p.fullyInitialized || !p.finalizing || p.config.Primes.generator.Stats().Count != 0 {
		t.Fatal("actual null g_imp branch or first null rational DT prefix", e)
	}
	mask, e := p.config.Rationals.service.SourceEndedStaticMask()
	paramsMust(t, e)
	if mask != 1 {
		t.Fatal("first static DT entry not stamped", mask)
	}
	memoryProcessColdAudit(t, p)
	paramsMust(t, p.Close())
	if p.config.Rationals.closed || p.config.Symbols.closed || p.config.Debug.closed || p.config.Trace.closed || p.config.Primes.closed || p.sourceCause == nil {
		t.Fatal("parent abort adopted borrowed children or repaired source failure")
	}
}

func TestMemoryProcessIdentityAndColdNoActionFinalizer(t *testing.T) {
	p, b := memoryProcessFixture(t)
	before := b.Used()
	state := p.config.Runtime.State()
	prime := p.config.Primes.Stats()
	paramsMust(t, p.Finalize())
	if b.Used() != before || p.config.Runtime.State() != state || p.config.Primes.Stats() != prime || p.finalizationStage != 0 {
		t.Fatal("uninitialized source finalizer touched unused children")
	}
	copyValue := reflect.New(reflect.TypeOf(*p))
	copyValue.Elem().Set(reflect.ValueOf(p).Elem())
	other := copyValue.Interface().(*MemoryProcess)
	if !errors.Is(other.Initialize(0), ErrMemoryProcessOwner) || !errors.Is(other.Finalize(), ErrMemoryProcessOwner) || !errors.Is(other.Close(), ErrMemoryProcessOwner) {
		t.Fatal("shallow parent copy accepted")
	}
	paramsMust(t, p.Close())
	if !errors.Is(p.Initialize(0), arena.ErrClosed) || !errors.Is(p.Finalize(), arena.ErrClosed) {
		t.Fatal("closed parent ordinary entry accepted")
	}
	paramsMust(t, p.Close())
	memoryProcessColdAudit(t, p)
}
