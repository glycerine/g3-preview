git use: Do not amend or rebase commits in git. Make a corrective additional commit instead. We push
in the background and have seen an accidental git fork and we do not want to lose data.

A rule for storing test vectors and associated test data: do not create a 
directory named testdata. Avoid this as a directory name. Use test_vectors instead. 
The reason is that the Go fuzzer uses the directory name testdata/ for its ephemeral
storage and the directory will get wiped and we could lose the test vectors by 
accident and that would be bad.


Porting verification: run fast, focused tests for the features being ported.
Run the full native test suite sparingly, roughly once per hour, since it takes
about eight minutes. Keep commits frequent and do not commit known failing
focused checks. No Go generics, C/cgo, source hashing, wasm/wasi builds, or race
checks for the current single-goroutine porting work. Preserve the pkg/ split.
