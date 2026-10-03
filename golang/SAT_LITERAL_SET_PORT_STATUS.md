# Actual SAT literal-set storage

The concrete sat::literal_set at src/sat/sat_types.h230–284 now owns charged
SAT uint-set membership and ordered values. It does not reuse the earlier
uncharged satLiteralSet convenience facade as solver storage. Source registered
SAT solver assertions still require the actual solver; no registered literal-set
fixture or source ENSURE count is invented by this packet.

Source default/vector construction maps to newSATSolverLiteralSet and
newSATSolverLiteralSetFromVector. Implicit owning copy maps to Clone;
vector assignment maps to AssignVector, which resets BEFORE the first input
size/read. Set assignment delegates the actual uint-set membership-first then
values copy, retaining exact capacities and destroy-before-copy prefixes.
Insert/Remove/Pop/Contains/Empty/Size/Reset/Finalize/Intersect/Union/Close map
the corresponding source methods/member lifecycle, using literal.index and
to_literal conversions exactly. Both actual cold owner headers, all membership
bytes and full unsigned backing capacities are charged. No production slice
conversion, boxing, generics or additional numeric algorithm is introduced.

ToVector constructs a new scalar list in the caller-owned real LiteralStore,
then iterates the captured source range and pushes with source geometric
capacity. This adapts source returned owning literal_vector to an explicit
caller-owned storage environment. Failure destroys its unpublished root;
allocator pool/free storage stays charged until its actual owner closes.
Begin/End, Value, Advance, PostAdvance and Equal represent the actual wrapped
const-pointer iterator operators; Iter/Next is an additional combined traversal
adapter. Scalar Equal does not dereference freed values and compares two
captured source null pointers equal across owners. Physical Value access
guards freed/replaced storage; logical Reset/pop does not end scalar lifetime
or change a captured endpoint. Checked origin/self/lifetime/overflow domains
are Go adapters, not additional source reference callbacks.

Tests were written before production and isolated HEADd25fd72 plus only the
new tests observed missing-API RED90174. The first six groups passed after
implementation (checked/debug12696); source Begin/End/operator tests then
observed missing-API RED before implementing those operations. Seven groups
cover real vector construction/output/assignment, independent1500-step ordered
membership oracle, exact copy capacities/deep independence, union/intersection,
retained iterator lifetimes, source flag-before-list refusal, reset-before-invalid
input,32 constructor refusals,25 returned-vector failures and immediate whole
physical charges before Close, plus warm zero allocations and owner guards.
These are additional finite Go fixtures, not source registered assertion sites.

Frozen prospective committed tree .cache/review/sat-literal-set-final-d25fd72
contains HEADd25fd72 plus only this packet's two Go files and final metadata.
Full native58584 PASS (root2.446s, mpnum21.763s); full g3checked+g3debug72336
PASS (root2.642s, mpnum22.174s). Fresh exact seven race groups64468 PASS
(root1.042s) and actual checked/debug JS50675 PASS (root0.156s); each exact
RUN/PASS verified independently. Whole-module checked/debug vet99279 and
WASI build82801 PASS; WASI is compilation only. Logs are the frozen tree's
.cache/literal-{native,debug,race,js,vet,wasi,audit}.log. Source/no-generics
inventory remains33ported/87partial/1982unported; full SAT solver and all
original complete-port requirements remain open.

Frozen Go hashes:

```
af727ef0d31642918be4860272de1b7684d73570856b5a04792a3462a0ac40f5  sat_solver_literal_set.go
e419b75caa37266ad78295611b7883abec0b463db0cf4fe95d06f51041cff643  sat_solver_literal_set_test.go
```
