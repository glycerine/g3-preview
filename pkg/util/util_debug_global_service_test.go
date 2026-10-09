// Source: src/util/debug.cpp:28-68,73-117; src/util/debug.h:42-65;
// src/test/mem_initializer.cpp:13-20; src/util/gparams.cpp:609.
package util

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
)

func debugGlobalBytes(s *DebugGlobalService) uint64 {
	n := s.header + s.context.ownerBytes + s.streams.header + s.input.header + s.actions.header
	n += s.streams.stdout.Stats().OwnerBytes + s.streams.StderrField.Stats().OwnerBytes
	n += debugTagColdBytes(s.context)
	return n
}
func debugGlobalRegionBytes(r *arena.BytesRegion) uint64 {
	s := r.Stats()
	d := s.Descriptors
	return s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}
func TestDebugGlobalServiceDefaultOwnersAndSourceFinalize(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	region := traceRegion(t, b)
	s, e := NewDebugGlobalService(DebugGlobalServiceConfig{Budget: b, Region: region})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, s.Close())
		paramsMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	d, e := s.Context()
	paramsMust(t, e)
	if d != s.context || d.assertionStreams != s.streams || d.assertionActions != s.actions || s.streams.stdout.File != os.Stdout || s.streams.StderrField.File != os.Stderr || s.input.file != os.Stdin || s.input.tie != s.streams.stdout {
		t.Fatal("actual default provider identity")
	}
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	if b.Used() != debugGlobalBytes(s)+debugGlobalRegionBytes(region) {
		t.Fatal("named cold graph", b.Used())
	}
	b.FailAfter(0)
	paramsMust(t, s.FinalizeDebug())
	on, e := d.AssertionsEnabled()
	paramsMust(t, e)
	if on || d.Tags != nil || s.actions.closed || s.input.closed || s.streams.closed {
		t.Fatal("source finalize destroyed static helpers or reset switch")
	}
	if b.Used() != debugGlobalBytes(s)+debugGlobalRegionBytes(region) {
		t.Fatal("finalized named graph", b.Used())
	}
	b.FailAfter(-1)
}
func TestDebugGlobalServiceEveryConstructorRefusalKeepsBorrowedRegion(t *testing.T) {
	first := -1
	for ordinal := 0; ordinal < 24; ordinal++ {
		b := arena.NewBudget(1 << 26)
		region := traceRegion(t, b)
		key := traceKey(t, region, "caller-owned")
		foreign, e := NewDebugContext(region, b)
		paramsMust(t, e)
		b.FailAfter(int64(ordinal))
		s, e := NewDebugGlobalService(DebugGlobalServiceConfig{Budget: b, Region: region})
		want := debugGlobalRegionBytes(region) + foreign.ownerBytes
		if s != nil {
			want += debugGlobalBytes(s)
		}
		if b.Used() != want {
			t.Fatal("actual acquired graph before fault lift", ordinal, b.Used(), want)
		}
		if foreign.closed {
			t.Fatal("adopted unrelated context")
		}
		if _, re := region.Read(key.Allocation(), key.Offset()); re != nil {
			t.Fatal("caller bytes", re)
		}
		b.FailAfter(-1)
		if s != nil {
			paramsMust(t, s.Close())
		}
		paramsMust(t, foreign.Close())
		paramsMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if e == nil {
			first = ordinal
			break
		}
	}
	if first < 0 {
		t.Fatal("did not reach first constructor success")
	}
	t.Log("actual first constructor success", first)
}
func TestDebugGlobalServiceOwnerCopyAndDefaultBorrowLifetime(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	region := traceRegion(t, b)
	s, e := NewDebugGlobalService(DebugGlobalServiceConfig{Budget: b, Region: region})
	paramsMust(t, e)
	duplicate := reflect.New(reflect.TypeOf(*s))
	duplicate.Elem().Set(reflect.ValueOf(s).Elem())
	if _, e := duplicate.Interface().(*DebugGlobalService).Context(); !errors.Is(e, ErrDebugGlobalOwner) {
		t.Fatal(e)
	}
	paramsMust(t, s.Close())
	paramsMust(t, s.Close())
	if _, e := s.Context(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	// The borrowed caller region and process descriptors survive owning Close.
	ref, e := region.Allocate(1)
	paramsMust(t, e)
	paramsMust(t, region.Write(ref, 0, 9))
	if os.Stdout == nil || os.Stderr == nil || os.Stdin == nil {
		t.Fatal("closed borrowed defaults")
	}
	paramsMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestDebugGlobalServiceProcessSubprocessHelper(t *testing.T) {
	route := os.Getenv("G3_DEBUG_GLOBAL_CHILD")
	if route == "" {
		return
	}
	q, b, r := rationalGlobalFixture(t)
	region := traceRegion(t, b)
	profile := DebugInteractive
	if route == "no-debugger" {
		profile = DebugNoDebugger
	}
	s, e := NewDebugGlobalService(DebugGlobalServiceConfig{Budget: b, Region: region, Profile: profile})
	paramsMust(t, e)
	d, e := s.Context()
	paramsMust(t, e)
	symbols, e := NewSymbolService(SymbolConfig{Budget: b})
	paramsMust(t, e)
	trace, e := NewTraceContext(region, b)
	paramsMust(t, e)
	primes, e := NewDefaultPrimeService(b)
	paramsMust(t, e)
	p, e := NewMemoryProcess(MemoryProcessConfig{Runtime: r, Rationals: q, Symbols: symbols, Debug: d, Trace: trace, Primes: primes, Budget: b, Params: ParamsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}, Catalog: catalogTestConfig(b), Global: gparamsTestConfig(b)})
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, p.Close())
		paramsMust(t, primes.Close())
		paramsMust(t, trace.Close())
		paramsMust(t, symbols.Close())
		paramsMust(t, s.Close())
		paramsMust(t, region.Close())
	})
	paramsMust(t, p.Initialize(0))
	if p.parameters.namespaceAssertions != d {
		t.Fatal("generated child did not borrow exact service Context")
	}
	numeric := route == "numeric-continue" || route == "numeric-throw"
	var expectedEmpty error
	if !numeric {
		paramsMust(t, p.parameters.FinalizeSource())
	}
	debugGlobalProcessAudit(t, p, s)
	b.FailAfter(0)
	if numeric {
		g, e := p.parameters.Manager()
		paramsMust(t, e)
		_, expectedEmpty = g.GetValueName("")
	}
	if route == "bad-stderr" {
		paramsMust(t, os.Stderr.Close())
	}
	// The actual namespace consumer is reached through the process's sole
	// Context. Null continuation is guarded; bare_str C uses the empty-name body.
	if numeric {
		_, e = p.parameters.SourceGetValueSymbol(NumericalSymbol(1))
	} else {
		_, e = p.parameters.SourceGetRef()
	}
	throwing := route == "throw" || route == "numeric-throw"
	if GlobalParamsSourceDebug && throwing {
		var ex DefaultException
		if !errors.As(e, &ex) {
			t.Fatal(e)
		}
	} else if numeric {
		if (e == nil) != (expectedEmpty == nil) || e != nil && e.Error() != expectedEmpty.Error() {
			t.Fatal("actual empty-name engine", e, expectedEmpty)
		}
	} else if !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("continued null source access", e)
	}
	state, e := s.input.State()
	paramsMust(t, e)
	if GlobalParamsSourceDebug && state.Position != 1 {
		t.Fatal(state)
	}
	debugGlobalProcessAudit(t, p, s)
	b.FailAfter(-1)
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	paramsMust(t, p.Finalize())
	enabled, e := d.AssertionsEnabled()
	paramsMust(t, e)
	if enabled || d.Tags != nil || d.closed || s.actions.closed || s.input.closed || s.streams.closed {
		t.Fatal("generated finalizer changed helper lifetime")
	}
	debugGlobalProcessAudit(t, p, s)
	if route == "borrowed-files-live" {
		paramsMust(t, p.Close())
		paramsMust(t, s.Close())
		if _, e := os.Stdout.WriteString("borrowed stdout live\n"); e != nil {
			t.Fatal(e)
		}
		if _, e := os.Stderr.WriteString("borrowed stderr live\n"); e != nil {
			t.Fatal(e)
		}
	}
}
func TestDebugGlobalServiceActualProcessDefaultFalseConsumerActions(t *testing.T) {
	for _, route := range []string{"continue", "throw", "eof", "no-debugger", "numeric-continue", "numeric-throw", "bad-stderr", "borrowed-files-live"} {
		t.Run(route, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDebugGlobalServiceProcessSubprocessHelper$")
			cmd.Env = append(os.Environ(), "G3_DEBUG_GLOBAL_CHILD="+route)
			text := "C"
			if route == "throw" || route == "numeric-throw" {
				text = "T"
			}
			if route == "eof" {
				text = ""
			}
			cmd.Stdin = strings.NewReader(text)
			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			e := cmd.Run()
			if ctx.Err() != nil {
				t.Fatal("bounded child", ctx.Err())
			}
			want := 0
			if GlobalParamsSourceDebug && (route == "eof" || route == "no-debugger") {
				want = 110
			}
			actual := 0
			if e != nil {
				var exit *exec.ExitError
				if !errors.As(e, &exit) {
					t.Fatal(e)
				}
				actual = exit.ExitCode()
			}
			if actual != want {
				t.Fatal("actual default process exit", actual, want, stdout.String(), stderr.String())
			}
			if GlobalParamsSourceDebug {
				prefix := "ASSERTION VIOLATION\nFile: src/util/gparams.cpp\nLine: 609\ng_imp != 0\n"
				if strings.HasPrefix(route, "numeric-") {
					prefix = "ASSERTION VIOLATION\nFile: src/util/symbol.h\nLine: 96\n!is_numerical()\n"
				}
				if route == "bad-stderr" {
					if stderr.Len() != 0 {
						t.Fatal(stderr.String())
					}
					return
				}
				if !strings.HasPrefix(stderr.String(), prefix) {
					t.Fatal(stderr.String())
				}
				if route == "no-debugger" {
					if stderr.String() != prefix {
						t.Fatal(stderr.String())
					}
				} else {
					want := prefix + DebugSourcePrompt
					if route == "borrowed-files-live" {
						want += "borrowed stderr live\n"
					}
					if stderr.String() != want {
						t.Fatal(stderr.String())
					}
				}
			} else {
				want := ""
				if route == "borrowed-files-live" {
					want = "borrowed stderr live\n"
				}
				if stderr.String() != want {
					t.Fatal("release provider erased", stderr.String())
				}
			}
			if route == "borrowed-files-live" && !strings.Contains(stdout.String(), "borrowed stdout live\n") {
				t.Fatal(stdout.String())
			}
		})
	}
}

