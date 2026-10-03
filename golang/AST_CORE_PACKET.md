# First real AST runtime packet

The selected first runtime-core packet is implemented and frozen against
HEAD9f8974d, and held for coordinator review and integration. The original
design was approved against HEADd0ceaf5. Whole-manager completion remains OPEN,
with the remaining source boundaries recorded below. This document
supersedes the provisional dependency list in AST_CORE_PORT_STATUS.md,
without changing that frozen document. The
no-generics rule in AGENTS.md overrides generic sketches in GO_PORT_PLAN.md.

## Scope and source boundary

The first runtime packet implements all five kinds, all six parameter kinds,
sort/function information, the actual structural chaining table, registration
and logical deletion, concrete owning references, expression persistent
arrays, and the five plugins used by eager initialization. It must support a
real normal manager and a real formatting manager. A Boolean-only manager,
lazy placeholder plugins, or an integer array with dummy value callbacks does
not meet this contract.

Evidence: ast.h parameter/node definitions and plugin classes; ast.cpp
parameter/decl_info sections, get_node_hash/compare_nodes, constructors and
init at 1297–1398, registration at 1666–1780, deletion at 1781–1840, factory
and quantifier sections; ast.h 1411–1497 and 1916–1930 for array ownership.
The basic plugin is in these files, not a separate basic_decl_plugin file.
Registered tests are src/test/ast.cpp and src/test/parray.cpp. Source vector
parameter destruction is ascending, per vector.h::destroy_elements; it must
not be replaced with reverse element cleanup by analogy to class members.

The complete manager remains the goal. This packet does not claim the copy
constructor, cross-manager AST translation, every proof builder, arithmetic
plugin/coercion implementation, dependency operations/arrays, marks, source
printers or resource-limit integration. Those have explicit follow-on gates
below. Constructor-owned empty dependency headers are real initialization
state, not callable stub dependency algorithms.

## Frozen external contracts to approve

All operations return a typed error when validation or a predictable reserve
fails. Zero ASTHandle represents source nullptr. Factories return non-owning
handles with source RC initially zero; obtaining a handle does not implicitly
pin it. Owned wrappers perform the source retain/release actions explicitly.
Public calls serialize through the manager; private helpers do not reacquire
the lock. Borrowed symbol/numeric owners must outlive both the normal manager
and its nested formatting manager.

```go
type ASTHandle struct { origin uint64; slot, generation uint32 } // private fields
type ASTKind uint16 // App=0, Var=1, Quantifier=2, Sort=3, FuncDecl=4
type ASTProofMode uint8 // Disabled=0, Enabled=1
type ASTManagerConfig struct {
    Symbols *SymbolManager
    Numerals *mpnum.RationalService
    Budget *arena.Budget
    ProofMode ASTProofMode
    OutOfMemory func() bool
}
func NewASTManager(ASTManagerConfig) (*ASTManager, error)
func NewASTFormatManager(ASTManagerConfig) (*ASTManager, error)
```

Normal construction creates a full nested format manager with proofs disabled
before initializing its own families/plugins. Format construction initializes
the same five real plugins and pins and has no nested manager. GetFormatManager
returns self for the latter. Both share supplied symbols/numerals, but have
different AST origins, IDs, interning, allocation and pins. The normal owner
owns/charges the nested manager; no global numeric or symbol manager appears.

The two source trace overloads need concrete adapters, not silent omission:
NewASTManagerWithTrace(config, borrowed io.Writer) and
NewASTManagerWithTraceFile(config, path string). io.Writer is a cold stream,
never a hot record. The file overload owns its file and EOF/close action; the
stream overload borrows it. Nested format construction borrows the same
stream. Literal source trace events must work before declaring these overloads
ported. Printer-backed error/diagnostic text stays an explicit printer gate;
it is not a reason to substitute a fake stream. Test files remain under golang.

Proposed methods needed by the first tests and five-kind audit:

