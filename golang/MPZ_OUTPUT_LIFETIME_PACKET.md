# Expanded numeral output lifetime and transitive teardown diff proposal

Read-only existing production proposal. Base203486f includes the additive source correction. No consumer/destructor implementation edits have been made. Parent approved the listed lifecycle regions after source review; selected GCD bodies remain numeric-owner controlled. No race or WASM/WASI checks will be launched following the new user steering.

## Source versus Go assembly owner

The source BIG integer display action is sign insertion BEFORE construction of its inline1024 char sbuffer, requested uint32 `11*size`, then mpn conversion, then C-string insertion, then char-sbuffer destruction. Source SMALL integer insertion directly uses out<<m_val, with NO source char buffer or three Natural limb locals. The three mpn limb locals exist only for >1 limb conversion and destruct in reverse order BEFORE source C-string insertion returns to integer display. Each integer displayed by rational/binary/infinitesimal composition gets its own logical char/local-limb lifetime, not an accumulated persistent manager formatting scratch.

Propose NEW concrete `numeralOutputScope` in mp_format_output.go. One charged cold owner exists per public composed output operation, created before acquiring source formatting roots. It contains:

- private owner/self identity, borrowed Budget/Runtime, headerBytes;
- one [1024]byte Go assembly inline buffer and initialized/position metadata; optional actual []byte assembly overflow with its capacity charge;
- one concrete sourceCharBuffer with actual [1024]byte inline/source initialized mask, source uint32 pos/capacity, raw/pending memory.Blocks;
- three sourceLimbBuffer locals for the exact mpn conversion; scalar active/cleanup cursors;
- cold original/cleanup errors, with no additional exception allocation on failure.

The assembly bytes are a Go buffered-output adapter replacing the external ostream/ostringstream boundary, NOT source memory::allocate events. Actual assembly owner header/backing is charged by reflected physical size/capacity; old+new assembly growth peaks reserve before make, copy prior bytes, release old backing. Source char/limb overflow calls the actual shared Runtime and affects true source counters. No host intermediate text/string or fmt. Only `string(assembledBytes)` produces one final caller-owned string; FormatInto copies only after successful preparation, preserving the current documented buffered-destination adapter and ErrShortBuffer domain. No source-counter charge for that caller-owned string.

A single cold output-scope heap allocation, if compiler escape requires it, is explicitly charged; it is not hidden per-record boxing. Arithmetic warmed Mul/Div must still allocate zero Go objects. Output allocation counts must be measured and reported honestly rather than assert that this Go owner is C++ stack storage. An optional stack-friendly output implementation may remove that allocation later only if actual root retention remains correct.

## Private assembly binding API

Proposed signatures:

```go
func (n *NaturalManager) beginOutput() (*numeralOutputScope,error)
func (n *NaturalManager) endOutput(out *numeralOutputScope, original error) error
```

Natural has one cold borrowed `activeOutput *numeralOutputScope` field while an operation runs. beginOutput rejects reentrancy before publishing a second output owner; preparation helpers reserve/read/write ONLY through that exact owner. Existing reserveText/textRead/textWrite/reverseText/prepareInteger offset semantics can temporarily preserve signatures, but use this operation-owned assembly, never a persistent Natural SmallBytes owner. At success/failure endOutput clears that borrowed pointer exactly once. It cannot invoke public formatting while holding a service mutex; service helpers are private and use existing service->manager confinement.

Every public non-alias formatting entrypoint brackets its entire preparation+final copy with begin/end, including errors before publishText/finalString. Public alias methods delegate without constructing a second scope. endOutput joins errors without a new wrapper allocation: preserve original operation cause; attempt exact pending/source-local cleanup and report a retained owning error only when cleanup actually refuses. Parent selects retention of the ALREADY charged output scope itself on any cleanup refusal: Natural owns one cold pendingOutput pointer after activeOutput is cleared. No eight-root transfer directory/fallback is needed. Its actual header, assembly and source raw roots stay charged until checked cleanup completes. The scope implements the cold error interface itself, with fixed Error label, original/cleanup causes and cleanup stage/cursor; no new exception allocation. A new operation rejects a pending owning context before replacement. Natural CloseChecked cleans that actual context before its Runtime/header, and borrowed Runtime never closes. Cleanup retry does not resume formatting, emit delayed text, refree physically completed blocks or repair a failed TLS synchronization. Scope/parent cause links must remain acyclic; ordinary formatting arithmetic returns leaf transport causes, and a self/ancestor owner must never be stored as its own unwrapping cause. Final caller string is produced ONLY after source-local cleanup succeeds; assembly charges are refunded after final copy.