func debugGlobalProcessAudit(t *testing.T, p *MemoryProcess, service *DebugGlobalService) {
	t.Helper()
	c := p.config
	n := p.header + memoryProcessNumericalBytes(t, c.Rationals, c.Runtime)
	n += c.Symbols.ownerBytes
	if s := c.Symbols.manager; s != nil {
		v := s.Stats()
		d := v.Region.Descriptors
		n += v.OwnerBytes + v.Table.OwnerBytes + v.Table.TableBytes + v.Region.OwnerBytes + v.Region.PayloadBytes + v.Region.PageHeaderBytes + v.Region.DirectoryBytes + v.Region.LogBytes + v.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	n += c.Debug.ownerBytes
	n += debugTagColdBytes(c.Debug)
	n += c.Trace.ownerBytes
	regions := map[uintptr]bool{}
	for _, region := range []*arena.BytesRegion{c.Debug.region, c.Trace.region} {
		id := reflect.ValueOf(region).Pointer()
		if regions[id] {
			continue
		}
		regions[id] = true
		s := region.Stats()
		d := s.Descriptors
		n += s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	n += c.Primes.Stats().TotalBytes
	if v := p.context; v != nil {
		s := v.Stats()
		n += s.HeaderBytes + s.BlockBytes + s.ReceiptBytes + s.SourceFrameBytes + s.SourceSetBytes + s.ReplacementBytes + s.PageBytes + s.DirectoryBytes + s.FormatBytes + gparamsVectorGraph(s.Text)
	}
	if p.catalog != nil {
		n += catalogGraph(p.catalog)
	}
	if v := p.parameters; v != nil {
		n += v.OwnerBytes
		seen := map[*GlobalParams]bool{}
		add := func(g *GlobalParams) {
			if seen[g] {
				t.Fatal("duplicate gparams child owner")
			}
			seen[g] = true
			n += gparamsGraph(g)
			if d := g.constructorPendingDescriptor; d != nil {
				n += d.OwnerBytes + ParamDescriptorEntryBytes(len(d.Entries)) + ParamDescriptorNameBytes(cap(d.Names))
				if d.Texts != nil {
					n += gparamsVectorGraph(d.Texts.Stats())
				}
			}
		}
		if v.Current != nil {
			add(v.Current)
		}
		for g := v.Superseded; g != nil; g = g.RetainedNext {
			add(g)
		}
		for g := v.Failed; g != nil; g = g.RetainedNext {
			add(g)
		}
	}
	n += service.header + service.streams.header + service.input.header + service.actions.header + service.streams.stdout.Stats().OwnerBytes + service.streams.StderrField.Stats().OwnerBytes
	if n != c.Budget.Used() {
		t.Fatal("once-only explicit process/borrowed/retained cold graph", n, c.Budget.Used())
	}
}