```go
func (m *ASTManager) MkBoolSort() (ASTHandle, error)
func (m *ASTManager) MkConst(name Symbol, sort ASTHandle) (ASTHandle, error)
func (m *ASTManager) MkSort(name Symbol, info ASTSortInfo) (ASTHandle, error)
func (m *ASTManager) MkFuncDecl(name Symbol, domain []ASTHandle,
    rangeSort ASTHandle, info ASTFuncDeclInfo) (ASTHandle, error)
func (m *ASTManager) MkApp(decl ASTHandle, args []ASTHandle) (ASTHandle, error)
func (m *ASTManager) MkFamilySort(fid FamilyID, kind int32,
    parameters []ASTParameter) (ASTHandle, error)
func (m *ASTManager) MkFamilyFuncDecl(fid FamilyID, kind int32,
    parameters []ASTParameter, domain []ASTHandle,
    rangeSort ASTHandle) (ASTHandle, error)
func (m *ASTManager) MkFamilyApp(fid FamilyID, kind int32,
    parameters []ASTParameter, args []ASTHandle) (ASTHandle, error)
func (m *ASTManager) MkVar(index uint32, sort ASTHandle) (ASTHandle, error)
func (m *ASTManager) MkQuantifier(spec ASTQuantifierSpec) (ASTHandle, error)
func (m *ASTManager) IncRef(ASTHandle) error
func (m *ASTManager) DecRef(ASTHandle) error
func (m *ASTManager) GetKind(ASTHandle) (ASTKind, error)
func (m *ASTManager) GetASTID(ASTHandle) (uint32, error)
func (m *ASTManager) GetHash(ASTHandle) (uint32, error)
func (m *ASTManager) GetRefCount(ASTHandle) (uint32, error)
func (m *ASTManager) GetSort(ASTHandle) (ASTHandle, error)
func (m *ASTManager) GetArg(ASTHandle, uint32) (ASTHandle, error)
func (m *ASTManager) Stats() ASTManagerStats
func (m *ASTManager) Close() error
```

Slice arguments are borrowed only for the duration of the call, never stored.
Kind-specific getters cover every source scalar and indexed accessor; they
return scalars/checked handles, not arena record pointers. ASTQuantifierSpec
is a caller input containing forall, bound sorts/names, body, weight, qid,
skid, patterns and no-patterns. It is not the stored record. It does not invent
a simultaneous-pattern/no-pattern route rejected by the source constructor.
Nullable info tokens represent the source null-info distinction.

ASTParameter, ASTSortInfo and ASTFuncDeclInfo are scalar owner/slot/generation
tokens into concrete manager-owned typed records. Parameter constructors cover
int32, uint32-to-int32 source bit pattern, AST, symbol, rational, double and
external uint32 ID. Copy/Assign/Move/Del and scalar typed accessors are explicit.
Rational construction/copy uses the real RationalService; AST parameter
construction does not retain. Info constructors copy parameters as source
vectors do; unique node registration copies info again, then attaches its AST
parameters. Deleting unattached caller info does not call attachment del_eh.
Borrowed rational getters become explicitly named owned copies, with documented
additional Go boundary copy, never an exposed mutable raw numeral.

ASTRef and concrete ExprRef/SortRef/FuncDeclRef/AppRef/VarRef/QuantifierRef use
manager-issued scalar ownership tokens and explicit New/Copy/Assign/Reset/Del.
Copying a Go token is not acquiring another retain. Retire generations rather
than making a stale token valid again. Preserve source self-assignment and
retain-before-release ordering. No per-reference heap object is necessary.
Ref vectors are concrete typed storage with source null callback adaptation
explicitly recorded if required, not a generic ref_vector replacement.

## Storage, identity and charges

Use a concrete common node directory with named scalar fields: generation,
live state, kind, sourceID, hash, uint32 RC, mark1/mark2/shared flags, debug mark
owner identities and kind-record index. Five concrete pointer-free record
pages hold Sort, FuncDecl, App, Var and Quantifier fields. This avoids a boxed
node union or a large union of all five bodies. A temporary local record borrow
must be reacquired after any allocation or callback.

Sort records store Symbol plus info token. FuncDecl records store Symbol,
arity, range and domain payload descriptor. App records store declaration,
argument count/descriptor and source depth/ground/quantifier/label flags. Var
records store index/sort. Quantifier records store body, forall, weight, qid,
skid, all counts/descriptors, depth, unused-vars flag and labels flag. Name
payloads contain Symbol scalars; AST payloads contain compact internal refs.

Parameter pages contain named kind, int, internal AST ref, Symbol,
mpnum.OwnedRational, float64 and external-ID fields. OwnedRational is opaque to
g3 and is stored as its actual pointer-free type; do not serialize its private
fields into arbitrary byte offsets. Info pages contain family/kind, flags,
parameter-vector descriptor and SortSize or function flags. Real typed page
backings and source vector capacity storage must be charged, including any
union-layout inflation. Page directories are cold and pointerful, one entry
per page. No pointers, strings, slices, interfaces or functions appear in hot
node/parameter/persistent-array/chaining entries. Recursively audit the actual
imported record layouts on native and wasm targets.

Node origin/slot/generation is distinct from source ID. Source expression IDs
start at zero, declarations at 1<<31. Recycle/free-list behavior uses the
committed charged utilIDGen, including OOM skip and uint32 behavior. CompressIDs
later changes source IDs without moving public handles. Numeric ID caches in
the basic plugin must follow the source and cannot silently become slot caches.

