# Concrete exception values and error classification

Final parent integration passed full native default and CGO-disabled module
tests, full checked debug module tests, full checked debug root/mpnum race
tests, checked debug vet, WASI and native-386 compilation, and source/no-generics
audits. Actual checked debug JS/Wasm passed every package with all five new
groups explicitly RUN/PASS in `.cache/format-exception-integrated-final-js.log`.
Production/tests matched the verified snapshot before commit; final ledger
integration passed checked root/internal-portcheck tests. Inventory is 25 ported,
46 partial, 2031 unported of 2102 source units and 677 API operations.
WASI and native-386 runtime execution are not claimed.

Owned files: util_z3_exception.go/test, util_cmd_exception.go/test, and this
status. All writes/build cache/temp/output remain under golang; no central
metadata/git/other-owner files were edited. No Go generics, unsafe access, hot
interface fields, or command runtime scaffold were introduced.

## Source mapping and remaining boundary

| Source | Go implementation |
| --- | --- |
| error_codes.h:24-38 all14 macros | ErrorCode constants ErrorOK=0 and ErrorMemout101 through ErrorAllocExceeded113 |
| z3_exception.cpp:27-33 base code/has_code | cold Z3Exception surface; DefaultException.ErrorCode returns0; concrete HasErrorCode compares dispatch result to0 |
| z3_exception.cpp:35-37 z3_error constructor | NewZ3Error, typed zero-code rejection |
| z3_exception.cpp:39-56 z3_error::msg | Z3Error.Message/Error fixed labels, exact unsoundess typo;112 and all unlisted values return unknown error |
| z3_exception.cpp:57-59 z3_error code | Z3Error.ErrorCode preserves the uint32 input |
| z3_exception.cpp:70-75 std::string constructor/msg | NewDefaultException stores full immutable string bytes; Message/Error observes c_str first-NUL prefix |
| cmd_context_types.h:51-55 compose | NewCmdExceptionSymbol/At, one final message builder, raw checked owner spelling bytes, source symbol ostream form |
| cmd_context_types.h:57-63 five constructor routes | NewCmdExceptionCString, NewCmdException, NewCmdExceptionAt, NewCmdExceptionSymbol, NewCmdExceptionSymbolAt |
| cmd_context_types.h:65-67 position methods | HasPos line>=0; Line/Pos typed guard; negative pos is valid when line>=0 |
| cmd_context_types.h:70-71 stop_parser_exception | distinct zero-size StopParserException, Go error transport label only |
| symbol.h:97-111 ostream branch; tptr.h:43 signed UNBOXINT | compose string bytes/null/k! signed-int32 numeric display, consistent existing AMD64 numeric-symbol adaptation |

The complete error_codes.h classification is covered. z3_exception.h/.cpp remain
**partial** because the formatted constructor at z3_exception.cpp:61-68 is not
implemented. Its actual warning.cpp:106-143 format2ostream dependency is also
**open**: printf/va_list semantics, OS invalid-parameter handling, non-Windows
AMD64 initial1024-byte vector, one-call truncation/fallback behavior, versus
other platforms' initial128-byte repeated resize2n+1 route. Go fmt is not a
substitute and no formatting method was exported. Virtual C++ destruction has
no independently owned service graph here; cold Go values/messages are reclaimed
by caller lifetime. The command header remains partial for commands/parser
runtime classes; only the complete CmdException and stop signal units are covered.

## Ownership and byte/location domains

Z3Exception is a cold returned error interface, not a retained boxed hot record.
DefaultException/CmdException message strings and their cold values are caller
boundary storage; they do not reserve from SymbolManager or any other owner
Budget. Input Go strings are immutable and keep their backing live; full
std::string-constructor bytes, including NUL suffix, remain privately retained.
The exposed source msg C-string observation stops at first NUL. CString command
constructor and the compose prefix stop at first NUL before storage/concatenation.
Source constructor null C pointers are not introduced as a Go nullable API.

Symbol constructors borrow the manager only during construction under its cold
mutex. They validate source plain-symbol provenance, read cached length and
checked spelling bytes directly, use a stack decimal digit buffer, pre-grow one
final caller-owned message, and retain no Symbol/owner reference in the result.
Null prints null; numeric prints k! followed by source signed-int32 UNBOXINT
interpretation. Marked/foreign/closed owners receive existing typed errors.
Owner Budget and allocation metadata remain unchanged even with FailAfter0.
Both4/8 pointer widths and all255 non-NUL byte values are independently checked.
Returned exception text survives manager Close. Negative line means no position;
Line/Pos reject it, while all signed32 position values are accepted when line>=0.
StopParserException is deliberately not a Z3Exception; its Error string is a
Go-only transport label for the source empty distinct type.

## Tests, source registration, and RED

src/test/main.cpp:193 registers TST(ex), calling src/test/ex.cpp:64-67. Its tst1
prints testing exception using local custom ex/ex2 classes; tst2 prints formatted
Format 12 twelve through the still-open variadic constructor. There are **zero
active ENSURE/SASSERT test assertions** in ex.cpp; those two printed cases are
not claimed ported by this packet, and the formatter consumer remains pending.
The implementation nonzero-code and location SASSERT obligations are exercised
explicitly by meaningful source fixtures. New tests are independent Go coverage,
not invented source assertion counts.

Five actual registered test groups were written first. RED observed missing
concrete constructor/type APIs (364a20), then all5 groups passed after translation:

- TestZ3ExceptionSourceCodesAndLabels
- TestDefaultExceptionSourceStringAndEmbeddedNUL
- TestCmdExceptionSourceConstructorRoutesAndLocationGuards
- TestCmdExceptionSourceSymbolCompositionAndBorrowedLifetime
- TestCmdExceptionIndependentByteDomainsAndOwnerAccounting

Native targeted all5 RUN/PASS (47059a). Final checked race/actual checked JS,
vet/WASI/no-generics results appended on completion. No reference checkout is
needed by tests at runtime.

## Source SHA256

```
a9a944d327c1755b886d1f95a9ec0a1bdc48582c20e91b7c4bc576b038cedd3c  src/util/error_codes.h
a73002ce776425cbd302e2aca030c18babae39ea754175b57ec49cf8d022ee5a  src/util/z3_exception.h
3fef3958a95b8020caeb2feddbc5bb7b3074dcb38c205b6af74b67c562b9a74d  src/util/z3_exception.cpp
09356c28b4a280a7cf48d43e724396cbd3162db66949df033aebc5ad2bb5fbe9  src/util/cmd_context_types.h
6ef3f7c9d46d2d496f53f0d20ad5a66bc09b9df755997da2e0d9d53e2edc6db3  src/util/warning.cpp
d610cf65557aae3d4fbbc4963f3916694bb9ddf9661c6cb21083a68518e9d0b3  src/test/ex.cpp
```

Final targeted gates on frozen production/tests:

- Native all5 RUN/PASS (47059a).
- Checked race all5 RUN/PASS, exit0 (99053/e2d4cb).
- Fresh actual checked JS all5 named RUN/PASS, exit0 (af9330).
- Root package vet exit0 (ae5930).
- Checked WASI cross-compile exit0 (fa9c9e).
- Source/no-generics syntax audit exit0 (77096/2de1ba):2102 units/677 APIs,23 ported/41 partial/2038 unported.

Files are frozen for central review/combined integration gates and commit.
