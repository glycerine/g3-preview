# Infinitesimal formatting correction

Final parent integration against committed prerequisites passed full native
default and full checked debug module suites, full checked debug race tests
for root/mpnum, checked debug vet and WASI compilation, and source/no-generics
audit. Actual checked debug JS/Wasm passed every package with all four new groups
explicitly RUN/PASS in `.cache/format-exception-integrated-final-js.log`.
Production/tests matched the verified snapshot before committing. WASI runtime
is not claimed.

Scope: only `RationalService.InfString` and `InfIntString` bodies in the existing concrete representations, plus removal of the now-unused `strconv` import. No numeric constructors, operators, lifecycle hooks, or public signatures changed. The frozen InfEps specialization provides the private shared formatting helpers; this packet depends on that earlier packet.

## Source and method mapping

- `src/util/inf_rational.h::to_string` → `InfString`: epsilon zero formats the first rational alone; nonzero formats `(first +/-e*abs(second))`. The absolute-value rational is copied, made absolute, formatted, then destroyed at the end of its expression before the closing parenthesis. The shared `infEpsInfPrepareNumeral` helper preserves that action order and immediate failure cleanup.
- `src/util/inf_int_rational.cpp::to_string` → `InfIntString`: integer epsilon zero formats the first rational alone; nonzero formats the opening parenthesis and first rational, then the sign and absolute integer coefficient, then the closing parenthesis. Existing typed `ErrInfIntOverflow` adaptation for signed MinInt32 negation remains; first-component formatting precedes the domain check.

Both use existing manager-owned, budgeted decimal workspace, scalar byte storage, and one final caller-owned output string. Intermediate Go strings and `strconv` output are eliminated. The format buffer and numeric decimal scratch retain their manager ownership until Close. Growth reserves incoming storage while outgoing storage remains charged. No new owner/header, global mutable manager, per-record pointer, unsafe access, Go map, or Go generic is introduced.

## Tests first and validation

Behavioral RED was observed before changing either formatter: `TestInfFormatOneFinalAllocation` measured three warmed Go allocations for each formatter against the required one. After the two body changes both measure exactly one final output allocation.

Four test groups cover allocation counts and fixed decimal outputs; epsilon zero, both signs, rational coefficients, signed integer boundaries, foreign-owner and closed-owner guards; finite injected-failure prefixes with immediate independent numeric-cell and Words-root accounting before any subsequent String/Del/Close; and the old-plus-new workspace growth peak. The peak test withholds enough budget that a net-size-only reservation would succeed but the incoming buffer cannot fit, checks unchanged storage and charges on failure, then verifies successful growth and full refund at Close. These are additional correction tests, not newly claimed source assertion sites.

Final isolated snapshot: `.cache/release/inf-format-final`, based on HEAD `7309f1a`, overlaid only this packet's three Go files and the two frozen InfEps prerequisite Go files. All temporary paths and caches remain beneath `golang/.cache`.

- Full native `go test -count=1 ./...`: PASS, mpnum 10.423s.
- Full checked `go test -tags=g3checked -count=1 ./...`: PASS, mpnum 10.208s.
- Checked race, all four `TestInfFormat` groups: PASS, 1.080s.
- Fresh actual checked JavaScript/Wasm execution: all four groups individually RUN/PASS, 0.150s.
- Checked vet: PASS.
- Checked WASI compile: PASS; no WASI runtime claim.
- `go run ./cmd/portcheck`, including no-generics syntax audit: PASS.

The unrelated `InfSetInt32` constructor-copy route remains a separately identified inventory item and is outside this correction packet.
