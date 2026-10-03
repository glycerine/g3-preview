# Explicit numerical owner copying

GO_PORT_PLAN.md B12 requires rational values moving between owners to be copied
into the destination manager. Source rational.h/.cpp uses one global manager,
so this is an explicit ownership adaptation, not a new source arithmetic method
or a claim that AST translation is implemented.

`mp_copy.go` adds concrete IntegerManager.CopyFrom and RationalManager.CopyFrom.
Same-owner calls delegate to the existing source Copy implementation. A different
source manager supplies indexed live digits; they never become an escaping slice,
decimal string, hidden buffer, shared cell or foreign destination handle.
Integer copy retains source sign, small/big tag, live size, source capacity on
allocation and existing destination capacity when it can be reused, following
mpz.h set and mpz.cpp big_set. Fallible preparation precedes destination writes.

Rational copy stages numerator and denominator in destination-owned cells, then
moves the complete pair. It does not normalize: unreduced2/4, negative denominators,
zero numerators with nonunit denominator and big representations remain intact.
Partial pair allocation is cleaned in lexical component order; failed publication
preserves the entire destination. All physical old/new cells and Words are charged
by the existing numerical service. Completed destination copies outlive source
reset/deletion/Close independently. No new owning manager header or cache is added.

These are low-level raw-manager APIs. Raw Int/Rational records do not contain
provenance; callers must supply the genuine owner of each record, as required by
existing manager operations. They cannot detect all foreign records whose
owner-local slots/generations happen to coincide. The separately implemented
rational service validates scalar provenance before invoking this bridge.
Nil managers receive ErrConfig; nil numerals and closed owners use existing typed
guards. One identical raw address across distinct managers receives ErrConfig
because a single owning record cannot belong to both domains. Same-owner identity
remains the existing checked no-op. No public foreign-handle safety is falsely
claimed from low-level slot checks alone.

Five tests first produced missing-CopyFrom API RED in an isolated committed module.
GREEN tests verify source storage shape, reused allocation under FailAfter0,
independent seeded exact values after source Close, raw rational representations
in both manager modes, and warm zero-allocation copies. Failure sweeps occupy
actual free cell slots, then advance every reservation prefix through success
for small/big integer/rational outputs. Before rendering or cleanup, each failure
requires exact destination live-cell/Words baselines, unchanged destination raw
identity and unchanged source raw identity/cells/Words/budget. No Close call can
mask leaked temporary ownership. Final whole-owner budgets return to zero.

The isolated module passes full native tests, actual checked JS execution of all
five new names, checked vet and WASI cross-build. Full checked-race module tests passed; the added big-rational warm-copy fixture
also passed fresh targeted checked-race tests. WASI remains
compilation evidence only. No Go generics, unsafe, boxed numeral records, production
math/big arithmetic or additional storage service is introduced. Rational wrapper,
AST parameters and AST translation remain separate incomplete units.
