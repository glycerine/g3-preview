# Actual cooperation service: selected source no-OMP profile

Sources: src/util/cooperate.cpp/h and src/util/z3_omp.h. This packet preserves
actual `_NO_OMP_` bodies: thread ID0, in_parallel false, all nested-lock primitives
and omp_set_nested no-ops. It implements the source enabled/checkpoint/cooperate,
section constructor/destructor and task constructor/destructor actions. Runtime
enabled and disabled cases both execute; this is no always-success replacement.
OpenMP multilane scheduling, recursive lock handoff and TRACE streams remain OPEN.

Actual global enabled/owner state becomes a single explicit cold utilCooperation
owner, confined to its operation lane. reflect.TypeOf charges its complete header,
including self/budget pointers and lifetime guards. No hot records, task strings,
closures, mutexes, fabricated lock allocation or per-operation heap objects exist.
Source empty lexical RAII section/task objects use matching Begin/End methods;
callers run End only after successful Begin. The owner rejects nil/zero/shallow
copies and refunds its own header exactly once. These are explicit Go ownership
adapters, not source assertion behavior. Borrowed callers must outlive the service.

Checkpoint asserts enabled at its actual debug site, releases matching owner0,
then publishes owner0 in the source critical section. Section destruction only
sets enabled=false; it does not release owner. Task destruction releases owner0
even after a section ended. Repeated no-owner task destruction is a no-op. Source
m_task is never assigned, so there is no retained task spelling or validation of
an otherwise unused task string. Debug BeginTask first asserts enabled, then
fails the actual parallel assertion because no-OMP in_parallel is false. Release
BeginTask calls checkpoint even when disabled. Release nested sections do not
save/restore old flags. Close models no-op lock destruction, retaining raw scalars.

Tests FIRST observed missing APIs in .cache/review/cooperate-red, handle13108.
Five source/lifetime groups first passed release74260/debug4281; a sixth fixture
records thirteen transitions from actual C++ cooperate.cpp compiled with
-D_NO_OMP_ under .cache/cooperate-source/probe.cpp. The original source was only
read; executable, fixture and compiler temporary files remained below golang.
The fixture confirms m_task=nullptr at every site. This release fixture is skipped
in debug because actual debug assertions are tested by the separate action groups.
No registered source ENSURE/VERIFY exists or has been invented for this unit.

Frozen final .cache/review/cooperate-final-3bee762 contains ONLY four new Go files
on committed HEAD3bee762; unrelated active registry/substitution/polynomial and
full-solver RED drafts are excluded. All terminal gates passed:
- full native11874 and checked-debug79862;
- checked-debug race97359 and actualJS73376: five PASS, one release-fixture SKIP;
- release race93563 and actualJS54244: all six exact RUN/PASS;
- full checked-debug vet5239 and WASI build25486 (compile, not WASI execution).
Source/no-generics audit handle32597 is recorded below once terminal.
Logs: .cache/cooperate-{native,debug,race,js,race-release,js-release,vet,wasi,audit}.log.

Frozen SHA256:
- util_cooperate.go: b0eb2b0df031b9a82ac6fe83225a722f20b9a9d7d575d5d1cae49c591f0beee0
- util_cooperate_debug.go: 3e0c9359f50fa43aa646e4bcc1f3b0b4f198752d99fff3774b37b1fe6317ad69
- util_cooperate_nodebug.go: 9380ecb46db27df87d2999bc136506aaa1584604dabf5e541790218684a741cc
- util_cooperate_test.go: 6bdbd4f4ff6e30f9b7268e5daec18209537681fab32b4bd560619468f768d073

Audit32597 terminal exit0:33ported/91partial/1978unported; cooperate.cpp/h
become partial. Exact immutable Go hashes and all named gate results rechecked
before commit. Full solver/SMT/proof/interpolation objective remains incomplete.
