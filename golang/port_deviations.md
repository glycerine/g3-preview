# Representational adaptations under review

This ledger records translation choices, not permission to replace algorithms.
See package status files for precise methods and test evidence.

* Storage: Go-managed pointer-free slabs, fixed-width handles and generational
  public tokens implement the plan's deliberate allocator change. Domain cleanup
  remains mandatory. Release internal stale-slot reuse detection is unavailable;
  checked builds retain generations. Scoped region and payload work is pending.
* Numerics: mpn uses owner-confined reusable scratch in place of an OpenMP lock.
  Exact source limb algorithms are retained. Invalid shorter-numerator division
  is rejected by explicit precondition; source mpz handles this before mpn.
  Signed managers, numeric cells and diagnostics remain unported.
* Hashing: portable Go reproduces little-endian loads and signed-char tail bytes
  of the baseline rather than varying hash output with host ABI. Source C++
  portability to other endianness/char signedness remains an audit item.
* Diagnostic and platform replacements must be recorded individually before
  their source units can be marked complete.

### Explicit source formatter arguments and profiles

PrintfFormatter uses concrete borrowed arguments in place of C va_list and
implements Linux C-locale integer/C-string profiles. The selected null-%s glibc
policy and finite failure for the source persistent retry loop are explicit
adapters. Linux %I64u is width64 plus unsigned32; source uint64 mismatches
receive a typed error. Other C conversion domains and Windows CRT state remain
open, so the formatter and formatted exception constructor stay partial.
See PRINTF_PORT_STATUS.md for source callsites, buffer actions and tests.

### Parameter references storage and null stream behavior

Parameters borrow explicit symbol/numeric owners and use scalar references,
pooled concrete entries and owned nullable C text. Reset/Validate mutate shared
backing as in the source. A failed detach retains its published prefix while
cleaning the old reference; pooled pages remain physically charged until Close.
Null C strings model the audited libstdc++ stream badbit prefix with a typed
error. The source global empty value and debug print become explicit owner-local
and diagnostic APIs. See PARAMS_REF_PORT_STATUS.md for remaining adaptations.

### S-expression undefined display continuation and scalar wrappers

The source atom display falls through to an invalid composite cast; Go returns
the defined printed prefix plus ErrSexprDisplayContinuation. Empty composites
retain the source closing-parenthesis-only rendering. Scalar handles/snapshots
replace raw borrows; owner Close destroys outstanding owned numerals. Vector
null callbacks become no-ops, nonempty self-append rejects unbounded source
growth, and capacity preflight avoids leaked increments. BV source debug
short-circuit temporaries and inline16/doubling temporary storage are preserved.
See SEXPR_PORT_STATUS.md for detailed lifetimes, guards and source mappings.


### Concrete string-buffer ownership and numeric boundaries

StringBuffer32/64/128 embed physically charged inline arrays and charge doubled
overflow with old/new overlap. Scalar reads and caller-owned CStr text replace raw
borrows; CStr still writes its terminator and grows at the exact source boundary.
Bulk append preserves successful capacity prefixes before copying bytes or
publishing position, and cross-buffer insertion completes source CStr first.
The heap self-append path that would use freed source storage gets a typed guard;
valid inline and full-heap paths retain source actions. Cold locking and owner
identity checks are explicit Go lifetime adapters.

Source sprintf("%d", unsigned) has a high-bit type-mismatch undefined domain.
The approved adapter interprets uint32 bits as signed32, preserving observed
AMD64 outputs; this is not claimed as portable C semantics. Stack24-byte numeric
conversion avoids invented formatter allocations. LongBits32 rejects values
outside source signed32 range. Arbitrary initial sizes and Windows secure-CRT
runtime/error profiles remain open. See STRING_BUFFER_PORT_STATUS.md.

### Integer inverse domain adapter

The source integer Inv body contains only SASSERT(false); release C++ otherwise
leaves its operand unchanged. Go returns ErrIntegerInvUnsupported after its
existing ownership checks, preserving the operand and budget. This explicitly
checked unsupported domain is not an inverse algorithm or a release no-op
equivalence claim. See MPZ_FACADE_EXTRA_PORT_STATUS.md.

### Recycled source ID reservation

The embedded cold ID generator charges its actual separate backing arrays; its
header is charged by the enclosing owner. Recycle reports fallible reservation,
and an explicit preflight method prepares physical capacity without changing
logical IDs, retaining successful growth prefixes. Reset retains capacity and
cleanup refunds it. This supports future explicit AST reclamation but does not
implement the AST consumer or full process memory service. See ID_PORT_STATUS.md.
