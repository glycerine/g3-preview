# ParamDescrs concrete port

Final parent integration passed the full native and checked debug suites,
checked debug race tests for all thirteen descriptor groups, checked debug vet,
and WASI compilation. Actual checked debug JS/Wasm passed every package with
all thirteen groups explicitly RUN/PASS in
`.cache/inf-eps-descr-integrated-final-js.log`. The production/test bytes matched
the verified snapshot before commit. Final ledger integration passed checked
root/internal-portcheck tests and source/no-generics audits: 24 ported,
44 partial, 2034 unported of 2102 units, with 677 API operations.
Full params_ref and command runtime implementations remain open.

Scope: all `param_descrs` declarations in `src/util/params.h:114-145`, its
implementation/imp in `src/util/params.cpp:44-308`, and the complete
`cmd_arg_kind` numeric ordering and label switch. The rest of params.h/.cpp,
command classes in cmd_context_types.h, generated parameter collectors, and
`params_ref` are separate units. Name normalization belongs to the parent
`util_param_name.go` packet; descriptor name wrappers do not normalize.

No standalone active descriptor assertion test exists under src/test. The
registration consumer `src/test/gparams_register_modules.cpp` instantiates
collectors/owners but contains no direct descriptor ENSURE assertion to port.
Therefore the source assertion count here is **0 active test assertions**.
Three implementation SASSERT obligations are exercised explicitly: nonnumeric
insert, same-kind duplicate, and nonnull description even when display excludes
its text. Independent bounded checks and exact source fixtures are additional
Go tests, not invented source assertion counts.

## Method mapping

| Source | Concrete API/implementation |
| --- | --- |
| params.cpp:44-69 imp::info constructors | pointer-free paramDescrInfo, ParamInvalid/missing defaults and nullable scalar text refs |
| params.cpp:74-83 imp::insert | prepareInsert/insert/publishInsert, Insert/InsertName |
| params.cpp:85-98 erase/contains/get_kind | Erase/EraseName, Contains/ContainsName, Kind/KindName |
| params.cpp:100-110 split_name | splitName, checked first-dot scan, prefix character vector push sequence |
| params.cpp:112-127 get_kind_in_module | KindInModule, direct lookup first, suffix lookup and exact module interning before name mutation |
| params.cpp:129-148 module/descr/default | Module, Description/DescriptionName, Default/DefaultName; ModuleName is a convenience adapter |
| params.cpp:150-159 size/get_param_name/lt | Size history length, ParamName, bytewise symbolCompare |
| params.cpp:162-197 display | Display, original-name sorting before label transformations, exact indentation/kind/default/newline output |
| params.cpp:199-205 copy | Copy, merge in ascending USED bucket order, existing metadata retained, successful prefixes retained |
| params.cpp:209-214 constructor/destructor | NewParamDescrs/Close, owned graph reserve/unwind/release |
| params.cpp:216-288 public forwarders | corresponding methods above, char-name wrappers explicitly Intern |
| params.cpp:290-308 five insert helpers | InsertMaxMemory/MaxSteps/ProduceModels/ProduceProofs/Timeout, exact descriptions and nil actual default/module |
| cmd_context_types.h:33-45 enum | ParamKind constants 0..19 |
| cmd_context_types.cpp:21-47 label switch | ParamKind.String, unknown fallback including ParamInvalid |
| dictionary.h:24-28; map.h cached default_map_entry | private paramDescrEntry cached hash, explicit FREE/DELETED/USED state, null key supported |
| hashtable.h:194-242 move/expand/remove_deleted | rehash old USED buckets in physical order, exact doubled/same capacity |
| hashtable.h:367-407 insertion probe/grow | strict preinsert >75%, wraparound, **last** encountered deleted bucket reused |
| hashtable.h:492-523 find; 545-579 remove | cached-hash equality, FREE terminates, next-FREE erase vs tombstone; cleanup only after tombstone when deleted>size and deleted>64 |
| vector.h:60-105 expansion; 174-179 sized-data constructor | names/display/prefix capacities 2 then (3old+1)/2; prefix constructor pushes every character before terminator |