Use actual arena.SmallBytes for variable scalar payloads with readable fixed
encoding, or concrete typed payload pages where checked ref width warrants it.
The implementation must state which layout it chooses before writing tests.
Charge actual target sizeof via audited reflection/layout constants, whole
allocated capacity, every owner/header/directory/free-list, retired ledger,
format manager, plugin/cache, scratch buffer and old+new growth peak. Never
charge fictitious C++ object bytes in addition to a Go record's actual storage.
Constructor fault tests sum the complete graph rather than assuming fixed
header bytes across architectures. Borrowed SymbolManager/RationalService
headers are excluded; storage they acquire for copied numerals is included in
the operation's full-service oracle.

## AST-specific source chaining table

Implement a separate concrete AST table specialization, reusing the audited
algorithm, not the int32 comparator. Entries contain only internal AST refs
and next indices. Key hash is node.hash; equality first tests matching hashes,
then source compare_nodes. Source pointer identity means equality of live
internal AST identities, not source IDs or an address cast. AST erase must use
identity even when structural equality would find another candidate.

Preserve bucket/cellar layout, modulo, growth before duplicate detection,
insert-at-head displacement, copy reversal, repeated cellar doubling, bucket
ascending/chain traversal, collision counters and failed-copy UsedSlots prefix.
Do not implement it as UintMap or Go map. Hashing is uint32 source arithmetic,
not raw struct-byte hashing: range is omitted from function hash but checked
by equality; quantifier hash uses declared-sort hash, pattern count and body.
Array hash consumes triples backwards; its 0/1/2/3-element branches remain
separate. Permanent expected vectors must cover every branch and overflow.

Defined structural quirks are mandatory fixtures: sort cardinality and private
parameters ignored; function idempotence ignored; null-info distinction;
quantifier qid/skid ignored; range checked despite hash omission. decl_info
compares using the LEFT parameter count without checking right count. Preserve
finite left-prefix equality in a forced hash collision; a right-shorter read
is undefined C++ and becomes a typed validation error, not a new symmetric
length rule. NaN equality and finite double hash conversion follow source;
out-of-range double-to-uint32 conversion is undefined and receives a typed
domain adapter. Function is_null ignores flat-associativity, kind and params.

## Publication and destruction actions

Registration allocates candidate/trailing payload, computes hash, inserts into
the real interner, then allocates source ID. Duplicate candidates free their
raw storage only: no info deep copy, child retains, or consumed ID. Unique
sort/function info is copied after insertion/ID, then init_eh runs parameters
ascending. Function retains domain ascending then range; app retains decl then
args ascending; var retains sort; quantifier retains declared sorts ascending,
body, patterns ascending, then no-patterns ascending. Derive app flags during
that source traversal. Constant app flags are the shared constant defaults;
nonconstant app depth saturates at 65535, quantifier depth does not use that cap.

Source C++ allocation exceptions after interner publication can leave an
incompletely constructed graph. The Go adapter must explicitly unwind only
new candidate ownership, identity-erase that candidate, release completed
attachments and clean copied info, without undoing source table growth,
collision/failed-copy counters, ID consumption or legitimate numeral scratch.
Tests must inspect these prefixes immediately. This adapter addresses an
invalid source continuation; it is not an assertion of whole-operation
transactionality. Parameter assignment deletes an old rational before copying
the new one, so failure must not restore that old rational silently.

Deletion uses source ptr_buffer inline16 and doubling overflow, lexical per
delete invocation. Remove identity first, recycle source ID next, release info
AST/external attachments ascending then destroy parameter vector ascending.
Info releases may recursively invoke deletion independently of the outer
worklist. Function queues domain ascending then range; app decl then args;
var sort; quantifier sorts/body/patterns/no-patterns. The queue pops LIFO.
Free node only after these actions. Do not replace this with a recursive child
walk or an arbitrary retained vector capacity schedule. Preflight charged ID
free-list cleanup capacity before a publication that requires it; document
each physical preflight adapter. Worklist reserve failure must leave an owned
continuation ledger or be prevented before destructive release, not abandon
RC-zero nodes. Exact buffer/failure policy is a review gate before coding.

Release IncRef wraps uint32; wrapping does not itself delete. DecRef at zero
is a source assertion domain and returns a typed error. In g3debug, preserve
the source preincrement UINT_MAX assertion and predecrement zero assertion;
g3checked alone is handle validation, not Z3DEBUG. Null manager Inc/Dec is a
source no-op. Debug contains/slow_not_contains/assertion routes need separate
conditional tests and method inventory; no claim that value equality tests
cover these actions.

Runtime debug_ref_count is separate from g3debug: it suppresses ID recycling
and info del_eh/deallocation, records free IDs in real UintMap, and frees node
storage. Retained info must have a charged cold retirement ledger. Close may
destroy retained numeric payloads as an explicit owner-teardown adapter, but
must not replay skipped AST/external callbacks against freed nodes. Until
these behaviors are implemented and fault-tested, that mode remains OPEN.

