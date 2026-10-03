# SAT main configuration: implemented source contract

This packet implements the concrete runtime-owned main configuration from
`src/sat/sat_config.h/.cpp`, with defaults and typed lookup routes audited against
`src/sat/sat_params.pyg` and `scripts/mk_genfile_common.py` lines 828–948. It uses
the actual `GlobalParams`, `ParamsContext`, and `SymbolManager` owners. It does
not introduce a replacement registry, an injected fallback snapshot, a fake SAT
solver, or an implicit registration callback. The full solver remains open.

## API and source methods

The owned implementation is `sat_main_config.go` and
`sat_main_config_update.go`; tests are the transferred
`sat_main_config_test.go` and new `sat_main_config_policy_test.go`.

* `newSATMainConfig(globals, local, budget)` implements `config::config`:
  RestartMax=0 precedes eleven symbol constructors in declaration order;
  NumParallel=1 follows them, then the real source Update runs.
* `Update(local)` implements the complete `config::updt_params` body, including
  every branch, every scalar assignment, and the three exact exception messages.
* `Clone(budget)` / `Assign(source)` implement the source implicit scalar and
  symbol copying. They perform no parameter lookup, Update, registration, or
  symbol construction. Assign retains destination Go ownership/runtime bindings;
  cross-symbol-domain assignment is a checked API error.
* `State()` returns the pointer-free scalar state and its initialization mask.
  `Close()` releases only the actual config header; it is idempotent and does
  not close borrowed owners or erase their interned symbols.
* `satConfigMegabytesToBytes` implements `util.h::megabytes_to_bytes` for the
  selected Go host word width. The scalar 32/64-bit helper is exercised at
  saturation and UINT_MAX sentinel boundaries without huge allocations.
* `config::collect_param_descrs` was already independently committed as
  `collectSATConfigParamDescrs` with all 24 generated entries. This packet
  leaves that implementation and its five tests unchanged.

There is no separate public generated `sat_params` facade claim. All 23 generated
getter routes actually consumed by config use their exact local → current sat
module → literal typed fallback. The generated `dimacs.core` option belongs to
the descriptor collector but has no config member/read; tests preserve that.

## Source behavior preserved

Each Update calls the real `GlobalParams.GetModuleName("sat")`, obtaining a new
retained reference to the current module cell, and destroys that lexical
snapshot on success or error. It never requests descriptors or lazily invokes
the catalog registration dispatcher. Wrong-kind local values fall through to
the matching module kind; wrong-kind module values fall through to literals.
Symbol-default construction is evaluated before lookup even when overridden.

The three hidden simplification parameters read local parameters only. A zero
result from the ordinary random-seed getter triggers the source local-only
zero-default reread. All four phase choices, both restart choices, and all five
GC choices are accepted, including `psm_glue`. Only dyn_psm assigns SmallLBD/K;
K is clamped to 255 after assignment. Other GC modes retain the prior values and
initialization state. NaN, infinities, signed zero, negative doubles, zero and
maximum unsigned values are preserved without invented policy validation.

Invalid restart, phase, and GC return source `default_exception` values with
messages `invalid restart strategy`, `invalid phase selection strategy`, and
`invalid gc strategy`. Every completed earlier assignment remains visible;
later fields retain their previous values. No whole-config rollback is used.

## Ownership, accounting, and explicit adapters

The one cold config owner has self/provenance checks, borrowed real-owner
pointers, one embedded state, and eleven embedded Symbol values. Its complete
actual `reflect.TypeOf(satMainConfig{}).Size()` is reserved before allocation
and refunded on construction failure/Close. Embedded fields are not charged
again. Clone charges a second actual header. There are no persistent slices,
text copies, tables, or ParamsRefs owned by the config. Symbol/cache storage and
parameter/module backing are charged by their actual borrowed owners.

The 26 scalar source members are represented by a pointer-free state with a
26-bit Initialized mask. Go zeros for uninitialized SmallLBD/K do not advertise
source-defined values: callers must check the corresponding mask bits. All
ordinary source assignments set their bits only after successful typed reads.
Implicit copies preserve the mask without inventing assignments. The mask is
explicit charged representation metadata, not a change to source defaults.

Cold owner shallow copies, foreign ParamsRefs/symbol domains, and closed borrowed
owners produce typed errors before mutation. These checks adapt source global
ownership/pointer preconditions. The borrowed GlobalParams/ParamsContext/
SymbolManager must outlive the config. Operation and Budget use require caller
confinement; this owner does not claim concurrent Update/Close safety.

