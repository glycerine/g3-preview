# Concrete symbol service

The complete selected symbol.h/cpp behavior has a concrete owner service in
util_symbol.go. Symbol is a pointer-free scalar origin/slot/generation/tag record.
The source global manager becomes an explicitly constructed SymbolManager which
may be shared wherever source-wide interning identity is required. Strings from
separate services belong to separate identity domains and cannot be reinterpreted
by a foreign service. Numeric/null identities do not own spelling storage.

NewSymbolManager/Close map initialize_symbols/finalize_symbols and the internal
symbol-table lifetime. Intern, InternNullable and AssignString map the string
constructor/assignment; NumericalSymbol and raw Symbol{} map numeric/null
construction. Source equality/inequality use Go's comparable scalar record.
IsNumerical/Number, private mark/unmark, DummySymbol, String/Display/BareString,
Hash, Contains, Size, EqualString and Less preserve the source predicates,
rendering, cached hash, C-string equality and byte order. SymbolToken/FromToken
replace c_ptr/mk_symbol_from_c_ptr with checked opaque identities, not an ABI.
BareString represents a nullable pointer through a presence flag and copies its
text to caller-owned output. Display is the common stream/string-buffer output.

Intern follows source growth-before-duplicate behavior, stores sizeof(size_t)
bytes of cached hash immediately before the spelling, and includes its NUL.
The source64 hash prefix has zero high bytes. Region alignment selects the
logical page/header geometry (4 or 8); numerical symbols retain the full uint32
AMD64 baseline on every Go target because no integer is packed into an address.
Source tag0/null, tag1/numerical and mark+2/unmark-2 are explicit. Marked numeric
values are not numerical. UINT32_MAX renders k!-1, hashes as UINT32_MAX and has
source size35; numerical size is not rendered text length. Null String is
<null> while Display is null. Contains(0) includes the spelling terminator.
Source invalid null/marked data dereferences are explicit precondition errors.
Less returns false for identical marked/dummy values before any dereference,
as the source does. Its signed-char comparisons and termination-before-next-byte
checks are retained, including nonempty-prefix ordering before negative bytes.

Cold owner, region/identity/descriptors, page backing/header/directory, logs and
hash-table backing are budgeted exactly once. The service has no Reset and
preserves all strings until Close. Table closes before its spelling region.
The cold mutex covers intern preparation/allocation/publication and all region
reads/teardown; hot symbols contain no mutexes, pointers, strings or interfaces.
The caller owns returned Go strings; no temporary budget reservation claims to
account for an escaped output's lifetime.

Source get_str inserts its transient query before spelling allocation, then
replaces the entry pointer. The storage adaptation prepares source growth/probe
without publishing a caller string, allocates and fills owned region backing,
then publishes the owned key under the same lock. No query is retained after
allocation failure. Preparation may retain grown table/descriptor capacity after
a later failure; assignment publishes only on success. Raw arena refs remain
owner-local; symbol records and table keys carry unique scalar origin IDs.

All 23 active src/test/symbol.cpp assertions and exact foo boo foo newline were
ported first and failed RED for missing Symbol APIs, then passed GREEN. Added
BareString and identical-marked Less cases also went RED before their changes.
Independent finite raw-byte identity/order checks cover both pointer geometries,
including signed high bytes, empty strings, and source prefix termination order.
Tests inspect physical hash prefixes and complete budget graph totals, cover
NUL truncation, tag/null/numeric quirks, opaque tokens and foreign owners,
constructor/failure prefixes, publication, close cleanup, source duplicate
pre-growth failure, warm zero allocations, and concurrent canonical interning.

The string->int map and scoped symbol_table trail algorithms remain separate
unported components. A source global service should be shared by future solver
owners when its identity domain is needed; separate services are not implicitly
merged. Full AST/solver/proof/interpolation behavior remains unimplemented.

Integration gates: full native suite, actual g3checked race and vet pass. Actual
checked Nodejs/wasm output explicitly registers and passes all six Symbol tests;
WASI compilation passes. The arena agent independently reviewed the complete
source methods and assertion map. Its prepared-miss epoch finding was reproduced
RED and corrected GREEN in StrHashtable before spelling allocation. Prospective
Symbol commit runs the whole hermetic native module suite.