## Real eager plugins and constructor teardown

Families initialize in order basic, label, pattern, model-value, user-sort,
arith. Registering arith's family does NOT register an arithmetic plugin.
Preserve int_real_coercions=true initially; missing plugin factory returns
source nullptr. Unsupported arithmetic coercion continuations are a real
dependency gate, never a fake conversion or forced disabled flag.

Five concrete cold plugin objects implement real declaration/sort/value hooks.
The plugin registry may use cold interfaces as GO_PORT_PLAN permits; no hot
record stores one. Basic implements its complete header declaration dispatch,
Boolean/proof sorts, pinned Boolean declarations, undef proof declaration,
eq/oeq/ite caches by source sort ID, proof-arity caches and their real flags and
finalization order. Actual proof declaration variants are included even though
manager proof builders are a later packet. Label validates int/symbol params
and real Boolean domains; pattern accepts source argument sorts and returns
Boolean; model-value uses int+AST-sort parameters, private parameters and real
value uniqueness; user-sort registers names in source vector/map order and
copies them in mk_fresh. Base default no-op/false hooks are actual source
defaults, not invented callbacks.

Initialization order: nested format manager first; basic plugin pins Bool and
Proof sorts and declarations; manager pins Bool, Proof, undef app; register
remaining four plugins; manager pins true then false. Failure tests cover
every successful reserve prefix in both managers and prove completed roots,
numerals, pages, headers and owned streams are refunded.

Close drops manager Bool, Proof, true, false, undef pins in that order; finalizes
plugins in registry order, destroys plugins in that order, then resets registry.
Remaining interner roots are found by actual source child marking and table
traversal, forced to RC zero and deleted. Since registry is already reset,
external del_eh on leaked roots cannot find a plugin: preserve that source
fact. Close then destroys nested format manager and owned stream. Outstanding
public AST handles become invalid. Escaped expression-array refs need an
explicit teardown policy: proposed bulk invalidate/cleanup their physical
graph after node teardown without calling AST DecRef on freed nodes, matching
allocator teardown rather than inventing live arrays after manager Close.
This differs from standalone Int32PArrayManager.Close's live-cell guard and
requires an explicit integration test, not an implicit reuse of that Close.

## Concrete expression persistent arrays

ASTExprArray is a scalar owner/cell-generation/updates record. Implement all
manager array methods Mk/Del/Copy/Size/Empty/Get/Set/SetFrom/Push/PushFrom/
Pop/PopFrom/Unshare/Unfold/Reroot. Algorithms are the actual parray ROOT/SET/
PUSH_BACK/POP_BACK forward and reverse trails, with RC30, preserve_roots=true,
factor=2, max_trail_sz=16 for ast_manager. The disabled parray.cpp tst4 has a
separate max_trail_sz=8 specialization; do not claim it from the manager's 16.
Stored values retain/release through actual locked AST helpers. ROOT payloads
own all initialized expression refs; trails own values/links as source. The
documented inverse-PUSH idx defect remains, with its dedicated regression;
no copied-slice implementation can replace these actions. Warm updates must
plateau in real AST nodes and array cells, not just heap bytes after Close.

## Tests first and completion evidence

1. Port the two registered Boolean identity assertions and seven expression
   array assertions verbatim in operation routes (tst1/tst5 of ast.cpp). Its
   tst2–4 and alternate earlier tests are commented out, not active coverage.
2. Port parray.cpp active tst5: 100 actual Bool variables, copied array, exactly
   1,000,000 Set updates at i%100, actual AST-variable factories and Del(a2)
   before Del(a1). Replace platform libc rand only with a documented finite
   fixture stream in tests; no production RNG substitution. The test has zero
   ENSURE sites; add independent saved-version/value/RC and plateau assertions.
3. Before production: missing-API RED for all five kinds, six params, eager
   normal/format managers, plugin branches, actual owning wrappers and arrays.
   Independent serializer/oracle tests use bounded typed DAGs and a test-only
   map; frozen hash/collision fixtures do not call production hash twice.
4. Exercise every metadata quirk, app flags/depth, quantifier updates, source
   ID recycle separate from slots, stale/foreign handles, mark-owner checks,
   raw RC boundary without billions of operations, nested numeral lifetimes,
   duplicate candidates and source LIFO destruction order.
5. FailAfter prefixes inspect live cells, references, child RCs, ID state,
   tables, plugin pins and copied numerals BEFORE String/Del/Close. Only true
   source partial outputs/cached scratch are allowed. Audit actual layout,
   all physical budget bytes, allocator plateau and warm allocation counts.
6. Frozen snapshot gates: full native and g3checked+g3debug, targeted race,
   actual fresh checked/debug JS named RUN/PASS including million updates,
   vet, WASI build and no-generics syntax audit. Status lists each source
   method/conditional counterpart, not a whole ast.cpp completion label.