## Ownership, failure, and storage adaptations

One SymbolManager is borrowed; it must outlive descriptor owners and must not
be torn down concurrently with operations. Foreign/marked symbols, numerical
insertion, duplicate-kind mismatch, missing-null split/display, null description
display, and invalid index receive typed errors rather than source assertion or
undefined pointer operations. Numerical query does not intern and returns absent.
Null is a valid ordinary dictionary key with exact hash **0x9e3779d9**.
A cold self-identity check rejects shallow-copied owners. Cold mutexes serialize
an owner; Copy serializes multiowner locking with one cold global mutex. Borrowed
Budget/callback environments remain external and caller-serialized if shared.

Source info borrows immutable C pointers. The Go ownership adapter copies each
nonnull C string (through its first NUL) into private arena.Words packed words.
Nil is absent; empty is a live one-word NUL payload. Entry/name records contain
only scalar Symbols, cached hash/state, kind, and Words refs/lengths. Metadata is
recycled on erase; inactive bucket scalar payload is retained like the source.
Copy directly reads source Words and writes destination Words, without loaded
intermediate strings. Each destination payload is charged while source payload
remains live; destination survives source Close while both still borrow the same
symbol manager. Caller-returned text copies survive erase/close.

Table/header, names capacities, Words owner/descriptors/pages/metadata/directories
are charged exactly once. New array reserve precedes allocate/copy; old array
charges release only after publication. Constructor failure unwinds nested
owners and all earlier charges. Nil Budget permits unlimited storage; explicit
Text.Descriptors.Budget must equal the owner Budget (or default to it).
Default table capacity8; configurable power of two. Default text page256 words,
descriptor page2 are storage adaptations, not source container capacities.
Source names capacity/history never shrink on erase/reinsert, intentionally;
metadata page storage independently plateaus with scalar slot generations.

**Source dictionary publication precedes names.push_back.** If names growth
fails, the live dictionary entry and its metadata remain published while history
length stays old. Retry duplicate returns and never appends that omitted history
entry. Metadata adapter failures before dictionary publication release all newly
live refs; retained Words/table capacities may remain. Copy retains completed
insert prefixes, including this final published entry without history. Erase
cleanup reservation failure keeps the completed erasure and old tombstones.
OutOfMemory's cold diagnostic skips source cleanup when true.

Display charges the actual temporary Symbol vector with source growth and old/new
peaks; concrete in-place heapsort realizes the standard std::sort ordering/output
contract without a reflection swapper or second work array. Comparator, name, and
metadata output read checked bytes directly; only final output storage is
caller-owned, like the source external ostream. KindInModule charges every
actual prefix char vector growth (constructor loops push_back, then pushes NUL),
and explicit Go byte-array/string bridges for suffix/module interning. Prefix
scratch is destroyed after split returns, before suffix kind/module lookup.
All such temporary charges release on every exit. Borrowed-manager successful
intern prefixes persist if a later descriptor operation fails.

## Tests and evidence

Initial six source/model groups were written before implementation and observed
RED with missing ParamDescrs/ParamKind API. A meaningful additional source-order
RED showed dictionary missing after names reserve failure; fixed publication
order. A second meaningful source-vector RED showed 'four.key' split incorrectly
succeeding at FailAfter5 because constructor growth was collapsed; fixed exact
2→3→5 prefix push sequence and six total source/adapter reservations.

Thirteen registered groups:

- TestParamDescrsSourceHistoryDefaultsAndDisplay
- TestParamDescrsSourceModuleAndNullKinds
- TestParamDescrsSourceKindLabelsAndHelpers
- TestParamDescrsCopyBucketOrderLifetimeAndPrefix
- TestParamDescrsIndependentDictionaryStateMachine (1800 bounded operations)
- TestParamDescrsBudgetFailuresAndWarmDuplicate (zero warmed duplicate Go allocations)
- TestParamDescrsSourceProbeThresholdAndCleanup (wrap, last deleted, strict load, normal/OOM/failure cleanup)
- TestParamDescrsOwnershipNullableMetadataAndGuards
- TestParamDescrsCopyDisplayFaultPrefixesAndPlateau (90 merge prefixes, immediate complete graph accounting)
- TestParamDescrsSourceNamesFailureKeepsDictionaryPublication
- TestParamDescrsModuleScratchFailuresAndMetadataPublication
- TestParamDescrsSourcePrefixConstructorUsesPushGrowth
- TestParamDescrsCheckedCapacityBoundaries

