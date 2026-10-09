# C++/Go Conformance Tracing Ideas

## Extending the Ivy `xtracer` approach

The Ivy design is a strong starting point, but tracing all of Z3 as formatted text would become the bottleneck. We should preserve its best properties—streaming comparison, bounded history, and immediate divergence reporting—while changing the representation and trace granularity.

The central limitation is that compiler-generated trace points describe implementation structure. For a cross-language port, we need to compare semantics. C++ and Go may split functions differently, inline different helpers, and use different control flow while producing identical results. Automatic compiler instrumentation can help discover execution boundaries, but it cannot completely replace deliberate matching checkpoints.

The main options are:

| Approach | What it provides | Cross-language usefulness | Runtime cost |
|---|---|---:|---:|
| Manual C++ macros and Go calls | Stable semantic checkpoints and chosen values | Excellent | Controllable |
| Generated calls from annotations | Semantic checkpoints with less repetitive editing | Excellent | Controllable |
| `-finstrument-functions` | Every C++ function entry and exit | Limited | High |
| Clang XRay | Runtime-enabled function tracing and efficient circular buffering | Limited to good for selected functions | Low when disabled; moderate when active |
| SanitizerCoverage | Functions, basic blocks, edges, comparisons | Poor for direct Go comparison | Moderate to extreme |
| LLVM/GCC compiler plugin | Arbitrary automatic instrumentation | Potentially good, but considerable infrastructure | Depends on inserted calls |
| uprobes/eBPF/perf | External function-boundary observation | Mostly diagnostic | Low to moderate |

My preferred design would combine manual semantic checkpoints with annotation-driven generation.

A shared checkpoint definition could look conceptually like:

```text
ID 24017
name util.rational.normalize.exit
payload numerator denominator sign
```

The C++ might contain:

```cpp
G3_TRACE(24017, numerator, denominator, sign);
```

and the Go equivalent:

```go
xtracer.Event(24017, numerator, denominator, sign)
```

Eventually we could reduce the source noise with annotations:

```cpp
[[clang::annotate("g3trace:24017")]]
rational normalize(...);
```

```go
//g3trace:24017
func normalize(...) ...
```