SourceCharBuffer growth is source uint32 doubling, actual profile size_t byte request, copy source POS bytes only, old overflow free before pointer/capacity publication. Char construction with `(11*size,0)` runs source resize/push zeroing; no premature UINT32 domain rejection. Source buffers remain separate from assembly. Per-integer conversion resets local logical lifetimes/initialization, and any overflow must actually free after that integer's source display; no retained capacity from one numeral to the next.

## Exact entrypoints requiring output bracketing

| Existing file | Scope-owning methods | Delegating aliases or private composition |
| --- | --- | --- |
| mp_format.go | Natural FormatInto/String; Integer FormatInto/String; Rational FormatInto/String | prepareNatural/prepareInteger/prepareRational compose into active scope |
| mp_display.go | Integer FormatSMT2Into/SMT2String; Rational FormatSMT2Into/SMT2String/FormatDecimalInto/DecimalString/FormatPrettyInto | ToRationalString delegates String; appendDisplay/prepareSMT2/prepareDecimal private |
| mpbq_format.go | Binary String/FormatInto/PrettyString/FormatPrettyInto/SMT2String/FormatSMT2Into/DecimalString/FormatDecimalInto/IntervalDecimalString/FormatIntervalDecimalInto | DecimalStringDefault delegates; appendUnsigned/prepareBinary/prepareBinarySMT2/prepareBinaryDecimal/prepareIntervalDecimal private |
| mpbqi_format.go | BinaryInterval String/FormatInto | appendBound/prepareInterval private |
| mpff_format.go | FastFloat String/RawString/SMT2String/DecimalString/FormatInto/FormatRawInto/FormatSMT2Into/FormatDecimalInto | RationalString/PrettyString/FormatPrettyInto/DecimalStringDefault32_128 delegate; appendText/appendUnsigned/appendSigned/prepareDisplay/prepareDecimal/prepareNatural private |
| inf_rational.go | RationalService InfString | lock/check then one output scope, source arithmetic expressions unchanged |
| inf_int_rational.go | RationalService InfIntString | same lock order/private output helpers |
| inf_eps_rational.go | RationalService InfEpsInfString | private append rational/epsilon and in-place byte shifts stay on active scope |

Do not blanket wrap both aliases and targets. Validate source/current input before scope allocation where the current method does so; source formatting metadata preparation is an explicit Go safety adapter, and required input error prefixes are tested. Check actual source/public byte order at every outer route; this table alone is not evidence that all routes are translated.

Tests FIRST: real per-call char/limb allocation/free counts for >16 and >1024 paths; source sign before char refusal; >1 versus one-limb branch; rational numerator locals actually destroyed before denominator creation; source conversion failure leaves prior assembly/source writes, buffered caller dst unchanged; every public output early-return releases assembly charge; actual beforefree/postfree local cleanup roots retained/continued once; borrowed runtime unchanged; single final string allocation boundary; actual measured inline output allocations; no retained formatting Runtime root after success. No source-output compiler dependency in permanent tests.

## Exact transitive constructor/destructor region edits proposed

mpq.go: NewRationalManager allocate self owner immediately after reflected header reserve; on Integer constructor failure release unused Rational header and forward exact child owner error; on success retain z immediately. Add private component/root cursor state. Replace error-ignoring drop ONLY for owning destruction with checked numerator then denominator cleanup. SourceFinalize root order nTmp/addTmp1/addTmp2/ltTmp1/ltTmp2/addMulTmp then child SourceFinalize. CloseChecked performs same actual leaf prefix then child owning CloseChecked and header release. Existing void Close panics the SAME retained owner on refusal. Public arithmetic addSub/mul/div/normalize bodies remain unchanged. Existing Del/drop arithmetic sites must be inventoried for cleanup result propagation separately, not silently change formulas.

