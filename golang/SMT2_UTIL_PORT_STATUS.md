# SMT2 symbol utilities and complete escaped byte adapter

The four smt2_util functions and complete escaped byte adapter are implemented
in util_smt2_util.go/test and util_escaped.go/test. Only those four files and this
assigned status were edited. No central metadata or Git state changed. No Go
generics, unsafe, cgo or boxed hot records were introduced. Applicable source
Microsoft2011 notices are retained. Full AST printers remain unported.

## Complete source surface and concrete names

| Source | Concrete Go API | Exact behavior |
| --- | --- | --- |
| smt2_util.h:24; .cpp:21-29 | IsSMT2SimpleSymbolChar(byte) bool | ASCII digits/letters and exactly ~!@$%^&*_-+=<>.?/; no whitespace, colon, pipe, backslash or non-ASCII byte |
| smt2_util.h:25; .cpp:31-51 | IsSMT2QuotedCString(*string) bool | Nullable borrowed C string; stop first NUL; null/empty false; first decimal digit true; special outer-pipe scan; otherwise any non-simple byte true |
| smt2_util.h:26; .cpp:53-57 | IsSMT2QuotedSymbol(*SymbolManager, Symbol) (bool,error) | Validate live owner/plain token; numerical false; null delegates nullable false; named spelling read directly from owner bytes |
| smt2_util.h:27; .cpp:59-72 | MkSMT2QuotedSymbol(*SymbolManager, Symbol, *arena.Budget) (string,error) | Assert the source predicate before local buffer construction; open pipe, bytewise prepend backslash before pipe/backslash, close pipe, CStr and final returned text |
| util.h:379-389; util.cpp:122-136 | Private escapedEnd helper | Null yields empty; first-NUL limit; trim only the maximal suffix of LF bytes, preserving all other bytes |
| util.h:385-389; util.cpp:138-152 | AppendEscapedCString(*StringBuffer64, *string, bool, uint32) error | Source stream actions: quote emits backslash then quote; every other byte emits unchanged; after each retained LF emit exactly indent spaces |
| util.h:388 stream insertion | EscapedCString(*string, bool, uint32, *arena.Budget) (string,error) | Optional cold convenience wrapper around the complete append path, temporary owner closed on every exit, one final caller-owned string |

The approved concrete APIs are implemented. ErrSMT2NeedsQuoting rejects an
invalid mk request before any buffer reservation. Null/numerical inputs return
false for the predicate but fail this constructor precondition. Marked/foreign,
stale and closed symbol inputs preserve existing SymbolManager error guards.
The byte-only pure predicate needs no owner, allocation or Unicode conversion.

## Source quirks that must survive

The predicate name is not a test for an already quoted string: true means the
source caller will generate quoting. Null and empty strings are false. A digit
anywhere after the first byte remains valid, but an initial digit is true even
when subsequent bytes are simple. This is an exact whitelist, not an SMT-LIB
keyword/reserved-word validator: names such as true are not specially quoted.

When the C spelling begins and ends with pipe and has length at least2, interior
backslash immediately followed by backslash/pipe is skipped only when its escaped
byte is before the closing delimiter (i+2<len). Every other interior pipe or
backslash makes the predicate true. All other interior bytes, including spaces,
digits and control bytes, are accepted in this special branch. Thus ||, |a b|,
and correctly escaped interior delimiters are false. A backslash immediately
before the closing pipe is true. A single pipe is true. Do not add a generic
parser, unescape existing names, or strip existing delimiters.

MkSMT2QuotedSymbol always wraps and escapes the full bare spelling, even when
that spelling already has malformed delimiters; it does not repair/remove them.
The source uses successive char appends, not one precomputed bulk append. A
backslash prefix before an offending byte is a completed source action if a
later append fails. Preserve capacity doubling and CStr boundary growth exactly;
do not preallocate an exact final buffer or replace it with strings.Builder.

escaped is implemented in src/util/util.h/.cpp, not nonexistent util/escaped.h
or util/escaped.cpp. It escapes only double quote. Existing backslashes, CR,
tabs, high-bit bytes and interior newlines are unchanged. With trim=false a final
LF also gets indent spaces. With trim=true all final LF bytes vanish before
rendering; final CR is not trimmed. Null and empty have identical empty output
but remain distinct borrowed input domains. Source indentation is unsigned32;
Go must check storage arithmetic rather than wrap count/capacity computations.

## Symbol and byte ownership / predictable failure

Symbols remain pointer-free borrowed scalar owner/ref/generation tokens. Acquire
the existing manager mutex, validate checkPlain, and read m.key spelling length
and m.region bytes after the existing hash-prefix offset. Do not call
BareString/String to create an unbudgeted intermediate copy. Keep the borrow
only through the synchronous operation; outputs remain valid after manager Close.
Numeric and null routing must follow source semantics and cannot use Display's
null text or numerical k! spelling in the predicate/quote constructor.