## Subsequent source-method packets, still required

| Packet | Remaining source surface and dependency |
|---|---|
| Core follow-through | All recognizer/accessor routes, six quantifier update overloads, fresh declarations, ID compression/compaction, sort substitution, resource hooks; record any that move into first packet explicitly. |
| Proof manager | Every proof builder and disabled-mode boundary in ast.cpp; preserve the three-argument mk_proof helper's count=2 defect, actual declarations/app DAG and source premise/fact checks. |
| Dependencies | dependency.h concrete expression leaf/join with real callbacks, RC30 and marks; persistent dependency arrays, linearization/contains; constructors already own genuine empty metadata. |
| Arithmetic/plugins | Real arithmetic declaration plugin and numeric/polynomial prerequisites; actual int/real coercion factories and sort checks, additional families/plugins. |
| Reference/marks | Remaining obj_ref/ref_vector/fast-ref specializations, fast/ref/scoped/sparse marks, mutable mark owner checks and retain order. |
| Translation | AST translation and actual plugin mk_fresh/external translation; copy constructor copies families/plugins after eager init, not a shallow owner clone. |
| Printing/trace | Source atom/metadata/low-level/bounded/SMT printers, printer-dependent exception/trace routes and owned-file/borrowed-stream overload gates. |

The coordinator approved private scalar tokens, actual typed page payloads,
charged intrusive deletion continuation metadata, escaped-array physical
teardown, and the explicit trace/dependency boundary before implementation.
These are source ownership obligations, not permission requests to the user.


## Current implementation evidence (not a freeze)

The observed initial missing-API RED used an isolated committed HEAD archive
with the actual source assertion routes. Native and checked/debug targeted
runs now pass the two Boolean plus seven array assertions and the exact
million-update workload. Its varying stream is utilhash.Random(seed1), an
explicit adaptation of external libc-rand test input, with all100 repeatedly
updated sites verified to change. No production RNG was replaced.

Typed pages are concrete64-record arrays. The common node directory points by
private kindIndex into compact independently reused Sort/FuncDecl/App/Var/
Quantifier pages. Every actual page and pointer-directory old/new overlap is
charged; pages are not priced as C++ node sizes. Parameter pages retain actual
mpnum.OwnedRational fields. Quantifier name storage uses the approved explicit
24-byte named-field Symbol codec, with source token widths and guarded offsets.
Variable refs and expression-array cells use explicit scalar encodings; no
representation reinterpretation or opaque numeric-token encoding is used.

The normal constructor creates a genuine nested format manager before its own
eager initialization. Both instantiate the five actual plugins, real Bool/
Proof sorts, declaration caches, undef/true/false applications and source pins.
The arithmetic family is registered without a fabricated arithmetic plugin.
Empty dependency owners have real borrowed-owner/storage fields and no fake
callable dependency algorithm. Trace streams/files remain in the follow-on
printing boundary.

| Selected source actions | Current concrete translation and evidence |
|---|---|
| parameter constructors/copy/move/assignment/destructor/hash/equality | ast_parameter.go; all six kinds, owned rational copies, uint32 hash domains (including -0.5), attachment RC fixtures |
| decl_info constructors/deep copies/init_eh/del_eh/hash/asymmetric equality | ast_info.go + register/delete; source push capacities and ascending parameter destruction; private/cardinality/idempotence quirks |
| all5 node constructors/hash/comparison and named scalar queries | ast_core_factories/query/storage + ast_table_hash; actual compiled source ast_array_hash permanent vectors for lengths0..10 |
| structural chained interner insert/find/identity erase/copy growth | ast_table.go; head displacement/copy reversal/counters/source failed-copy used prefix; specific collision/failure models still under review |
| register_node_core unique/duplicate lifecycle | actual pre/post source debug contains calls, optional full slow_not_contains traversal, ID preflight physical adapter, real deep info copies and child retains |
| delete_node | lexical inline16/doubling worklist; parent-first ID recycle then ascending child pushes/LIFO destruction; exact recycle fixtures pass normal and reserve-failed paths |
| obj_ref selected AST owning specialization | ast_ref.go; scalar owning record tokens, source assignment retains before releasing, move/swap/steal; remaining typed conveniences are follow-on inventory |
| ref_vector selected AST owning specialization | scalar encoded cold owners and entries, separately charged DirectOnly byte storage for source heap vector backing; reserve grows size; push and invalid-index set preserve completed INC failure prefix |
| expression parray actual expr_array_cfg | ast_expr_array*.go with real AST callbacks, RC30 and preserve_roots; all nine source assertions and actual million update routes |
| basic plugin set_manager, Boolean/proof declaration dispatch/caches/finalize | ast_basic_plugin*.go; source eager order, per-sort equality/ite caches, fixed/variable proof counts, parameterized proof routes and source finalizer order |
| label/pattern/modelvalue/usersort plugins | ast_builtin_plugins*.go; real validation/private parameters/names; pattern ignores kind/parameters; dictionary<int> cached hashes/null key/last tombstone retention; fresh-user replay rebuilds registration kinds |
| plugin values/default policies | source basic/model values, selected-plugin equality/distinctness, actual manager some-value fallback and uninterpreted sort predicate |
| default no-trace normal/format ctor/destructor | ast_core.go; source pins and plugin order, physical graph accounting; every predictable constructor reserve prefix audited before borrowed-owner Close |

