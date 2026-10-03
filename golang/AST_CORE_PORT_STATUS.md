# AST core original proposal

This historical proposal predates the implemented runtime packet. Current
implementation, frozen tests, ownership adapters and remaining whole-manager
work are recorded in AST_CORE_PACKET.md.

The original read-only audit of `src/ast/ast.h`, `src/ast/ast.cpp`, registered
`src/test/ast.cpp`, and concrete chaining/persistent-array dependencies proposed
an owner-local pointer-free core covering all five node kinds. This proposal itself was an audit artifact; the later runtime packet implements
real managers, nodes and eager plugins. Solver behavior remains open.

The source constructor eagerly initializes basic/label/pattern/model-value/
user-sort plugins and retained roots, and creates a second real formatting
manager. A reduced fake constructor is not an acceptable counterpart. Source
`basic_decl_plugin` is embedded in ast.h/ast.cpp, not separate files. Family
order is basic, label, pattern, model-value, user-sort, arith; arith family
registration alone does not implement the arithmetic plugin.

Registered ast.cpp has nine active assertions: two Boolean interning checks
and seven persistent-array checks. Tests2–4 and old alternate tst1 are commented
out. The persistent array requires actual parray ROOT/SET/PUSH_BACK/POP_BACK
records, root-preserving trail reversals, rerooting and expression reference
actions, not copied slices. Registered parray.cpp also covers both int32
preserve-roots policies and a million-update AST stress route.

The immediate dependencies are real source chaining storage and fully charged
ID free lists. AST IDs must remain independent of arena slots/generations.
Public scalar handles need owner/slot/generation; hot records retain separate
kind, source ID, hash, RC and marks. Parameter support must include all six
source kinds and rational copy/destruction plus attachment-time AST retains.
The shared contracts remain provisional until those dependencies and actual
manager/plugin interfaces receive final review.

Interning fixtures must preserve ignored sort cardinality/private flags,
ignored function idempotence, function hash's omitted range, ignored quantifier
qid/skid, backward AST-array hashing, and decl_info's left-count parameter
comparison without a right-count guard (defined finite cases versus undefined
reads must be distinguished). Quantifier construction explicitly rejects
simultaneously nonempty patterns and no-patterns.

Release AST RC increments wrap uint32; source debug asserts before increment
at UINT_MAX. This is distinct from sexpr source behavior. The three-argument
`ast_manager::mk_proof(fid,k,arg1,arg2,arg3)` builds three local arguments but
passes a count of two; later proof coverage must preserve this finite source
route. These are audit findings, not implemented coverage.

Full manager scope still includes all factories/operators/updates, dependency
and persistent dependency arrays, marks, translation/plugin inheritance,
arithmetic coercions, proof builders, resource limits, trace/printers and
specialized reference containers. No source unit is marked complete here.
