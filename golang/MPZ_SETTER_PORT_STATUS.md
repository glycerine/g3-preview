# MPZ selected setter/capacity/normalization closure

Owned Go files: `mpnum/mpz.go`, narrowly the base-two Power path in `mpnum/mpz_power_shift.go`, the specifically approved oversized-base-two failure branch in `mpnum/mpz_power_shift_test.go`, and new `mpnum/mpz_setter_publication_test.go`. No other production/test files, central metadata or Git files were edited. This document is the approved new packet record. Baseline `fdf6f3c`; isolated snapshot `.cache/release/mpz-setter-red` contains only these four overlays, excluding unrelated active SAT/Sturm drafts.

## Source mapping

- `mpz.h:132 allocate_if_needed` → new concrete private `allocateIfNeeded(*Int,uint32) error`. Clamp requested capacity to source6; sufficiently large big cell is unchanged. Small promotion publishes sign1 before allocation. Insufficient big cell is freed before sign1 and replacement allocation. No preservation/copy of old limbs. Allocation refusal leaves the source completed sign/free prefix and the already established exact owner failed token.
- `mpz.cpp:1450–1480` base-two Power fast path now operates on the actual caller destination. Small exponents Del then publish inline power; large exponents request `p/32+1` limbs using actual destructive helper, execute source size/capacity diagnostic sites, publish size, write lower zeros/high bit, then positive sign. Existing sufficient capacity is retained. Caller may equal base; switch selection occurs before caller mutations. Source expression `1<<31` is represented by explicit uint32 shift, an existing portable limb-bit adapter for signed C++ shift UB.
- `mpz.cpp:165,215` big signed/unsigned scalar setters retain existing allocation-before-sign refusal behavior and cell reuse. Added exact capacity>=6 assertion after optional allocation, before sign/limb mutation, selected by existing `modularSourceDebug`; error category existing `ErrIntegerAssertion`. No extra build-tag helper or API was needed. Signed INT64_MIN magnitude remains the documented unsigned-magnitude Go adapter for source signed-negation UB.
- `mpz.cpp:1556 ensure_capacity` remains distinct preserving `ensureOwnedCapacity`: replacement allocated before old release. It is not reused as destructive allocation. Its already-correct failure behavior is exercised independently.
- `mpz.cpp:1597 normalize` is already implemented by `normalizeOwned`. New direct fixtures verify all-zero Reset, signed one-limb INT_MAX collapse, negative magnitude2^31 remaining big, and leading-zero trim retaining payload/sign. The source has no additional normalize SASSERT site to invent.

The physical Words descriptor length is prepared for indexed writes, and `Words.Resize` clears shrunk logical tails. Consequently existing scalar setter size/sign/ref publication around infallible limb writes is a storage adaptation, not an exact instruction-by-instruction claim. There is no allocation/callback between those writes. Source externally observable successful values, ordinary allocation-refusal state, and setter capacity assertion placement are tested. Source retains physical unused cells/limbs differently; this packet does not claim its raw tail/allocator topology. The base-two caller path itself preserves source size/sign/allocation/loop action order, using the charged Go ownership representation.

The exact failed token is permanently dead charged metadata, never a live numeric cell. Normal reads reject it; Del/Reset provide the previously approved ownership cleanup continuation. This is not a claim that C++ dangling allocation state can be retried or recovered. Base-two refuses at actual existing allocator/storage access, after preceding source sign/free actions; no new early exponent/capacity cap was added.

## Tests-first evidence and precise scope

1. Actual public behavioral RED28459: `.cache/mpz-setter-public-red.log` shows base-two exponent1024 into existing negative big output retaining old sign/cell, whereas source frees old and publishes sign1 before refusal. This was observed before translation.
2. Private missing-API RED: `.cache/mpz-setter-private-api-red.log` reports missing `allocateIfNeeded` after tests were written.
3. Actual diagnostic behavioral RED98969: `.cache/mpz-setter-capacity-debug-red.log` shows scalar setter changing sign/digits on an actual scalar-record capacity5 fixture instead of executing source capacity assertion. Actual physical payload is still6; no invalid backing access or fake callback was needed.
4. First selected GREEN6569/89248: `.cache/mpz-setter-first-green.log` and `.cache/mpz-setter-target-green.log`. Ordinary scalar big allocation-before-sign refusal and normalization already worked; their tests are candidly immediately-GREEN confirmation, not fabricated RED claims.
5. Broader mpnum initial run44627 exposed obsolete old oversized-base-two atomicity fixture (`.cache/mpz-setter-mpnum-native.log`). Parent approved changing only that observed branch to immediate graph audit, positive sign/exact failed token/read guard, then Del before independent remaining domain fixtures. Successful arithmetic, shift table, generic high-exponent domain and 0^0 expectations remain unchanged.

