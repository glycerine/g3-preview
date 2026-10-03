# Standalone AST sort_size

Complete standalone sort_size surface in src/ast/ast.h:291-333 and stream
insertion in ast.cpp:235-239 is implemented by ast_sort_size.go/test. Only those
two files and this status are owned/edited; no central metadata, other production
or Git state changed. Source Microsoft2006 notices retained. No Go generics,
unsafe, boxed hot records, mutable owner or fake AST interface introduced.
Other AST classes/managers/printers and declaration plugin algorithms remain open.

## Complete method mapping

| Source | Go |
| --- | --- |
| ast.h:292-302 enum, fields and private constructor | private uint32 kind and uint64 size; pointer-free SortSize |
| ast.h:304 default constructor | Go zero value, InfiniteSortSize |
| ast.h:305 uint64 constructor | NewSortSize |
| ast.h:306 copy constructor | plain scalar assignment; no retained ownership |
| ast.h:307-316 rational constructor | SortSizeFromRational: actual RationalService.IsUint64 then Uint64; else VeryBigSortSize |
| ast.h:317-319 factories | InfiniteSortSize, VeryBigSortSize, FiniteSortSize |
| ast.h:321-323 predicates | IsInfinite, IsVeryBig, IsFinite |
| ast.h:325 threshold | SortSizeIsVeryBigBase2(power), inclusive >=64 |
| ast.h:327 size and source SASSERT | Size, typed ErrSortSizeNotFinite before irrelevant field read |
| ast.cpp:235-239 stream insertion | String: infinite / very-big / unsigned finite decimal, local20 digits and one final caller string |

Private source enum0=finite/1=very-big/2=infinite becomes Go0=infinite/1=finite/
2=very-big to make the zero value match the source default. Enum values are not
an exposed ABI. Nonfinite size is canonical0 in Go; source default leaves m_size
unspecified. This adapter avoids propagating an unspecified field during value
copy. Finite0 and UINT64_MAX are legitimate source sizes. Size's precondition is
checked in all Go profiles rather than exposing irrelevant/unspecified release
nonfinite bits. External callers cannot construct another private kind.

## Actual borrowed rational route and ownership

The real owner-local RationalService and pointer-free owning token are borrowed
only during conversion; SortSize retains neither. IsUint64 validates input owner,
lifetime and actual numeral shape/value, then the source-success branch invokes
Uint64 separately. Negative/fraction/too-large become very-big, including negative
values mathematically unrelated to a cardinality: no invented domain validation.
Provenance/closed/deleted inputs propagate actual service errors; nil service is
an explicit typed owner precondition adapter. No setter/arithmetic/copy/finalizer,
reservation or text serialization is introduced by construction.

Callers must confine mutation/deletion of the borrowed operand across the two
source actions. Each real service method locks its owner; this wrapper does not
promise an atomic observation of a concurrently changed token. Result is an
independent scalar after owner destruction. Value storage is charged by its
future enclosing owner/container, never as an invented separate header. Returned
text is caller-owned; constant labels allocate no mutable storage.

## Actual consumers and source assertions

Read actual consumers: ast.cpp:858 Boolean sort cardinality2;
bv_decl_plugin.cpp:147-153 inclusive64 bit-size threshold and actual rational
power-of-two; array_decl_plugin.cpp:85,104 very-big;
fpa_decl_plugin.cpp:199-200 default then very-big;
dl_decl_plugin.cpp:78-91,111,117 and738 cardinality metadata;
datatype_decl_plugin.h:112-188 offset/plus/times/power/sort parameter evaluation
with source finite rational arithmetic. These algorithms are not ported by this
scalar packet. No fake sort_info, manager, expression or plugin is introduced.

No active src/test assertion directly targets this class. Searches matching
Z3_get_bv_sort_size test consumer different public API width metadata, not these
cardinality methods. Source implementation guards: ast.h:327 SASSERT(is_finite())
and ast.cpp:238 finite display SASSERT. Factories ensure valid display kind;
Size's guard is directly tested. Zero source ENSURE counts are invented.

## Tests-first evidence

Four tests were written FIRST; isolated missing-API RED2e88d1 was observed, then
all4 native named RUN/PASS c95165:

- TestSortSizeSourceDefaultFactoriesAndGuards
- TestSortSizeRationalActualUint64Classification
- TestSortSizeRationalBorrowedOwnerLifetime
- TestSortSizeIndependentModelPointerFreedomAndWarmAllocations

Coverage: finite0/1/2/UINT32_MAX/2^63/UINT64_MAX; threshold0/63/64/65/UINT32_MAX;
exact labels; independent2000 deterministic uint64 decimal cases; actual
negative/fraction/2^64 constructor inputs plus normalized4/2; foreign/null/deleted/
closed owner and result surviving Close; reflect pointer-freedom and zero warmed
predicates. Finite rational conversions succeed with FailAfter0 while immediate
borrowed Budget.Used and service cell/payload/cache stats remain unchanged.
Tests use real numerals, not fake coefficients or AST scaffolding.

Final isolated tree .cache/release/sort-size-final is HEAD at extraction plus
only these two assigned Go files. All archives/cache/temp/binaries are under
Go subtree. Terminal gate evidence follows after completion.

## SHA256

```
5d006f0ccd5f44f25bc8e0de9f629d49cd1df8fa1275bb121696cd922d0069fd  src/ast/ast.h
ac1de339a71c12fabaf951db2b0936bb1a84990b337ce268490633cb7bb5ff52  src/ast/ast.cpp
e43a2983cbaee86eea1e9048ee13870dfe0d1c4b163abb25fcfdb4d8589a31e0  ast_sort_size.go
228c109cb5ab1103529298cbc152a2a7448cb3d27f78a4a7a2583dbc2324ce6e  ast_sort_size_test.go
```

## Final frozen gates

All terminal exit0, no live gate handles:

- Full native `go test ./... -count=1`: ec61d0.
- Full checked/debug `go test -tags='g3checked g3debug' ./... -count=1`: 7fb732.
- Targeted checked/debug race: a287c4, all4 named RUN/PASS.
- Checked/debug JS compile f00077, then actual minimal Node execution
  `-test.run '^TestSortSize' -test.v -test.count=1`: 7dae29, all4 named RUN/PASS.
- Full vet: 1cbd1d.
- Checked/debug WASI compile: a2688c.
- Source/no-generics audit with explicit reference root: fd1656,
  2102units/677APIs/32ported/56partial/2014unported before this packet integration.

Production/tests remain frozen at the hashes above. All other AST units remain
open except separately reviewed/committed packets; no whole ast.h/.cpp completion
is claimed by this standalone scalar port.

## Coordinator integration

Frozen `.cache/release/parray-sort-size-integrated` contains the two SortSize
files and final two persistent-array files atop HEAD6c30500. Full fresh
CGO-disabled native and checked-debug modules passed (mpnum9.803s/9.833s).
All19 named checked-debug race and actual JS RUN/PASS groups were independently
verified; checked-debug vet and WASI package builds passed. Four Go hashes,
the two changed source rows and preserved source inventories/API entries were
checked. Logs: `.cache/parray-sort-size-integrated-{race,js}.log`.
WASI build is not runtime evidence.
