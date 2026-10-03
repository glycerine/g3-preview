# Source hashes and RNG: initial utility port

Owned production files: `hash.go`, `random.go`. They import no solver or numeric/arena packages. Source copyright/license notices are retained in the translations. `LICENSE.txt` remains the repository license.

| Source operation | Go counterpart | Coverage |
| --- | --- | --- |
| `hash.h`: `hash_u`, `hash_ull` | `Uint`, `Uint64` | finite source constants including zero, high bit, maximal values |
| `combine_hash`, `hash_u_u` | `Combine`, `UintPair` | combine source constant; pair composition |
| `mix`, `mk_mix` | private `mix`, `Mix` | source mix constant and string/composite use |
| `hash.cpp`: `string_hash`, `read_unsigned` | `String`, internal `hashBytes`, `WordBytes` | embedded fixture lengths 0–25 including every residual length and high-bit bytes |
| `get_composite_hash` | `Composite` | arities 1–10; independent exact child callback visit order |
| `util.h`: `random_gen` constructor, `set_seed`, both calls, `max_value` | zero value/`NewRandom`, `SetSeed`, `Next`, `Bounded`, `RandomMax` | source constants for seeds zero and UINT_MAX, reseed, bound and max |
| `util.h`: `shuffle` | generic `Shuffle` | source permutation of five elements with seed zero |

`src/test/random.cpp` only prints its seed-zero stream under TRACE and asserts nothing. The port adds exact finite expectations rather than passing a vacuous test. There is no standalone `src/test/hash.cpp` in this checkout. Golden values in `test_vectors/source_hash_random.json` were collected from the actual source headers and `hash.cpp` with GCC 13.3.0, not from the Go implementation. `test_vectors/reference.cpp` records the input generator. It is maintainer-only; ordinary tests embed JSON and neither compile C++ nor access repository source.

Representation decisions: all source unsigned arithmetic is fixed-width uint32 (uint64 in hash_ull); overflow wraps. Source string_hash uses machine-endian memcpy loads and implementation-defined char signedness. The fixture baseline is little-endian and signed-char (this source on Linux/amd64); the Go port fixes those semantics on all targets including wasm. This is a recorded adaptation, not a replacement hash. `WordBytes` feeds the same hash state transitions using an allocation-free word/byte accessor for the bit-vector hash. Inputs must fit source uint32 lengths. `Bounded(0)` and zero-child Composite are invalid preconditions and panic in Go; they do not invent a valid result. Random state belongs to its caller, with no global RNG or math/rand substitution.

Unported source scope: identity hash functors (`int_hash`, `unsigned_hash`, `size_t_hash`, `uint64_hash`, `bool_hash`), object/handle/pointer adapters, `triple`, and pair/triple functor wrappers. Pointer-address hashing needs the coordinator's compact handle convention and cannot use unsafe/raw addresses. `triple_hash`'s declared pair parameter accesses `.third`, an existing dormant template issue to inventory rather than silently fix. Generic hash tables and other util.h functions are separate unported components. This status does not declare the whole util component complete.

TDD evidence: source-vector tests first failed to compile due to undefined operations; production translation then passed. Package tests are hermetic. Native CGO-disabled and g3checked tests, vet, and both wasm cross-builds pass. Actual js/wasm tests pass using toolchain-matched go_js_wasm_exec and Node v22.22.2 with a minimal environment (the inherited environment exceeds wasm argv/env limits). Toolchain: go1.26.4-pont-onethread linux/amd64. Wasmtime 34.0.0 launches WASI tests but the patched toolchain panics before heap initialization: its os_wasip1.readRandom throws while getEnvEarly in runtime/proc.go supports only selected POSIX systems, so supplying GO_DSIM_SEED=1 cannot resolve startup. WASI execution is unverified for these packages with this toolchain; compilation passed.