Array plateau checks compare physical bytes/live AST nodes before final saved
version queries. With two100-element refs, strict update_counter>size permits
size+1 inverse trails plus two roots; live cell equality across different
phases would be an invalid oracle. The first final saved-version Get may grow
real reroot scratch and is charged separately. Source inverse-PUSH idx defect
is retained from the audited int32 algorithm, not silently corrected.

Failure adapters only cover source unsafe exception/dangling continuations:
charged intrusive metadata finishes exact deletion LIFO after worklist reserve
failure; consumed ref/array tokens are cleared or republished; copied numeral
and candidate ownership unwind before error publication. Ref-vector failed
pushes and index-guard set retain their source-completed RC increments instead
of inventing rollback. Retry increments again; vector Close consumes only real
entries; caller DecRef or source manager forced teardown can balance retained
nodes. All live numeric/root checks occur before String/Del/Close, so manager
teardown cannot hide temporary leaks.

Escaped expression arrays are physically invalidated after node teardown and
bulk-cleaned by the actual allocator, with no AST callbacks against freed
nodes. The graph-refund fixture subtracts the manager's full owned graph and
permits caller SymbolManager spelling/cache prefixes to remain owned.

The source g3debug is_pattern diagnostic executes before quantifier candidate
allocation and checks actual pattern family/kind AND every child being App;
no-patterns have no matching assertion. Release accepts valid non-pattern
expressions. RC increment wraps uint32 in release without implicit deletion;
g3debug guards the source max-counter assertion. Runtime debug_ref_count,
TRACE/printer events and the subsequent source packets above remain OPEN.
No whole ast.h/ast.cpp completion is claimed. Remaining core API/alias/
failure audits and final frozen native/checked-debug/race/actual-JS/vet/WASI
and no-generics gates are required before central integration.

### Additional source audit corrections and API inventory

The source `vector::resize` used by basic declaration caches expands one
physical capacity at a time (2,3,5,8,12). A behavioral RED demonstrated the
former direct jump to12. The concrete helper now retains every completed
backing expansion when the next reservation fails, while leaving logical size
unchanged. The model-value spelling buffer has a runtime warmed inline64
zero-Go-allocation fixture, and a second-overflow refusal fixture verifies
lexical cleanup immediately before numeric inspection or owner teardown.

`MkFuncDecl` maps the metadata reference overload, including its `is_null`
collapse. `MkFuncDeclWithInfo` maps the pointer overload and preserves a
nonzero metadata token even when `is_null` is true. The interner regression
keeps these identities distinct. `MkFamilyFuncDecl` maps the actual domain
overload dispatch for the five eager plugins; each ignores its optional range
argument in the source. Missing plugins return null. No additional plugin is
fabricated to satisfy this API.

`ParameterInt/AST/Symbol/Double/External` provide concrete checked scalar
getters. `ParameterRationalCopy` executes the source output-extraction
assignment only on a rational-kind match; its mismatch preserves the caller
numeral. The C++ const-reference getter is kept private rather than exporting
a second owning Go numeral token. Kind predicates derive from ParameterKind;
printer-backed parameter display remains in the printing packet.

The selected AST ref-vector specialization now includes construction from
borrowed scalar inputs, append overloads, default lookup, erase-by-value,
empty/back/contains, reset/finalize, copy/move/assignment/swap, reserve/resize/
fill/shrink/pop/reverse, and element assignment/borrowed iteration. Scalar
element and iterator views bind the actual payload allocation: growth or
finalization invalidates the borrow. Typed lifetime/bounds errors replace
undefined dangling-pointer dereferences. Null ordinary set remains supported;
element assignment preserves its separate source g3debug non-null assertion.
No per-element owner object, escaping slice, or boxed callback is allocated.
The other unmanaged/fast reference specializations and ref_vector_ptr_hash/eq
template utilities remain outside this selected specialization inventory.

The structural AST table has a direct permanent fixture for source displaced
head/cellar ordering, copy reversal, grow-before-duplicate refusal, identity
erase, collision counters and free-list reuse. It uses real Var records and
their comparator with bounded constant-hash collision inputs. Production
hashing remains the source algorithm covered by permanent compiled-reference
vectors.