Failed construction destroys the unpublished config header and its lexical
module reference while retaining already completed symbol publications in the
borrowed manager. The cold failure sweep reaches actual success after all
observed reservation prefixes and checks live Params blocks/references before
teardown. Inspection reads the real symbol table: calling Intern to inspect a
failed prefix would itself potentially threshold-grow the table before lookup.

## Tests-first and final verification

The parent's initial three API groups were observed missing-API RED (handle
56836, `.cache/sat-main-config-red.log`). Expanded tests were observed actual
missing-API RED on committed HEAD 2f9931b plus only the eight actual registry
production files (terminal handle 8329,
`.cache/sat-main-config-expanded-red.log`) before either implementation file
was written. Initial GREEN handle 85504 and final expanded GREEN 3716 passed.

There are **zero original registered SAT-config test assertions** to claim.
The **16 new config test groups** derive finite action/branch expectations from
the complete source bodies and add independent width/IEEE/ownership checks.
The five preexisting static collector groups run unchanged. The solver-level
assertion in `sat_user_scope` is not attributed to config-only tests.

Final frozen-byte snapshot:
`.cache/release/sat-main-config-final`, committed base
`2f9931b49f54cece187019c7eb667a892722eed3` plus only the four owned Go files and
arena agent's eight frozen actual registry **production** files. Unrelated
drafts and registry test drafts are excluded. Registry hashes exactly match
`.cache/gparams-final-hashes.json`; copied dependency inventory is
`.cache/sat-main-config-registry-sha256.txt`.

Terminal PASS results:

| Gate | Handle | Evidence |
| --- | --- | --- |
| Full native `./...` | 99951 | g3 2.553s; mpnum 21.427s |
| Full `g3checked,g3debug ./...` | 7436 | g3 3.030s; mpnum 21.864s |
| Named checked/debug race | 47619 | all 21 top-level config/collector RUN/PASS; 1.150s |
| Actual checked/debug JS wasm | 14673 | same 21 top-level RUN/PASS; 0.212s |
| Checked/debug vet `./...` | 87279 | PASS |
| Checked/debug WASI test compile | 87404 | PASS, `.cache/sat-main-config-final.wasi.test` |
| Source inventory + no-generics audit | 92659 | PASS, committed baseline 33/97/1972 |

All log paths are `.cache/sat-main-config-final-{native,debug,race,js,vet,wasi,audit}.log`.
Both race and actual JS logs explicitly contain all 21 expected named RUN/PASS
registrations, including every new config group and the unchanged collector
groups. Warm Update with a real retained module and exhausted config/service
reservation counters performs zero heap allocations, leaves budgets unchanged,
and returns the original immediate live block/reference counts before Close.

Frozen Go SHA256 inventory is `.cache/sat-main-config-final-sha256.txt`:

```text
271c136622bc39d81f5f452fcf590660771c7d6ac2a6d7ab0b5ddcd44229330f  sat_main_config.go
d584a11f87217ec3567ca40a858650b30f5b4d39fd846bf07a7b36bf704210a9  sat_main_config_update.go
f517ddefa103e74b50062a2e2e1cda419a8a7e07c984e37b2abdc2bb3a86d917  sat_main_config_test.go
801da4837450cab30502587712bac68415a5a03f6707b99230a5209680093145  sat_main_config_policy_test.go
```

No central ledger/progress, Git metadata, descriptor files, existing owner files,
or frozen substitution files were changed. All writes and build artifacts are
under golang. All code, tests, and tools use concrete Go types without generics.
The actual registry was subsequently committed as
`e59f032025bfccb1bb17b58e478fe75ac9e64244` with identical production hashes.
The parent is running final integration on that committed registry base plus
these four frozen Go files; those separate integration results are not claimed
as completed in this document.

## Coordinator integration

Frozen `.cache/review/sat-main-config-integration-e59f032` adds only the four
verified config Go files to committed actual registrye59f032, including its
twenty-two registry tests. Full native handle10933 and checked-debug15735
passed; mpnum22.352s/22.875s. Coordinator verified four source hashes and all
twenty-one exact race/actual JS RUN/PASS groups. Updated source/no-generics
audit82917 passed, ledger33ported/99partial/1970unported. Existing source
config rows remain partial; full SAT/SMT/proof/interpolation work remains open.