rational_service.go: constructor header allocation/self setup before child; forward failed child unchanged if service owns no further roots. For later zero/one/minusOne/inf init failure, retain initialized source prefix and original cause, attempt checked cleanup, return same service owner if refused. Replace destroyCache with cursor-aware ascending entry NUM then DEN; do not release backing until all entries complete. Close/SourceFinalize stage infRat→infInt→cache→zero/one/minusOne→manager; source order remains current audited order. Service mu is locked once; private continuation helpers do not recursively enter public methods. Header released only after pending locals/cache/child fully cleaned. Stats must expose actual physical topology, not report fake Words roots.

inf_rational.go: ONLY initialization/destruction helpers initially. Initialization records each genuinely initialized first/second component. Finalization each constant zero/one/minusOne SECOND then FIRST; clear ownerID only after both complete. It must not use a normal numeral-read guard to reject an owned physically-released-but-pending cleanup token.

inf_int_rational.go: ONLY corresponding helpers initially; first rational cleanup per zero/one/minusOne, clear token only after completion. Source signed epsilon and arithmetic publication bodies unchanged. Output-only InfString/InfIntString regions are additional distinct changes subject to table approval.

Tests FIRST: Integer→Rational→Service construction error retains most-derived actual root graph, no extra error/header allocation; distinct original/cleanup causes and bounded acyclic Unwrap chain; partial NUM-success/DEN-refusal preserves denominator and component cursor; partial cache entry and header charge; actual postphysical acknowledgement without TLS repair; source destructor order and owning Close extra direct-cell cleanup; foreign/copied owners rejected while exact-owner failedRef cleanup remains allowed.

## Exact obsolete topology diagnostics and replacement

upolynomial_modes.go: IntegerWordsRefs and IntegerNaturalWordsRefs currently count z.words.Live and Natural arena refs. These fields cannot keep fake live payloads. Proposed zero legacy fields plus explicit actual allocator/runtime/Natural block fields; report borrowed Runtime totals as domain totals, not exclusive Integer charges. Numeric descriptor/cached Int counts remain real slab ledgers. Container/parent owns this diff; current frozen adoption files remain untouched.

rational_service.go Stats currently returns LivePayloads from z.words. Replace with explicit actual cell-object/root statistics (legacy field classification clearly documented); CacheLength/Capacity/Bytes retain actual source vector behavior. Do not equate one pooled cell with one raw malloc block.

mpq_ops_test.go qopsAuditLiveOwnership, mpz_source_publication_test.go named Words graph and numerous existing ownership tests assume one cell→Words ref plus four Natural refs. Replacement tests must count genuine descriptor roots, allocator pooled chunks/direct objects/pending blocks, Natural actual overflow/pending roots and real directory/initialization/header bytes. Runtime may be shared, so whole-domain root totals must include other known owners or use an isolated fixture. Failed source allocation requests can increase accounting without physical roots; never require source accounting equals live physical/Budget bytes.

Affected existing test files requiring observed RED and individual review before edits: mpz_test.go/mpz_owner_budget_test.go/mpz_source_publication_test.go/mpz_mul_div_test.go/mp_format_test.go/mp_display_test.go/mpff_ownership_test.go/rational_service_test.go/rational_service_inf_lifecycle_test.go/scoped_numeral_test.go/mpq_ops_test.go and additional formatting/cache tests using the same private text topology. This inventory is not permission to weaken any assertion or change helper ownership without coordination.

Cross-manager physical cell/coefficient adoption, actual process-global runtime/generated finalization and native stdout bindings remain required later. This packet does not claim full backend/solver completion or source-wide allocation coverage.


## Current implementation evidence and disposition

Parent authorized full lifecycle table/source-local integration and exact transitive constructor/destructor regions on203486f. Existing audit/test changes still need their own real named RED and permission. New helper prerequisites currently pass seven selected native and checked/debug groups: five sourceLimbBuffer groups plus two actual sourceCellBacking groups. They are NOT yet wired into mandatory Integer/Natural; the existing manager still uses its arena backend. No full-backend completion claim or metadata promotion. Separate sourceBuffer failure fixtures include genuine lower TLS refusal and actual copied-prefix guard RED. Full output/context/cursor implementation remains active work.