The dictionary reset shrink-refusal route is explicitly adapted: source has
already deleted its old table before throwing, so its continuation is
undefined. Go retains completed FREE marking in owned old backing and zeros
logical counters before returning the allocation error. A behavioral RED
exposed stale nonzero counts over FREE entries; GREEN verifies immediate
accounting and coherent retry. This does not claim fidelity to a defined
source dangling-table continuation. Constructor borrowed-owner validation now
also checks the actual RationalService lifetime before any AST reservation;
closed symbol codec operations reject stale spelling owners.

Remaining whole-unit conditional surface is explicit: TRACE and printer-backed
DEBUG_CODE output, runtime debug_ref_count storage/deallocation mode,
parray diagnostic-only assertions beyond the checked structural/RC guards,
and printer-backed error text are OPEN. The complete manager proof methods,
dependency operations, marks, compaction, translation, arithmetic coercions,
additional plugin specializations and remaining convenience/update overloads
continue in their separately inventoried packets. Passing the nine active AST
assertions does not change those source-unit completion boundaries.

### Frozen selected first-core packet validation

The selected first runtime-core implementation is frozen for coordinator
review against committed HEAD9f8974d. The isolated snapshot is
`.cache/release/ast-core-final`, constructed from `HEAD:golang` plus only the
64 Go files under the explicitly owned core/parameter/info/table/ref/expr-array/
basic-plugin/builtin-plugin prefixes. Uncommitted coordinator proof files and
other agents' drafts are excluded. `AST_CORE_SHA256SUMS` in that snapshot
enumerates those frozen64 bytesets; its SHA256 is
`4aec354d9486698967efd7b2c127d6541c440c17079f6812ca22b01302700f10`.

Full module native and full `g3checked g3debug` tests pass against the frozen
code. All45 named AST groups execute and pass under checked/debug race and
actual Node JavaScript/Wasm (`-count=1 -v`); the million-update source route
passes in16.97s under race and5.55s under JavaScript. The final logs are
`native.log`, `checked-debug.log`, `race.log`, `js.log` in the snapshot.
Checked/debug full-module vet and WASI build pass. The portcheck command's
Go AST no-generics scan and unchanged baseline manifest validation pass.
Compiler/reference commands and test temporary artifacts use cache/temp paths
strictly below golang; no C++ reference dependency is needed to rerun tests.

The last borrow regression observed behavioral RED for an element_ref after
moving its vector and destroying the moved-from object. Source binds the
physical payload, which remains live under the new vector. The corrected
scalar borrow therefore validates AST origin and payload generation, not the
original vector record identity. Element/iterator borrows survive move/swap,
and growth/finalization invalidate freed allocations. The final45-group race
and JavaScript runs cover these corrected bytes. This freeze does not upgrade
the explicitly OPEN whole-source-unit surfaces above to complete.

### Ref-vector retained-slot correction

The subsequent source review of util/ref_vector.h and its ptr_vector backing
found that logical size was incorrectly treated as a universal release bound.
Three new behavioral regressions first failed against the original implementation
(release50748). An isolated original-core checked/debug run17211 also failed
those regressions and the corrected existing SetIndexGuard expectation. The
corrected targeted release56849 and checked/debug11519 suites passed. Two further
immediately-GREEN regressions verify real growth/failure accounting and the
freed-node dereference boundary; these added expectations are not invented source
assertions or a claim to port new original test registrations.

Source vector indexing asserts idx<size only in debug. Direct Get/Set and new
element creation retain that source-site debug ErrASTDomain; release accesses
may reach previously initialized retained capacity after shrink/pop/erase/reset.
Set still INCs the incoming node before indexing or DECing the old node, including
the retained INC on a debug index error. Shrink above the old size performs no
reference callbacks because dec_range(begin+newSize,end) is empty; release
publishes the larger size without null construction when it remains within
initialized physical capacity. Debug reports the original shrink assertion.
Resize remains the separate source default-construction route. Erase past size
has no valid source forward-loop continuation and is guarded after reading the
old pointer, before shifts/DEC. Empty Back/Pop and actual physical OOB remain
typed concrete memory-domain guards.

The cold owner record remains16 bytes (astRefVectorRecordBytes). Its backing now
contains an actual charged12-byte header: capacity, size and initialized high-
water. The last word is a Go representation adapter that distinguishes
never-constructed pointer slots from Go allocator zeros. Entries remain two
scalar32-bit words and requested source capacity growth is unchanged. Both the
source8-byte-header capacity domain and actual12-byte uint32 allocator domain
are checked; Go charges the physical12+8*capacity allocation and real old/new
overlap, not a fictitious source header. Move/swap transfer the metadata with
the backing; growth preserves all old-capacity bytes and initialization history,
including completed allocation prefixes before a later injected failure.

