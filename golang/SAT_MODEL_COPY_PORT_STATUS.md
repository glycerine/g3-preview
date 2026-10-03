# SAT model converter implicit ownership operations

This packet adds the actual source implicitly generated class copy constructor and assignment to the committed real model converter. It does not replace named `Copy`, which remains ascending APPEND over a captured source end. The source user-declared destructor suppresses implicit move: rvalue construction/assignment uses the same copy route. `Clone(VectorConfig)` and `Assign(*satModelConverter)` are explicit Go ownership APIs, never shallow cold-owner/record copies. Full solver search/propagation, outer watch ownership, global physical allocation hooks and efficient stable sorting remain separate dependencies.

## Exact source map

Read complete sat_model_converter.h/.cpp and vector.h copy/destruction bodies. Header41–55 entry initializes the31-bit var/1-bit kind and deep-copies m_clauses; entry has no move constructor. Header57–60 owns vector<entry> and declares destructor but no copy/move; standard implicit class copy invokes its member's owning copy. vector.h102–118 copy_core allocates EXACT source capacity, publishes capacity/size before ascending constructors, and constructs only logical entries. vector.h161–175 copy/move constructors,187–214 copy/move assignments and47–56/120–126 destruction determine member action order. The selected source entry layout16 bytes is used only for source growth overflow, never claimed equal to Go scalar-record storage.

`Clone` constructs a new charged cold model/store owner, then exact-capacity outer copy. Each source entry's var/kind initializes before its real nested literal CopyList; nested capacity is exact and word ownership belongs to the destination. No source arena handle or callback environment is silently retained. Copied spare outer entries are unconstructed; a source allocated-empty vector retains its backing capacity on copy. Failed unpublished clone cleans every actual constructed root and all its cold owner graphs; original source is unchanged. Consumed reservation attempts/global scalar owner IDs are not rewound.

`Assign` validates actual owners, makes self-assignment a no-op, destroys current logical nested entries ascending, frees destination outer backing, advances checked reference identity, then performs the same exact-capacity copy. It preserves source publication/action prefixes rather than staging failure atomicity. Source variable/kind precedes the nested constructor. Full logical size is present before construction; a cold `copying` flag and scalar `copyCursor` describe the real completed prefix. They add no pointer/interface to hot records or temporary uncharged slice. Entry access guards unconstructed suffix at the actual read; Reset/Close clean constructed entries only. Destination source failure remains visible and charged immediately.

For source nontrivial outer growth, CAPACITY has not yet been written while entry moves/copies are interrupted; reading that header in copy_core is undefined. The Go capacity-read guard diagnoses ErrSATIncompleteGrowth at that point, after destination destruction for assignment. It does not reuse old sourceCapacity as if it were a newly published C++ header. A source interrupted exact copy has a valid complete capacity/size header: clone/assignment copy its constructed prefix, then guard the actual unconstructed entry access. Unpublished clone cleanup and partial destination cleanup account every real root once.

After interrupted assignment, Reset/Close and restarting Assign are explicit safe Go continuation operations, not claims of C++ exception recovery. New Mk/grow is blocked while that Go construction state is incomplete. Source self-copy through named Copy remains its separately tested finite captured-end operation, including late dangling-iterator guard after actual outer relocation. No invented owning Move API is added to a source class without implicit move.

The runtime is owner-confined. Budget is borrowed and may be shared sequentially with source/destination; a copied destination charges only its actual header/store/outer/nested graphs, never duplicate ownership of the source. All owner headers, real outer scalar arrays and nested Words page/block/directory/descriptor graphs are charged, with old+new peaks and retained pooled free storage explicit. Caller model arrays/clone configs are cold boundary values. No generics, unsafe, production math/big, per-entry callback/interface or source pointer escape is used.

## Tests and source inventory

Tests FIRST observed isolated missing Clone/Assign compiler RED before implementation. Eight meaningful additional groups cover:

- Source size3/cap5 clone exact spare capacity, deep nested capacities/independence and source-close lifetime.
- Empty allocated-capacity3 copy/assignment and destination destruction.
-35 injected assignment prefixes, destruction-before-refusal, full copy_core header publication, ascending constructor prefix, checked unconstructed entries and immediate complete graph charges before Close.
-40 clone constructor/copy prefixes with complete unpublished unwind, unchanged source ownership, self-assignment zero allocations under armed budget, closed and copied-owner guards.
- Late clone/assign guards on partially constructed source, independent destination cleanup, partial Reset and explicit reconstruction restart.
- Unfinished source growth's capacity-read guard and correct assignment destruction prefix.
- Shared borrowed Budget close/refund without charging/releasing original source graph.
- Ascending destruction/copy order evidence from the actual descriptor LIFO reuse/generation ledger (a storage-adapter ordering oracle, not C++ physical-address equality).

The full committed24 source-order/independent groups also run. No standalone model_converter source test registration/ENSURE was invented; these eight are new Go ownership assertions, not eight source ENSURE sites. Existing source assertions/checkers/reverse model logic remain unchanged.

## Final frozen gates

Snapshot `.cache/release/sat-model-owning-copy-final`: HEAD `472e79b0edac224f7b9e48b9147055d7efd77e49` plus only model production/test changes. Exact32 inventory `.cache/model-owning-names.json`, regex `.cache/model-owning-regex`, hashes `.cache/model-owning-hashes.txt`.

All terminal exit0: full native91101, full checked/debug23807, exact32 checked/debug race17266 (1.054s), actual exact32 checked/debug JS32132 (0.186s), full checked/debug vet35532, WASI96436 and source/no-generics67803. Every exact RUN/PASS name was independently checked in both runtime logs. Logs `.cache/model-owning-{native,debug,race,js,vet,wasi,audit}.log`.

Frozen hashes:
- sat_solver_model_converter.go:9fe61c6df1a0450ae36f39b9966af1da8097fec7a1af21a088770a62bdb94e41
- sat_solver_model_converter_test.go:218dbc97ea8c5fb497269992c4b98096f83de70d6b39e9e7e5ddddadb83d73e9

No committed watch files, root-owned SAT_SOLVER_PACKET.md/central metadata or Git were edited in this packet. Full solver registered-test missing APIs are excluded from this prerequisite snapshot, not hidden behind build tags or replaced by another solver.

## Preserved source hashes

- src/sat/sat_model_converter.h:760b847933f74903f81b19dcc5cd98a560f1d01407dabc8042638aaab9d01c9a
- src/sat/sat_model_converter.cpp:57f99643ce2dc3eaf8866c01a9c4e621a64c84b496154e071cc8d819b79ab6e9
- src/util/vector.h:aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e

## Coordinator integration

Both Go hashes were verified against live files and the frozen source packet.
The prospective committed tree `.cache/review/sat-model-copy-integration-81f1059`
contains HEAD81f1059 plus only these two Go changes and final metadata. Fresh
full native PASS (root2.451s, mpnum22.856s) and full g3checked+g3debug PASS
(root2.717s, mpnum23.478s) verify interaction with the newly committed sparse/shared
owners. Every32 RUN/PASS name in the frozen race and actual JS logs was checked
by the coordinator. Final source/no-generics inventory remains33ported/87partial/1982unported;
whole-source completion flags remain false. Unrelated dependency-ref and full
solver drafts are excluded from this independently verified committed packet.