Final targeted native all12 RUN/PASS. Native full module passed before the final
prefix growth correction; final refreshed checked full module, targeted checked
race, actual checked JS, vet/WASI, and no-generics results appended below once
completed. No source reference checkout is needed at test runtime.

## Source SHA256

```
50738b66c08a5d24f818eedf5fa4bafdaec8593b41408ae1173c78adc762c6fb  src/util/params.h
adfec5df1efec4cd4d08844d06463b58096d1c91e0374e7e432a5f1d9bb70dd9  src/util/params.cpp
09356c28b4a280a7cf48d43e724396cbd3162db66949df033aebc5ad2bb5fbe9  src/util/cmd_context_types.h
3dd955e878614004a06cca24f82c0f1a24ca33892869300abcb82fc08d9bc64f  src/util/cmd_context_types.cpp
defa2be995dc90b70c12342c23374a55159ce9fabde29ac83c76ddf2661b6289  src/util/dictionary.h
62db3b3819aa71f67bababdb3d5583d14fecebbde6f2d22866c28a86215aa850  src/util/map.h
bdc562a7db2c0d914dd896577c43e0648f501538873460d2e1a6c2d0932f64f2  src/util/hashtable.h
aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e  src/util/vector.h
```

Final refreshed gates after the prefix constructor correction:

- Native targeted `-count=1 -v`: all12 RUN/PASS (60f073).
- `-tags=g3checked -race -run '^TestParamDescrs' -count=1 -v`: all12 RUN/PASS, exit0 (26564).
- Checked full module `go test -tags=g3checked ./... -count=1`: exit0 (24970).
- Fresh checked JS compilation then minimal Node actual execution: all12 explicit RUN/PASS, exit0 (901247).
- Final `go vet ./...`: exit0 (55c4f9).
- Final checked wasip1/wasm root test binary cross-compile: exit0 (8e1a42).
- `go run ./cmd/portcheck`: exit0, 2102 units/677 APIs, 21 ported/42 partial/2039 unported (91401); syntax audit includes production/tests/tools/generated Go and rejects generics.
- Native full module final refresh: session88457 running at status write; completion appended separately.

All source/tests production frozen. Only the assigned three project files were
written; caches/scripts/binaries were confined below golang/.cache. No git or
central manifest/progress edits were made by this packet.

A final nonallocating capacity RED showed pdGrowth(MaxUint32) returning an
unrepresentable capacity. The helper now rejects uint32 result overflow as well
as host int arithmetic overflow; checked scalar capacity arithmetic is widened
like existing concrete vector adaptations. The same group checks invalid table
capacities and nested Words invalid-config constructor unwind. Final gates below
supersede the earlier twelve-name runs.

Final thirteen-name handoff gates after the capacity guard:

- Target native: all13 RUN/PASS, exit0 (77d4ae).
- Target checked race: all13 RUN/PASS, exit0 (32753/a28fa2).
- Actual fresh checked JS: all13 RUN/PASS, exit0 (d4e9b8).
- Final checked WASI compile: exit0 (3b660b).
- Final root-package vet: exit0 (c51d2e).
- Final no-generics/source audit: exit0 (12869/fe0825).
- Refreshed native full module88457: exit0 (5928f1), before only the final scalar capacity guard/test addition.
- Checked full24970 and prior full vet55c4f9 passed before that guard-only change.
- Latest full vet encountered an unrelated concurrent packet's temporary compile
  failure: mpnum/inf_format_test.go:9 unused math/big import (c3e57f). No file
  outside this owner's three-file scope was edited to resolve it. Parent combined
  gates must run after that packet stabilizes.

Final owned production/test files are frozen for central review/commit; status
can receive evidence corrections only. All thirteen tests are registered actual
Go test functions (verified native/race/JS named RUN/PASS output).
