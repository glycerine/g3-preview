# Concrete AST family registry

`ast_family.go` translates the real `family_manager` in ast.h:207-236 and
ast.cpp:144-166. It borrows the existing SymbolManager and owns the committed
SymbolInt32Table plus a pointer-free registration-order Symbol vector. No AST
manager, nodes, plugins, proofs or solver behavior is supplied by this registry.
The source has no standalone family-manager assertion suite; the new fixtures
are explicitly added tests, not fabricated source ENSURE counts.

## Source methods and publication

The default constructor starts next ID0 with the source table and empty names.
MkID first finds an existing binding; otherwise it captures next, increments
next, inserts the symbol/ID into the table, then pushes the symbol into names.
GetID returns the registered ID or source null_family_id=-1. HasSymbol delegates
to the table. GetName returns null for negative/out-of-range IDs; HasID tests
that same name-vector bound. AppendDom/AppendRange append to caller outputs in
source hash-table bucket order, not registration order. The six basic/label/
pattern/model-value/user-sort/arith names are tested in the actual init order,
without pretending to instantiate the AST manager or any plugin.

Completed source statements remain observable on predictable failure. A name
reservation failure retains the increment and already inserted mapping, so
GetID can succeed while HasID is false and GetName returns null. Retrying that
same symbol returns its mapping and does not repair names. A failed table
growth retains the increment but no mapping; a successful later retry takes
the next ID and appends into the prior vector position, preserving the source
ID/name gap. Explicit permanent tests freeze both prefixes before diagnostics
or cleanup. This registry does not silently make source compound operations
transactional. Underlying table reservation guarantees retain their existing
documented adaptation.

## Physical ownership and domains

The actual cold FamilyManager header is charged once. The separately allocated
SymbolInt32Table header/backing are charged by that real nested owner. Each
names array charges its full actual Symbol capacity using target reflect size;
source capacities are2,3,5,8,12,... . Growth reserves all new bytes while old
bytes remain charged, copies the logical prefix, publishes, then refunds old.
An exact limit test succeeds at old2+new3 Symbol cells and fails one byte below,
retaining the mapping prefix. The borrowed symbol graph is never double charged.
Close releases names then table (reverse source member order) and its header,
never closes symbols, and is idempotent. Close still refunds physical storage
after the borrowed symbols are closed, while ordinary operations diagnose that
invalid borrowed lifetime. Full nested Used sums and constructor prefixes are
checked immediately.

FamilyID is a concrete signed32 alias, distinct from physical slots/generations.
Source signed next++ overflow is undefined; a fresh registration at INT_MAX is
a typed bounds error before mutation. Existing duplicate lookup still works.
Null/dummy/marked/foreign symbols retain existing table guards. Caller-owned
enumeration outputs are outside the owner budget. Cold mutexes serialize the
registry and acquire table then symbol locks through existing operations; no
symbol lock is held across public table calls. Shared external Budgets still
require caller confinement across owners. Cold self identity rejects copied
headers; no hot entry has a Go pointer/interface. Source implicit owning copy
construction/assignment is not exported by this packet and remains an explicit
unimplemented surface. The actual AST copy_families_plugins route registers
family names in order; that full manager/plugin route remains open.

## TDD and verification

Six initial groups observed missing-API RED109bf1 in isolated HEAD82e17f0 plus
only ast_family_test.go. Concrete implementation passed those tests; the seventh
exact-overlap/borrowed-lifetime test passed immediately without manufactured RED.
The final frozen prospective tree `.cache/release/family-smt2-integrated` includes
these two Go files and the four SMT2/escaped files together. Full fresh
CGO-disabled native and checked-debug modules passed (mpnum10.361s/10.256s), all15
named family/SMT2/escaped checked-debug race and actual JS RUN/PASS groups passed,
and checked-debug vet, WASI build and source/no-generics audit passed. WASI build
is not runtime evidence. Logs: `.cache/family-smt2-integrated-{race,js}.log`.
Root independently matched every frozen Go SHA. Warm repeated lookup under
FailAfter0 allocates zero host objects and requires no reservation.

Named family tests:

- TestFamilySourceRegistrationAndQueries
- TestFamilySourceEnumerationOrderAndPrefixes
- TestFamilyNameFailureRetainsPublishedMap
- TestFamilyMapFailureRetainsNextIDAndRetryGap
- TestFamilyBudgetConstructorAndNameGrowth
- TestFamilyWarmGuardsAndSignedCounterBound
- TestFamilyExactNameGrowthOverlapAndBorrowedLifetime

Frozen Go SHA256:

```
2046faa039409fcf45aafbe8a2403afce0021ab3e869682d0f37b1443cb0aeba  ast_family.go
9240c5b9fd0f9b257b74bd50efaac8b0e766c96866295a2f4346bca94fc443a4  ast_family_test.go
```

Complete ast.h/.cpp remain partial: all five node kinds, real eager normal/format
manager construction, plugin behavior, parameter attachment/deletion, structural
interning, expression/dependency arrays and every higher consumer are still open.
See AST_CORE_PORT_STATUS.md; a green registry does not close the AST core.