Mk uses the committed concrete default StringBuffer64, with actual reflect-size
owner+inline storage charged once and full new overflow charged while old remains
live. The explicit supplied Budget owns temporary formatting storage; borrowing
the SymbolManager never duplicates its graph charges. Final returned immutable
text is the caller-owned output boundary, with no extra temporary text. Source
local initial array is stack storage; a physically charged Go temporary owner is
an explicit storage adaptation. Constructor/append/CStr failure closes the local
owner and restores temporary Used charges, while reservation fault-prefix
attempts remain consumed by existing Budget policy. Caller-owned inputs are not
charged. No new mutable global cache or retained owner is needed.

AppendEscapedCString models ostream with an existing concrete destination buffer,
so successful emitted byte/space prefixes remain published on failure; buffer
capacity growth and fault cleanup are the committed semantics. Source ostream
badbit/exceptions/custom stream-buffer state is not silently claimed by this
owned-byte adapter. The convenience wrapper can discard its temporary prefix
on error after releasing storage. Specify zero returned text on its failure,
while testing the explicit append API's visible partial output separately.
No AST object or fake AST interface is needed by any utility in this packet.

## Actual tests and consumers inspected

There is no standalone src/test/smt2_util.cpp, direct tst_smt2_util registration,
or active source test assertion for these four functions. The one active source
implementation assertion is smt2_util.cpp:60 SASSERT(is_smt2_quoted_symbol(s));
translate it as a checked constructor precondition test, not a fabricated
source numeric assertion count.

src/test/main.cpp:190 registers TST(escaped). src/test/escaped.cpp:21-35 has
thirteen print-only calls at22-34 and zero ENSURE assertions: five quote/text
variants, LF-only trim/nontrim variants, empty trim/nontrim, and null trim/nontrim.
Port all thirteen fixed printed expectations, explicitly preserving their source
print-only classification. SEXPR_PORT_STATUS.md documents five nonnull
default-trim/default-indent cases already covered only through sexpr. Those
tests do not close the helper's trim=true/null/indent surface. General indent
has no registered print fixture and needs independent exact fixtures.

Exact direct SMT2 consumers (none ported just by completing this dependency):

- ast/ast_smt2_pp.cpp:34-50 pp_fdecl_name;553-579 ensure_quote, ensure_quote_sym,
  pp_var;911-916 quantified declaration names.
- ast/ast_smt_pp.cpp:73-74 older printer buffer name insertion.
- model/model_smt2_pp.cpp:33-34 pp_symbol,109-110 constants,269-270 function names.
- interp/iz3pp.cpp:80-81 and169-170 declaration/name printing.
- cmd_context/basic_cmds.cpp:150-151 command names; opt/opt_context.cpp:1493-1494
  objective IDs; muz/base/dl_context.cpp:1111-1112 datalog names.
- util/statistics.cpp:74 simple-byte predicate while replacing non-simple key
  bytes with '-' (statistics renderer remains its own unit).

Escaped direct consumers inspected: sexpr.cpp:176 string rendering;
cmd_context/basic_cmds.cpp:42,46 help usage/description (indent4) and682 unknown
reason; cmd_context.cpp:1607 trim=true output; extra_cmds/dbg_cmds.cpp:243 error;
tactic_cmds.cpp:89,123; tactic/tactic.cpp:171; smt2parser.cpp:444,465 errors.
API z3_logger.h uses a separate ll_escaped class; do not conflate or claim it.

## AST dependency gate and finite tests-first coverage

ast_smt2_pp.h:24-34 depends on real ast/format.h, AST manager/nodes, params,
arith/bv/array/fpa/sequence/datalog/datatype declaration utilities. Environment
methods at36-70 return real format nodes and inspect real declarations/sorts.
Existing SymbolManager, StringBuffer64 and arena.Budget are enough to complete
the four utility methods and escaped independently. They do not unblock a fake
printer implementation in advance of actual AST/format/plugin prerequisites.
pp_fdecl_name additionally publishes len, creates manager-owned format nodes,
and handles null/numeric names; preserve those actions in its later domain port.

Tests written FIRST cover: exhaustive256-byte whitelist; independent finite symbol
grammar table and generated short spellings; nullable/empty/NUL/leading digit;
all outer-pipe/backslash boundary branches; numeric/null/marked/foreign/stale/
closed owners; all four source functions and the real SASSERT precondition;
quoted output for pipe/backslash/control/high-bit bytes with no Unicode changes;
exact63/64/65 boundary and escape-prefix growth failures; immediate local-budget
baseline before owner teardown; owner charges and old/new peaks; output surviving
manager Close; zero warmed predicate allocations. Escaped adds all thirteen
printed fixtures, independent full-byte quote model, trim suffix and indent
zero/nonzero including final LF, first NUL/null, failure after quote-prefix and
inside indentation, and caller-output lifetime. Finite bounds check huge indent
without attempting unbounded allocation. Do not invent source assertions.