Nine added groups cover destructive/preserving allocation distinctions; small promotion refusal and sufficient-capacity no-action; signed/unsigned scalar refusal, source assertion timing, exact scalar boundary/reuse mathematics; actual normalize branches; bounded base-two values and input/output aliases; every reservation prefix to success; requested capacity33 and retained sufficient capacity; warm source fast path zero Go allocations. Test-only math/big supplies independent exact mathematical expectations. Immediate audits count actual cells AND Words before rendering, fault disable or cleanup, excluding only the exact owner failed token. Constructor fixture Close returns actual Budget to zero.

No direct registered `src/test/mpz.cpp` or `src/test/mpq.cpp` assertion is newly claimed by these nine added regression/property groups. Previously ported source fixtures in the eight existing Power/shift test groups are preserved and rerun. Source internal assertion fixtures are diagnostic tests, not registered test-unit assertions. Generic Power loops, rational Power, integer/rational parsing/root/bitwise/GCD staging and sequential rational wrappers remain OPEN per `MPZ_REMAINING_PUBLICATION_PACKET.md`. Release integer0^0 and actual high-bit-mask continuation are intentionally not corrected by this narrower packet. Source small-object allocator/global-memory/TLS topology and TRACE/persistent ostream integrations remain OPEN.

## Frozen targeted verification

Exact17 top-level names (new9 + existing8) each explicitly RUN/PASS in native, checked/debug, checked/debug race and actual checked/debug JavaScript wasm logs. Inventory `.cache/mpz-setter-names.txt`, exact regex `.cache/mpz-setter-regex.txt`.

- Native99042: PASS mpnum0.856s, `.cache/mpz-setter-final-native.log`.
- Checked/debug1745: PASS mpnum0.805s, `.cache/mpz-setter-final-debug.log`.
- Checked/debug race84405: PASS mpnum12.073s, `.cache/mpz-setter-final-race.log`.
- Actual checked/debug JS18162: PASS mpnum3.027s, `.cache/mpz-setter-final-js.log`; toolchain-matched golang-local js-exec and TMPDIR preserved.
- Checked/debug mpnum vet: PASS, `.cache/mpz-setter-final-vet.log`.
- Checked/debug full-module WASI build15110 and mpnum test compile96929: PASS, `.cache/mpz-setter-final-wasi.log`, `.cache/mpz-setter-final-wasi-test.log`; output `.cache/mpz-setter-final-mpnum.wasm`.
- Source/no-generics audit60840: PASS, `.cache/mpz-setter-final-audit.log`; baseline ledger33ported/145partial/1924unported. No central coverage credit was changed by this agent.

Final full-module native/debug integration is parent-owned and not replaced by these targeted checks.

Frozen SHA256:

```
c5f56260015e28a1f37054d6d778f64e16a5d36ff5255c10f96710ece8af32a5  mpnum/mpz.go
d12c0cd2ff7b29b92a08c948791ea3dd71b34ee547c26b7dcda3c6004a31f756  mpnum/mpz_power_shift.go
3bc63f0eeb1dcd2ccbab74b3855c7a49c74b8bc799d2d4d214fe5d62e7d26f89  mpnum/mpz_power_shift_test.go
15301feccdb2b6f68c85714d31996c771596ad2841a27b60c05c540977ead405  mpnum/mpz_setter_publication_test.go
```

## Parent integration

Frozen `.cache/review/mpz-setter-integration-fdf6f3c` uses the exact four owned
Go overlays on committedfdf6f3c. Full native passes root2.831s/mpnum21.339s;
full checked/debug passes root3.134s/mpnum21.904s. Root independently verifies
all17 race/actual JS RUN/PASS entries and unchanged four-file hashes. Central
source evidence marks selected capacity/Power work partial and keeps all
higher-level obligations open. Final metadata source/no-generics audit passes
2102units/677APIs:33ported/145partial/1924unported.
Live ongoing Sturm and allocator tests are excluded from this frozen tree.