A Clang LibTooling program and a small Go AST tool could then generate entry and exit instrumentation from those annotations. Clang plugins can inspect the AST as part of compilation, while LibTooling is intended for standalone source tools. [Clang documents both models here.](https://clang.llvm.org/docs/Tooling.html)

The explicit numeric ID matters. File names and line numbers are useful diagnostics, but they should not define event identity because editing either implementation would renumber the trace. The shared definition also gives us a place to document canonical payload encoding.

For speed, I would change the wire protocol from one formatted line per event to framed binary records over dedicated pipes:

```text
event ID | flags | payload length | canonical payload or digest
```

Each process would buffer perhaps 64 KiB before writing. The comparator would decode records, compare them, and retain the last 30 decoded events exactly as the Ivy test does. Pipes provide backpressure, bounded memory, and no trace files. A dedicated inherited file descriptor would prevent solver output from getting mixed into the trace stream; stdout could remain available as a debug mode.

The current Z3 `TRACE` facility is poorly suited to this. It writes human-oriented output to `.z3-trace` and flushes frequently. We could borrow its compile-time and tag-filtering ideas, but conformance tracing deserves a separate small runtime.

The largest speed gain would probably come from hierarchical comparison:

1. Emit sparse checkpoints around public operations and major algorithm phases.
2. At each checkpoint, compare a canonical state digest and a few useful counts.
3. If a checkpoint differs, rerun with tracing enabled only for that component or interval.
4. Within the failing interval, emit block digests every 1,024 or 4,096 logical events.
5. Rerun the first failing block with full event detail.

That gives us a trace equivalent of binary search. A successful run might emit only hundreds of records even if the solver performed millions of internal operations.

For very hot loops, the implementation could absorb events into an incremental digest and emit only:

```text
checkpoint ID | number of absorbed events | digest
```

When it differs, rerun with smaller blocks. This preserves exact detection without paying transport and comparison costs for every event on every run. This hashes canonical runtime behavior only; it does not hash C++ source files.

Canonicalization will be the hard part. We need rules for:

- Arbitrary precision integers, rationals, algebraic numbers, and bit vectors.
- Sets and maps whose native iteration order differs.
- AST nodes and other graph objects.
- Pointer-based C++ identities.
- Random choices and seeds.
- NaNs, signed zero, and floating-point payloads where applicable.

Pointers must never enter the comparison. Graph objects can receive stable trace IDs in logical creation order, or be represented by structural digests. Maps should be serialized in a specified semantic order unless insertion order is itself meaningful. For the Go side, `insdict` is available when insertion order is required.

Compiler instrumentation can still contribute:

- **GCC/Clang `-finstrument-functions`:** The compiler inserts calls to `__cyg_profile_func_enter` and `__cyg_profile_func_exit`, including conceptually inlined functions. Files and functions can be excluded, and the tracer itself must use `no_instrument_function`. This is easy to prototype, but function addresses require symbol mapping and do not correspond naturally to Go helpers. [GCC documents the hooks and filtering here.](https://gcc.gnu.org/onlinedocs/gcc/Instrumentation-Options.html)
- **Clang XRay:** XRay inserts patchable entry and exit sleds and can enable them at runtime. Its FDR mode already uses bounded, recycled memory buffers, close to the behavior we want. It also allows functions to be forced into or excluded from instrumentation. This looks useful for discovering a C++ call path and tracing a selected set of one-to-one ported functions. It still lacks semantic state unless we add custom events. [LLVM XRay documentation](https://www.llvm.org/docs/XRay.html)
- **SanitizerCoverage:** It can invoke callbacks at function entries, basic blocks, edges, comparisons, divisions, loads, and stores. It can also restrict instrumentation without changing source. This is valuable for coverage and for finding which C++ path an input exercised, but it is tied closely to the compiler’s lowered control-flow graph and therefore would be fragile as a C++/Go equivalence protocol. [Clang SanitizerCoverage documentation](https://clang.llvm.org/docs/SanitizerCoverage.html)
- **Custom Clang/LLVM instrumentation:** An AST tool can recognize selected functions or annotations and insert stable event IDs. An LLVM pass can instrument everything automatically, but optimization and lowering make the resulting points less stable and less intelligible. I would choose an AST/source tool over an IR pass for this project.
- **uprobes/eBPF:** Useful for a zero-source-change survey of which C++ functions execute, but extracting complex C++ values through the ABI is painful. It is better as an exploratory tool than as the conformance oracle.

I would start with a deliberately small prototype in `util`: choose one deterministic numeric or polynomial operation, place perhaps 10–30 semantic checkpoints in both implementations, and compare compact binary records through pipes. That experiment would tell us the actual costs of canonicalization, hashing, and transport before we build an instrumentation generator.

My likely long-term arrangement would be:

- Manually assigned, stable semantic event IDs.
- A shared manifest describing each event and payload.
- C++ macros and Go functions that compile completely away in ordinary builds.
- Buffered binary pipes for ordinary tracing.
- Text rendering only inside the comparator or on divergence.
- Hierarchical checkpoint and block digests.
- Clang annotation tooling to insert routine entry/exit events for selected mapped functions.
- XRay as an optional C++ path-discovery aid, rather than the primary comparison protocol.

This retains the confidence of the Ivy method while making successful runs cheap and concentrating detailed tracing only where the two implementations first disagree.

## Go runtime tracing

Assuming “Go event tracer” means the standard library’s `runtime/trace`: it records a binary timeline of what the Go runtime is doing. It is mainly a performance and scheduling diagnostic tool rather than a general-purpose semantic event stream.

It automatically records events such as:

- Goroutine creation, execution, blocking, and wakeup.
- Syscall entry and exit.
- Garbage collection activity.
- Heap-size changes.
- Processor and scheduler activity.
- Nanosecond timestamps and stack traces for many events.

Your code can add three kinds of annotations:

- `trace.Log`: a point event containing a category and message.
- `trace.StartRegion` or `trace.WithRegion`: a timed region executed by one goroutine.
- `trace.NewTask`: a logical operation that can extend across multiple goroutines through a `context.Context`.

A basic use looks like:

```go
package main

import (
	"context"
	"os"
	"runtime/trace"
)

func main() {
	out, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	if err := trace.Start(out); err != nil {
		panic(err)
	}
	defer trace.Stop()

	ctx, task := trace.NewTask(context.Background(), "solve")
	defer task.End()

	trace.Log(ctx, "checkpoint", "sat.begin")

	region := trace.StartRegion(ctx, "simplify")
	simplify()
	region.End()

	trace.Log(ctx, "checkpoint", "sat.end")
}
```

Tests have built-in support:

```bash
go test -trace=trace.out ./pkg/smtcore
go tool trace trace.out
```

The binary trace is then explored as a timeline. It is particularly good at answering questions such as “Why did this goroutine stop running?”, “Was the GC active here?”, and “Did this operation block in a syscall?” The [official package documentation](https://go.dev/pkg/runtime/trace/?m=old) describes the events and annotations.

### Flight recorder

There is also a newer flight recorder, available in the current toolchain:

```go
fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
	MinAge:   5 * time.Second,
	MaxBytes: 16 << 20,
})

if err := fr.Start(); err != nil {
	panic(err)
}
defer fr.Stop()

// Run normally...

// When something goes wrong:
_, err := fr.WriteTo(destination)
```

The flight recorder continually retains a moving window of recent execution trace data. `MaxBytes` is an approximate bound and takes priority over `MinAge`. Calling `WriteTo` takes a valid snapshot that normal trace tooling can read. Go added this specifically for capturing the period before an error is detected. The [Go flight-recorder introduction](https://go.dev/blog/flight-recorder) explains its bounded-buffer model.

### Suitability for conformance comparison

For the C++/Go comparison, I would not use `runtime/trace` as the primary conformance stream.

1. **The trace contains runtime noise.** GC timing, scheduler events, timestamps, goroutine IDs, and system calls will differ between runs. Two correct executions will not produce equal binary traces.

2. **Records are optimized for later reconstruction.** The runtime uses per-thread buffers and lets the analysis tool reconstruct chronological order. The raw byte stream is not an immediately comparable sequence like `XTRACE` lines.

3. **User logs accept strings.** Encoding canonical structured values would require formatting them into the message. Repeated unique messages enlarge the trace’s string data and add allocations or formatting work.

   ```go
   trace.Log(ctx, "g3", "event 24017")
   ```

4. **It captures much more than we need.** Modern runtime tracing has reportedly fallen to roughly 1–2% overhead for many ordinary workloads, but a high-frequency `trace.Log` at every solver transition could cost much more. [Go’s execution-trace overview](https://go.dev/blog/execution-traces-2024) discusses its buffering and overhead.

5. **It has no equivalent on the C++ side.** We would still need a separate semantic protocol and then somehow extract only the Go user events from the runtime trace.

It could be extremely useful as a companion to the conformance tracer:

```text
C++ semantic stream ─┐
                     ├── live comparator
Go semantic stream ──┘         │
                               │ mismatch
                               ▼
                    snapshot Go flight recorder
```

At each sparse semantic checkpoint, the Go implementation could add a cheap runtime annotation containing only the stable event ID:

```go
if trace.IsEnabled() {
	trace.Log(ctx, "g3.checkpoint", "24017")
}
```

The actual canonical values and hashes would go through our lean binary comparison stream. If comparison finds a divergence, the parent harness could send the Go child a command over a control pipe. The child would call `FlightRecorder.WriteTo` and send the snapshot back through another pipe. That gives us:

- Exact semantic comparison through the small custom tracer.
- The last few seconds of Go scheduling, GC, allocation, and syscall context around a failure.
- Correlation through the shared checkpoint number.
- No routine trace files or unlimited disk growth.

We should guard message construction with `trace.IsEnabled()`:

```go
if trace.IsEnabled() {
	trace.Log(ctx, "g3.checkpoint", checkpointName)
}
```

For normal production builds, our semantic trace calls can still compile away entirely through build tags. The runtime annotations could live inside that same tracing implementation.

### Local toolchain note

The installed `go1.26.4-pont-onethread` toolchain has the complete `runtime/trace` and `FlightRecorder` APIs, but `go tool trace` currently reports `no such tool "trace"`. Trace collection will work, but interactive viewing would require installing or building the trace command, or opening the output with another compatible Go installation.

The Go event tracer is therefore worth using as the black box surrounding a divergence. The custom stream remains the right tool for determining that the C++ and Go executions first disagreed at a particular semantic checkpoint.
