# Concrete string_buffer prerequisites

Owned files: util_string_buffer.go/test and this status only. No other project
files, manifests, progress ledgers, or git metadata were edited. All scratch,
compiler temp/build cache/output remains below golang/.cache. No Go generics,
unsafe storage, cgo, boxed hot records, or per-record callbacks were introduced.

## Scope and source assertions

Complete source methods and insertion routes are implemented for the actual
fork INITIAL_SIZE specializations **32,64(default),128**. string_buffer.h remains
**partial as a template source unit for other initial sizes**, not falsely a
port of arbitrary template instantiations. All instantiated sizes found under
src are covered. Symbol buffer insertion for these concrete types is covered;
the rest of the symbol/AST/parser consumer algorithms are separate existing or
future units. No invented deep-copy constructor/assignment is exported for the
source's unsafe implicit ownership copy.

src/test/main.cpp:160 registers TST(string_buffer), invoking
src/test/string_buffer.cpp:48-52 tst1/2/3. Three active ENSURE assertions are
ported first: Testing10true, length10000 after random digits, Hello World across
32←128. The nondeterministic rand sequence is replaced by a finite LCG producing
digits0..9; the actual source assertion is length, not a specific random string.
The tst2 TRACE output is diagnostic and is not counted as an extra exact printed
assertion. Additional Go boundary/model tests do not inflate source counts.

## Complete selected method mapping

| Source | Concrete API/helper |
| --- | --- |
| string_buffer.h:31-34 initial array/control fields | physical [32]/[64]/[128]byte arrays in concrete owner, shared private scalar/control state |
| string_buffer.h:36-45 expand | stringBufferState.expand, exact doubled capacity, copy old m_pos bytes, release old overflow after copy |
| string_buffer.h:48-52 constructor | NewStringBuffer32/64/128 and default NewStringBuffer alias64 |
| string_buffer.h:54-58 destructor | Close, owner+overflow releases once |
| string_buffer.h:60-62 reset | Reset changes pos only; retains capacity/content |
| string_buffer.h:64-70 char append | AppendByte, grows iff pos>=capacity then publishes one byte |
| string_buffer.h:72-80 CString append | AppendCString, first-NUL length, repeated grow before bulk copy/pos increment |
| string_buffer.h:82-90 int append | AppendInt32, stack24-byte signed conversion then bulk append |
| string_buffer.h:92-100 unsigned append | AppendUInt32, explicit signed32-bit baseline adapter below |
| string_buffer.h:102-110 long append | AppendLong, LP64 default or32-bit source range guard |
| string_buffer.h:112-119 bool append | AppendBool, exact true/false strings |
| string_buffer.h:121-127 size/empty | Size unsigned32 cast, Empty; Position uint64 is a Go inspection adapter |
| string_buffer.h:129-135 mutating const c_str | CStr, exact boundary growth, NUL write without pos increment, first-NUL caller-owned copy |
| string_buffer.h:139-173 six insertion operators | corresponding named append methods for CString/byte/int/uint/bool/long, no Go operator overload |
| string_buffer.h:175-179 buffer insertion | all9 AppendBuffer32/64/128 destination/source combinations, source CStr first |
| symbol.h:113-129 buffer symbol insertion | AppendSymbol, plain/null bulk CString or k! prefix then signed UNBOXINT |
| source has no Read/Stats/public close guard | checked Read scalar byte, Stats, shallow-copy/closed typed guard adapters |

## Actual fork consumers audited

- src/util/symbol.cpp:97:128-byte numeric k! buffer; symbol.h buffer friend for all sizes.
- src/ast/ast.cpp:1239,2223:64-byte model/fresh declarations;2240,2247:32-byte fresh sort/variable names.
- src/ast/ast_smt2_pp.cpp:181,240,497,585:default64 numeral/floating labels, name owner, and variable fallbacks (these printer algorithms are not ported by this packet).
- src/ast/fpa/fpa2bv_rewriter.cpp:201 and ast/rewriter/bit_blaster/bit_blaster_rewriter.cpp:596:default64 reset/reuse names with symbol/unsigned fragments.
- src/util/smt2_util.cpp:61:default64 quoted-symbol byte escaping.
- src/muz/fp/datalog_parser.cpp:238:default64 lexer field; chars/reset/CStr at token publication.
- src/muz/transforms/dl_mk_magic_symbolic.cpp:116,126:64-byte query/answer symbols.
- src/smt/theory_str.cpp:489:64-byte fresh symbols with unsigned IDs.
- src/cmd_context/parametric_cmd.h:29/.cpp:24:owned default64 help description, CString append and CStr output.
- src/test/string_buffer.cpp:25,31,41-42:default64,32,128 source fixtures.

## Numeric/source undefined-behavior boundary

Source append(unsigned) calls sprintf("%d", unsigned), not %u. Scratch
.cache/string_buffer_unsigned_probe.c compiled with -Wformat-signedness records
the type-mismatch warning and AMD64 observed output2147483647, -2147483648, -1
for INT_MAX, INT_MAX+1, UINT_MAX. Probe source/binary/temp are below golang/.cache.
This is not claimed portable well-defined C high-bit behavior. The parent-approved
Go adapter explicitly casts uint32→int32 then prints signed decimal for all bits,
preserving the fork baseline's observed fresh-ID/counter output rather than
silently replacing it with mathematical unsigned decimal. Permanent boundary
fixtures cover the complete low/high-bit endpoints independently of the probe.

Int/long conversions use local [24]byte scratch, include source terminator space,
and delegate one bulk append. No PrintfFormatter owner/workspace allocation or
host formatted intermediate string is invented. MinInt32/MinInt64 magnitude
conversion avoids Go signed negation overflow. LongBits defaults64 (LP64), accepts
only32/64, and32 rejects values outside signed32 before buffer actions. This
ABI-range adapter does not claim the rejected value fits a source32-bit long.