Existing element_ref and iterator borrows bind physical storage. Their later
reads do not repeat an operator[] assertion against current logical size: they
may read retained initialized pointers while their backing remains live, even
in debug. Growth/finalization still invalidate the actual payload generation.
Retained pointer bits do not acquire new ownership. The separately pinned-node
regression tests valid source AST lifetimes; a separate freed-node test shows a
raw retained token can still be copied, while GetRefCount/dereference and node
reference callbacks reject freed storage with ErrASTHandle. This scalar-token
lifetime adapter does not promise valid C++ dereferencing of freed objects.

Final isolated tree `.cache/release/ast-ref-vector-final147` uses HEAD147c741,
the previous frozen64 core files, the coordinator's two frozen quantifier-update
files, and the new retained-vector test:67 Go files. Proof/SAT drafts are excluded.
The correction owns only ast_ref_vector.go, ast_ref_vector_access.go,
ast_ref_vector_ops.go, ast_ref_vector_borrow.go, ast_ref_vector_test.go, new
ast_ref_vector_retained_test.go and this appendix. The generic-helper and whole
AST source-unit obligations previously marked OPEN remain OPEN.

| Corrected core frozen-byte gate | Terminal evidence |
| --- | --- |
| Full native ./... | PASS30085, root2.519s/mpnum22.237s |
| Full g3checked+g3debug ./... | PASS13767, root2.512s/mpnum22.907s |
| Named52 actual checked/debug race AST groups | PASS45650,52 RUN/PASS,18.837s; full million-update route17.55s |
| Named52 actual checked/debug JS AST groups | PASS68805,52 RUN/PASS,5.842s |
| New5 release JS retained/domain/failure groups | PASS85586,5 RUN/PASS,0.159s |
| Full checked/debug vet | PASS26931 |
| All-package checked/debug WASI cross-build | PASS4680, unique artifacts under golang/.cache/wasi-ast-ref-vector |
| Source/no-generics verification | PASS72238,2102units677APIs33ported68partial2001unported; central AST row updates remain coordinator-owned |

Named logs are golang/.cache/ast-ref-vector-race.log and
golang/.cache/ast-ref-vector-js.log. Added tests are
TestASTRefVectorSourceRetainedNullShrink,
TestASTRefVectorSourceRetainedPointerAndBorrow,
TestASTRefVectorNeverInitializedPhysicalSlot,
TestASTRefVectorInitializedGrowthFailurePrefix and
TestASTRefVectorRetainedFreedHandleBoundary. They supplement the previous45
groups and two coordinator quantifier-update groups; they do not increase the
claimed original source assertion inventory.

Frozen correction SHA256 (all six Go files at the golang root):

```text
dbe739ac20a3febd181ceec1bd6f845f80fedeb87b2fffd2bc3aee514133f3ce ast_ref_vector.go
414c7beeb9591ddf5e6f7db4bdcd46bb254193df5e5d4e59668195deeeb0c936 ast_ref_vector_access.go
5177c68e01ad87f8309c6aade0f74d8a0ee01f5c2cb29ca0e530f54301bbd0af ast_ref_vector_ops.go
15fc7b736d931e900eb7ecd7ce8ace6f1cfed3f5d0d910696092f63aedb4cd5d ast_ref_vector_borrow.go
1c251f095e73c23b9aa1a5829222cd327241f5dcae01dffa59d201712aee3117 ast_ref_vector_test.go
a38f260cfbae6ef4abe7c861af7267780ff7677aa8341b9778f7cba36f283c2c ast_ref_vector_retained_test.go
```

All owned gate handles are terminal; corrected Go files and this appendix are
frozen for coordinator review. All writes/build/cache/temp artifacts stayed
under golang. No generics, unsafe, proof/SAT edits, Git writes or central metadata
writes were introduced by this correction.

### Coordinator integration scope

The integrated packet is the corrected67-file/52-test freeze above, including
the six quantifier-update overloads in ast_core_update.go and its tests. These
two files first produced missing-API RED, then passed; independent source review
verified ast.cpp2381–2503 identity returns, replacement/retention of patterns and
no-patterns, kind/body/weight changes and retained names/sorts/qid/skid. Payload
copying uses actual candidate storage with no intermediate handle slices. The
weight-update TRACE and constructor printer/trace events remain open.

The coordinator updated existing AST/ref/container source rows conservatively
and moved src/test/ast.cpp from unported to partial for its actual nine active
ENSURE actions. Conditional sizeof TRACE remains unported. Whole-unit audit
flags remain false and no solver API is relabeled. Metadata changes are separate
from the frozen Go test gates; their source/evidence audit is recorded by the
coordinator. Proof and SAT drafts remain outside this core integration.