Missing concrete API RED was observed (c61ff3). A meaningful review regression
then produced behavioral RED6c9865: the initial uint32 output prebound incorrectly
rejected a valid unsigned-max indentation before emitting any bytes. The fix uses
actual host-int representability only; unsigned-max indent under a finite budget
publishes LF and63 spaces before the first growth fails with ErrBudget. The full
source unsigned32 indentation domain is retained. Huge unlimited-budget requests
retain source potentially expensive behavior; no artificial UINT32 output cap.

Eight groups cover all13 actual print-only escaped inputs, the one constructor
implementation precondition, all256 byte domains, exhaustive short spelling
model, manager provenance/lifetimes, exact growth overlap and immediate failure
prefix accounting. The additional exact-growth/borrow-charge group passed against
existing production without manufactured RED. Predicate paths show zero warmed
Go allocations. Entire util.h/.cpp remain partial; escaped is complete for this
explicit owned-byte output adapter. Ostream external state remains separate.

Named groups:

- TestSMT2SimpleExhaustiveBytes
- TestSMT2QuotedCStringSourceBranches
- TestSMT2SymbolSourcePreconditionAndLifetime
- TestSMT2QuoteFaultBudgetAndIndependentBytes
- TestSMT2QuoteExactGrowthPeakAndNoBorrowCharges
- TestEscapedThirteenSourcePrintedFixtures
- TestEscapedIndependentByteIndentAndTrim
- TestEscapedPublishedFailurePrefixesAndBudgets

Lock ordering: Mk takes manager lock and uses private scalar methods on a fresh,
unpublished buffer; the buffer has no other caller and takes no shared-buffer
mutex. Thus it cannot invert the existing shared AppendSymbol buffer->manager
order. Pure symbol predicate takes manager only. Escaped takes its existing
shared destination buffer only and never a symbol lock. External Budget remains
owner-confined; separate locks do not make externally shared Budgets thread-safe.

Production/tests are frozen. Final gates are recorded below when terminal. The
isolated tree is .cache/release/smt2-head-final, git -C .. archive HEAD:golang at
82e17f0 plus only the four assigned Go files. Explicit parent Git directory is
necessary: archive from a subdirectory otherwise filters the archive to empty.
All scratch/archive/build caches/temp/output remain beneath golang/.cache.

## Audited source SHA256

```
37dfbf93e3e69e5c328fc17f316687bf53e9642394d32cebeae5915de948a797  src/util/smt2_util.h
d833e788ffce835f7882c0e0f99b42efd0d3d1e9eea3030490fa2ea6a1373d69  src/util/smt2_util.cpp
a87417423f67da7a52a6b7e98a40902289047848c1129d1b687f816932ce6a07  src/util/util.h
4ff022fc3d1a8a922cbd47f8289b6150b935f5265329d7d4bad900248218a3f8  src/util/util.cpp
b6268fc7569c7003f4a925e88b97c559c94ab0293c813f582828f8fb83930c97  src/test/escaped.cpp
```

## Final frozen evidence

All terminal exit0 on the isolated HEAD+four-files tree:

- Full native `go test ./... -count=1`: ca066d.
- Full checked/debug `go test -tags='g3checked g3debug' ./... -count=1`: 822657.
- Targeted checked/debug race, all8 named RUN/PASS: df939b.
- Checked/debug JS compilation: 5c2645; actual minimal Node wrapper with
  `-test.run '^(TestSMT2|TestEscaped)' -test.v -test.count=1`: 4a1409,
  all8 named RUN/PASS (not inferred from compilation or cached output).
- Full vet: ea5d74.
- Checked/debug WASI compilation: b5f9be.
- Source/no-generics audit: e10fa7, 2102units/677APIs,
  29ported/55partial/2018unported before central integration of this packet.

Production/test SHA256:

```
6e894c991703a3d61eaaa11709386a8a10001c1c775f2377e352a0cfc6311ac1  util_smt2_util.go
11a9b097c5ca63fcaf0ddf1179c54457b482ae3ca02de83b6d21c927444462e7  util_smt2_util_test.go
d83e8bdfbf2452942f9287176501f6ce9e27cff683b9fdc7e37d75d0ec9b390a  util_escaped.go
addd9b8d0e5a63bb61da4b2300a4a9c7973b1b90e01fc9d0120afd4b45846e2f  util_escaped_test.go
```

## Coordinator integration

The frozen `.cache/release/family-smt2-integrated` tree adds only these four
Go files and the real family registry to HEAD82e17f0. Full fresh CGO-disabled
native and checked-debug modules passed (mpnum10.361s/10.256s), alongside all15
named family/SMT2/escaped race and actual JS RUN/PASS groups. Checked-debug vet,
WASI package build and source/no-generics audit passed. All frozen Go hashes
and exactly five changed metadata rows were independently checked; source
hashes/assertions/conditionals/registrations and API inventory are unchanged.
WASI build is not runtime evidence. Logs: `.cache/family-smt2-integrated-{race,js}.log`.