## Budget, roots, and lifetime behavior

Actual concrete owner reflect-size includes the embedded inline array, mutex,
self pointer, control scalars, and overflow slice header. It is charged **once**
at construction. Stats.InlineBytes is informational and already in OwnerBytes;
no duplicate inline charge/backing allocation exists. Byte arrays are pointer-free,
and no backing slice/pointer escapes a public method. Overflow capacity doubles,
reserves full new bytes while old overflow is charged, copies pos bytes, then
releases old overflow. Embedded initial array remains part of the owner charge
even after overflow allocation. Owner self-identity rejects shallow copies.
Cold mutexes serialize an owner; borrowed shared external Budgets still require
caller serialization across other owners. No owner stores per-record callbacks.

Bulk CString/symbol/buffer appends may publish several successful capacity
expansions before a later failure. Position and appended bytes stay unchanged
until all growth completes, preserving source successful allocation prefixes.
Char append publishes one byte only after its required expansion. Source numeric
Symbol k! bulk prefix completes before integer suffix append, so suffix failure
retains k!. Plain symbols read the actual borrowed manager checked bytes directly
and preserve bulk semantics; no intermediate Symbol.String allocation is used.
The manager is borrowed only for the synchronous call, not retained in the buffer.

CStr is a mutation: pos==capacity forces one expansion, writes terminator at pos,
and leaves pos unchanged. Reset keeps all capacity/content; subsequent empty
CStr writes byte0 at the new pos0. Returned immutable text copies are caller-owned
output boundary storage and survive reset/close; this replaces source ephemeral
raw-pointer borrowing. Read exposes only a checked scalar byte within pos.
Size preserves source unsigned32 cast; Position exposes the widened counter.
Checked host/uint64 addition and capacity doubling reject unrepresentable Go
backing sizes rather than wrapping allocation arithmetic.

All9 cross-buffer routes acquire one cold global append mutex then both owner
mutexes. Source CStr completes first, including source growth and terminator,
even if the destination's later append reservation fails. A source CStr failure
leaves destination untouched. Source/destination may have different Budgets.
Self append works when no heap source pointer is freed, including inline-to-heap
growth (embedded inline source remains live), full-heap CStr growth before
append, or a nonfull heap append that fits. A nonfull heap self append requiring
reallocation would use freed source C storage; ErrStringBufferAlias rejects that
specific source undefined case before actions. It does not invent snapshot
semantics for a dangling pointer. Embedded source NUL stops buffer copy at first
NUL, regardless of source full pos.

## Tests and evidence

Source/lifecycle/model fixtures were written first and observed missing concrete
API RED3bf853. Initial7 groups passed after implementation;3 additional meaningful
physical-budget/ownership boundaries passed without manufacturing production RED.
Ten registered groups:

- TestStringBufferSourceThreeAssertions
- TestStringBufferInlineTerminatorResetAndBytes
- TestStringBufferNumericUnsignedAdapterAndLongDomains
- TestStringBufferFailureCapacityPrefixesAndImmediateBudgets
- TestStringBufferNineRoutesSourceCStrFirstAndSelfCases
- TestStringBufferSymbolBulkAndNumericPrefixFailure
- TestStringBufferIndependentByteModelAndWarmAllocations (1800 operations,0 warmed append Go allocations)
- TestStringBufferPhysicalInlineOwnerChargesAndExactPeak
- TestStringBufferSourceFailureBeforeDestAndReturnedTextLifetime
- TestStringBufferOwnerConfigurationAndSymbolGuards

Native all10 named RUN/PASS1dcab1. No source checkout/C compiler needed at test runtime.

Final frozen gates (all terminal exit0):

- Full native `go test ./... -count=1`: output108c32; mpnum10.197s.
- Full checked/debug `go test -tags='g3checked g3debug' ./... -count=1`: outputa1971c; mpnum10.045s.
- Targeted checked/debug race `go test -tags='g3checked g3debug' -race -run '^TestStringBuffer' -count=1 -v .`: output52d5f1, all10 named RUN/PASS.
- Actual checked/debug JS: compiled `.cache/string_buffer.test.wasm` (outputd98000), then minimal Node wrapper with `-test.run '^TestStringBuffer' -test.v -test.count=1`: outputf295eb, all10 named RUN/PASS, including the independent model and warm allocation test.
- Full `go vet ./...`: output729d46.
- Checked/debug WASI test compilation: `.cache/string_buffer.wasi.test.wasm`, output4484b7.
- Source/no-generics `go run ./cmd/portcheck`: outputd493ee, 2102 units/677 APIs; 29ported/51partial/2022unported before central integration of this packet.

Production and tests remain frozen at the hashes below. Only this assigned status
file was updated after the gates; no central metadata or Git state was changed.

Frozen production/test hashes:

```
e41b5870098cce2f0a9bf64ce48db4cf755f877778153c32cf2c22a9df09c7c9  golang/util_string_buffer.go
6e4d439a974e1481b39e65b39576b76cfdcb006a88d447bbb55fab2b1449458e  golang/util_string_buffer_test.go
```

Source SHA256:

```
dac7740d1679a51d8fce15a622850a42ef8fae82f4e80625fec02be2613d3b5b  src/util/string_buffer.h
e86a4980e1f170e7572fd29b89c820acb72f859bb3b3188a78b49654c0b43b40  src/test/string_buffer.cpp
60ce765fb3b761ec178ceec7db59f7ddbce00dd0fad5148f6a8ec234f62b4fbf  src/util/symbol.h
840d0480a03ddb1b0ed529338baebb4d32ad385db6fabd5f7037bcaa2b2db1bb  src/util/symbol.cpp
```
