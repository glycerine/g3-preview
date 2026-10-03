# Parameter-name normalization

Both norm_param_name overloads from src/util/params.cpp are translated.
params.h/cpp remain partial: descriptor and parameter-reference algorithms
are separate packets, not replaced by this helper.

NormParamName accepts an explicit nullable C-string representation. It stops
at NUL, removes exactly one leading colon, maps ASCII capitals to lowercase
and hyphen/colon to underscore, and returns underscore for null/empty names.
Non-ASCII bytes retain their exact values, including malformed UTF-8.
NormParamSymbolName reads the symbol owner's actual bare string directly under
its mutex; it does not intern names or allocate an intermediate spelling.
Numerical/marked/foreign/closed records receive typed checked errors. Outputs
are caller-owned strings; no normalized text or workspace is retained or charged
to the borrowed symbol budget. Existing source symbols remain unchanged.

Tests were written first and observed missing-function RED. The first implementation
also failed the byte/UTF-8 fixtures because Go string range skipped continuation
bytes; indexed byte iteration corrected that actual behavioral RED. Four groups
cover source branches, all255 non-NUL byte values,1024 CString suffix cases,
symbol ownership and lifetime, FailAfter(0) with unchanged owner storage/budget,
and exactly one allocation for the final caller string. No source standalone
normalizer test/assertion registration is invented.

The frozen packet passed uncached native and checked tests for every package,
checked race tests for the changed root package, and checked vet for every
package. Actual checked JavaScript/Wasm execution passed every package with
explicit RUN/PASS records for all four new groups in
`.cache/param-name-final-checked-js.log`. Checked WASI compilation passed;
this is compilation evidence, not WASI runtime evidence.
The source inventory and no-generics syntax audit passed: 2102 units,
677 APIs, 21 ported, 42 partial, 2039 unported. Production and test files
matched the verified isolated snapshot before committing.
