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
