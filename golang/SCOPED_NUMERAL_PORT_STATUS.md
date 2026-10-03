# Concrete scoped numerals and vectors

Owned only mpnum/scoped_numeral.go, mpnum/scoped_numeral_test.go and this report.
The selected IntegerManager/Int and RationalManager/Rational specializations are
implemented. Generic source headers remain PARTIAL for their other managers
(floating, binary-rational, polynomial and other numeral specializations).
Source vector copy construction is private and undefined, so these concrete
vectors intentionally expose no Clone or Assign. No generic Go code was added.

## Source audit and selected method mapping

Read complete src/util/scoped_numeral.h and scoped_numeral_vector.h, their mpz.h
inclusion/typedefs, mpz/mpq constructor/move/swap semantics, vector expansion,
copy/destruction/reset/push/pop/shrink/resize implementation, and active
src/test/mpz.cpp::tst_scoped. The active test has ZERO ENSURE assertions and THREE
print diagnostics; the translated test verifies all three exact lines, including
the independent product102951963583964173000063.

| scoped_numeral.h declaration | Both concrete scalar types |
| --- | --- |
| constructor/copy constructor/destructor | NewScopedInt/NewScopedRational, Clone, Close |
| m() | Manager |
| const/mutable conversions and get() overloads | Get: checked borrowed inline numeral pointer; Go has no const overload |
| assign scoped/numeral/int | Assign, Set, SetInt64 (source int32 domain plus manager's int64 convenience setter) |
| reset | Reset |
| both swap overloads | Swap and SwapValue |
| +=, -=, *=, /=, %= | AddAssign/SubAssign/MulAssign/DivAssign/RemAssign |
| ==, !=, <, >, <=, >= | Equal/NotEqual/Less/Greater/LessEqual/GreaterEqual |
| all five member and free sign predicates | IsZero/IsPos/IsNeg/IsNonpos/IsNonneg |
| abs-copy / neg | AbsCopy / Neg |
| copy-result +, -, *, / | Add/Sub/Mul/Div: Clone then source in-place manager operation |
| ostream forwarding | Display via existing manager String formatting |

All42 source scalar declarations (including overloads and equivalent free/member
predicates) were audited and mapped for these two managers. Arithmetic delegates
to existing manager methods; no new arithmetic algorithm appears in this packet.
The needed RationalManager.Rem dependency is supplied by the separately owned
mixed-MPQ packet. Its integer-valued precondition/alias behavior is preserved;
noninteger remainder errors are tested. Scalar wrappers have no source move
constructor: explicit manager Move remains available through borrowed Get, while
source wrapper copy/result operations deep-copy.

| scoped_numeral_vector.h | Concrete vector behavior |
| --- | --- |
| :27 private undefined copy constructor | noncopyable; Go struct-copy use rejected |
| :29 constructor / :30 destructor | NewScopedIntVector/NewScopedRationalVector / Close |
| :34 reset | ascending manager.Del for every live element, retain capacity |
| :42 m() | Manager |
| :44 push_back | Push grows backing, appends default numeral, then manager.Copy |
| :49 pop_back | Pop delegates to checked Shrink(size-1) |
| :53 shrink | Shrink deletes truncated numerals in ascending index, then changes length |
| :62 resize | Resize shrinks first when applicable; grows through source capacities, constructs numeral(0) defaults |
| inherited indexed access/size/capacity | Read deep caller copy, Set deep manager assignment, Size, Capacity |

Source vector expansion starts cap2, then (3*cap+1)/2 at each step. Int and
Rational source types have explicit move constructors, so source nontrivial move
relocation applies: only active elements move, numeric allocation identity stays
stable, moved-from cell references become empty and m_val survives. The concrete
code transfers pointer-free scalar records and clears only old cell references;
it does not delete/re-copy their payloads. No internal vector slice or element
pointer escapes. Copy-at-read/indexed-deep-set replace raw reference access to
avoid shallow owning-handle copies; unrestricted inherited raw vector operations
are not advertised as safe domain APIs.

## Lifetime, budget and failure policy

Each object borrows one live manager and charges its actual Go owner header to
that manager's existing Budget. Backing arrays charge actual concrete record size;
len/cap metadata already resides in the charged header, with no fictitious second
source counter-header charge. Manager cells/Words charge numeral payloads exactly
once; borrowed manager/Budget objects are not duplicated. Headers include a cold
self-identity pointer that rejects shallow Go wrapper/vector struct copies before
cleanup. Hot Int/Rational backing is pointer-free (reflect audit tests).
All objects use the existing owner-confinement contract, not per-record locks or
callbacks. Scalar Get is the source-style borrowed mutable reference to a cold
inline numeral; it must not be shallow-copied as an owning value and expires at
Close. Clone/Set/Assign are the deep ownership operations. Foreign scoped owners
are rejected. Caller read outputs/format strings remain caller-owned.

Close deletes scalar/vector payloads before refunding only its own header/backing;
it is idempotent. Managers must outlive scoped owners. If that source lifetime
precondition is violated, methods report ErrClosed; Close refunds independent cold
storage without deleting already-invalidated numeral cells and reports ErrClosed.
The manager already destroyed those cells at its own Close. This checked cleanup
adaptation avoids double deletion and leaked independent owner headers.

The root-approved source action-prefix policies are explicit:

- Failed Push deep-copy leaves the published default0 element (Rational den1).
  Failed backing growth leaves original length/capacity/payloads.
- Resize performs each individual source growth reservation. If a later growth
  fails, successful earlier capacity remains charged while length/values stay
  unchanged. Default construction happens only after growth completes.
- Old and newly prepared backing are simultaneously charged before transferring
  elements and releasing old backing. Exact one-byte-below/at-peak tests pass.
- Clone constructor failure releases its new header/partial scalar value; manager
  storage capacity grown earlier may legitimately remain, but immediate live cells
  and Words payload counts must return to baseline.

Invalid indices, empty Pop, too-large Shrink and invalid input numerals are checked
before source actions. Budget successful reservation attempts consume FailAfter
prefixes normally. No transactional controller bypass or hidden generic template
was added. A test's first fault fixture was corrected when parsing had already
warmed free cells: the final fixture occupies actual free slots to guarantee the
next deep copy reserves storage. Constructor peaks are separated from exact vector
backing peaks with actual charged caller-owned fixture backing, not fake owner
headers. Warm retained scalar/vector operations measure zero Go allocations.

## TDD and final gates

Source/lifecycle tests FIRST observed missing concrete API compile RED. Eleven
finite tests now cover source prints, scalar operators/copy/swap/owner guards,
Int1000-op independent vector oracle, Rational400-op independent vector oracle in
both scratch modes, denominator defaults/deep read independence, source capacity
prefixes, live-cell/payload balance, moved allocation identity, ascending cleanup
via third/second/first allocation-ref reuse, immediate failed Push Cells+Words+Used
baselines,24-prefix deep-Clone failure sweep with success/failure endpoints, exact
backing peak, invalid lifetimes/indices, pointer-free records, and zero warmed
allocation/retained-capacity behavior. Tests require no reference checkout/run.

Frozen production/tests validated:

- Full CGO-disabled go test ./... native session23835: PASS.
- Full g3checked native ./... session83050: PASS.
- g3checked native race TestScoped session1920: PASS.
- go vet ./mpnum: PASS.
- g3checked JS/wasm mpnum test build and WASI mpnum test build: PASS.
- Actual checked Node JS printed all eleven TestScoped names, all PASS.
- go run ./cmd/portcheck session29710: PASS, including no-generics syntax audit.
- Normal compiler environment; minimal Node-only runtime retained TMPDIR. All
  cache/temp/output stayed under golang/.cache.
- Coordinator's final full checked-race module session11885: PASS.
- Coordinator's final full actual checked JS module session68829: PASS; all33 new
  scalar/display/mixed/scoped tests had named RUN/PASS evidence.
- Final combined checked vet, WASI cross-build and source/no-generics inventory:
  PASS. WASI remains compilation evidence only.

No existing implementation files, manifests, global progress files or git metadata
were edited. Other manager specializations, arbitrary generic vector operations,
parser/AST/solver consumers and whole numeric/solver completion remain separate.

Source SHA256:

- scoped_numeral.h: e36703e7acdf210c3246844a06f9a2e8515f4c377dce2b1d07258460c3e90867
- scoped_numeral_vector.h: b8b591945603b19e5d50dff21f92821f60bf866ff07f7ef3687f01a5cfd64a29
- vector.h: aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e
- mpz.h: 41891597e12173581e7ae3f108c048897761c48a191b06fd3ba160d1e98049b1
- test/mpz.cpp: 183aa746a766abe4d7f3e2c908dfe3836f4534410f54152de62a5103582dff29
