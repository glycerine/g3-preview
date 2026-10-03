# Parameters reference and backing owner

Source: all `params_ref` methods in `src/util/params.h/.cpp` and all private
`params` methods/macros in `params.cpp`, 32-bit unsigned/signed casts. The
parameter-descriptor implementation is an existing dependency, not duplicated
or modified by this packet. Normalization remains in `util_param_name.go`.

## Source method coverage

`ParamsContext.NewValue` supplies the source default/null reference;
`NewValue` followed by `Assign` supplies its copy constructor. `Assign` performs
increment-new before decrement-old, including self assignment. `Del` supplies
reference destruction. `Copy` and `Append` implement sharing for a null
backing and otherwise initialization followed by ordered-entry merge.
`init`/`copyCore` preserve new-block publication before ordered detach copying.
Self copy resolves the source after initialization and traverses actual entry
identities, retaining rational self-assignment behavior.

`Empty`, `Contains`/`ContainsName`, `Reset`, and
`ResetKey`/`ResetKeyName` cover every source query/reset overload. Reset and
validation intentionally never detach shared backing. Six getter groups
(`GetBool`, `GetUInt`, `GetDouble`, `GetString`, `GetRat`, `GetSymbol`) each
have Symbol and nullable Name overloads; the five source fallback getter
families have `Get*FallbackName` overloads. First key **and kind** wins;
wrong-kind entries do not block later duplicates or fallback. Rational getters
accept unsigned entries with the source signed-int32 conversion. Returned
rationals are independent owned copies, including defaults.

Six setter groups (`SetBool`, `SetUInt`, `SetDouble`, `SetString`, `SetRat`,
`SetSymbol`) each have Symbol and nullable Name overloads. Source initializer
runs before Name key construction. Existing numeral assignments retain their
token; replacing another kind publishes the numeral tag and default initialized
value before assignment. New numerals are initialized and copied before vector
publication. Numerical symbol values violate source `bare_str`'s precondition
and receive a typed checked guard before graph mutation; numerical keys remain
valid. Null symbol values remain valid and display as `null`.

`Validate` applies descriptor module rewriting directly to stored keys, then
checks source kind rules. Earlier mutations and duplicate keys survive a later
error. Messages are exact `DefaultException` text, including sorted descriptor
names transformed only after sorting, defaults whenever present, and descriptor
nonnull-description preconditions even when descriptions are not printed.
Validation holds the descriptor owner lock throughout and reads its existing
payloads directly; it creates no intermediate descriptor-output string.

`Display`, `DisplayKey`/`DisplayKeyName`, `DisplaySMT2` and their caller-buffer
`FormatInto`/`FormatKeyInto`/`FormatSMT2Into` adapters cover source ostream
branches. DisplayKeyName interns a temporary symbol even for null backing,
unlike read getter/contains/reset paths. `DebugString` is the source Z3DEBUG
`pp` text boundary including its newline, available as an explicit diagnostic
adapter instead of a process-wide stream side effect.

## Ownership and storage adaptations

Symbols and numerals borrow explicitly supplied services that must outlive the
context. Their actual budget can differ from this context's budget. Neither
borrowed header nor numeral storage is charged fictitiously to context storage.
All ParamsRef, backing blocks, entry payloads, text refs, and entry allocation
metadata are pointer-free. No generics, unsafe, raw exported manager borrows,
per-object slices, or per-entry interface values are used.

A single cold context header owns the concrete block registry, typed pooled
entry pages, their metadata/directory, Words text pool, and formatting workspace.
Each actual array and owner is charged exactly, including old/new growth peaks.
Requested source svector capacity is 2 then `(3n+1)/2`; physical entry blocks
use geometric classes. Free physical pages remain context-owned until Close;
Reset frees the logical vector allocation and capacity. This is an allocator
adaptation, not a claim of identical source allocation timing. Generation and
reference-count overflow refuse before wrap. NewValue has no backing allocation.
Raw zero ParamsRef is invalid; the explicit constructor creates an owned null
token. Shallow token copying is forbidden; Assign is the owning copy operation.

Rational entries retain actual scalar OwnedRational tokens inline rather than
source separately heap-allocated rational objects. The actual entry width and
numerical service storage are accounted; no imaginary rational heap-node charge
is added. Growth transfers tokens and invalidates old slots after publication;
ResetKey shifts ownership and clears the vacated tail without a second Del.
Unpublished new values are destroyed on allocation failure. A detach failure
retains its published partial new block but decrements the old reference even
on error; source exceptions would skip that decrement and leak. Existing
RationalService staged allocation semantics are inherited and explicit.

Borrowed source C-string values become owned immutable first-NUL text, preserving
null versus present-empty. Owned-text replacement prepares its allocation before
removing the old value, an extra predictable failure adaptation. Symbol values
remain borrowed scalar symbols. Name reads use `SymbolManager.EqualString` and
never eager interning. Provenance guards precede source graph actions.

Formatting uses budgeted retained byte storage and stack-bounded numeric
conversion (`g`, precision 6, classic decimal/scientific spelling, signed zero,
inf/nan). Locale and custom ostream flags are not emulated. A final Go string
is caller-owned escaped output; its lifetime is not falsely arena-accounted.
Caller buffers exclude NUL and stay unchanged on short buffer or error.
For null **C-string** stream insertion, this adapter models the locally audited
libstdc++ badbit behavior: Display returns the defined prefix together with
`ErrParamsStreamFailure`, suppressing subsequent text; FormatInto leaves the
caller buffer unchanged. libc++ can instead dereference null through
char_traits::length: this is an explicit libstdc++-mode/Go adaptation, not a
universal source-language claim. A null module fails only when a matching entry
actually attempts insertion. Null SYMBOL rendering is a different valid case.

The source process-global `g_empty_params_ref` becomes explicit owner-local null
values; no process-global symbol/numerical owner is introduced. Cross-context
ParamsRef sharing is rejected. This utility packet does not claim AST or solver
consumer behavior is implemented.

## Tests and verification

No standalone params test or registration exists in `src/test`. Existing solver
setups using params_ref do not supply isolated params assertions and are not
counted as ported unit assertions. Eighteen permanent finite tests derive exact
source action traces and independent scalar/cast expectations: sharing and
reset, ordered copy/append, duplicate validation prefixes, all getter/setter and
fallback families, exact messages/displays, nullable inputs, read noninterning,
capacity progression, generation/provenance, host C-prefix length narrowing,
self-copy identity, cold headers,
whole-context accounting, old/new peaks, and warm zero allocations.

Initial observed RED was missing ParamsContext/NewParamsContext/ParamsConfig.
A later meaningful self-copy RED exposed loss of source entry identity; fixed
by traversing actual entries and resolving self source after init. Eight
reservation-prefix operation sweeps continue until success, exercise real cold
or wide storage growth, and audit live numeric cells, Words text, reference
roots, and unused entry ownership **before** rendering or teardown. Constructor
reservation prefixes unwind to the borrowed-owner baseline; every final graph
Close returns budget usage to zero.

Verification passed on frozen eighteen-test bytes: full native and actual
checked modules, targeted checked
race registrations, actual checked-JS eighteen RUN/PASS registrations, vet,
WASI build and no-generics audit. Parent integration independently passed full
native, full checked-debug, targeted checked-debug race, vet, WASI build and
actual checked-debug JS for all eighteen registrations. The Go files matched
snapshot hashes after these gates. Source fingerprint/ledger checked separately.
