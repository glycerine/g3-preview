// Test reflection keeps the original source field spelling while accepting exported Go names.
package smtcore

import "reflect"

func testExportedFieldNames(name string) (string, string) {
	switch name {
	case "HeaderField":
		return "Header", ""
	case "InvocationField":
		return "Invocation", ""
	case "NextField":
		return "Next", ""
	case "OwnerField":
		return "Owner", ""
	case "PendingField":
		return "Pending", ""
	case "ReceiptField":
		return "Receipt", ""
	case "SelfField":
		return "Self", ""
	case "ValueField":
		return "Value", ""
	case "acquired":
		return "Acquired", ""
	case "active":
		return "Active", "ActiveField"
	case "addActive":
		return "AddActive", ""
	case "addMul":
		return "AddMul", ""
	case "addMulInvocation":
		return "AddMulInvocation", ""
	case "addMulTmp":
		return "AddMulTmp", ""
	case "addOne":
		return "AddOne", ""
	case "addTmp1":
		return "AddTmp1", ""
	case "addTmp2":
		return "AddTmp2", ""
	case "addTwo":
		return "AddTwo", ""
	case "allocator":
		return "Allocator", ""
	case "arg":
		return "Arg", ""
	case "arrayCardinality":
		return "ArrayCardinality", ""
	case "atoms":
		return "Atoms", ""
	case "backing":
		return "Backing", ""
	case "bindings":
		return "Bindings", ""
	case "block":
		return "Block", ""
	case "blocks":
		return "Blocks", ""
	case "bodyCause":
		return "BodyCause", ""
	case "bodySucceeded":
		return "BodySucceeded", ""
	case "boolDivides":
		return "BoolDivides", ""
	case "boolDividesInvocation":
		return "BoolDividesInvocation", ""
	case "budget":
		return "Budget", "BudgetField"
	case "buffer":
		return "Buffer", ""
	case "buffers":
		return "Buffers", ""
	case "cache":
		return "Cache", ""
	case "cacheBytes":
		return "CacheBytes", ""
	case "caller":
		return "Caller", ""
	case "callerExecuting":
		return "CallerExecuting", ""
	case "callerSelf":
		return "CallerSelf", ""
	case "capacity":
		return "Capacity", ""
	case "cause":
		return "Cause", ""
	case "cell":
		return "Cell", ""
	case "cells":
		return "Cells", ""
	case "charFormatting":
		return "CharFormatting", ""
	case "character":
		return "Character", ""
	case "child":
		return "Child", ""
	case "childFrame":
		return "ChildFrame", ""
	case "children":
		return "Children", ""
	case "clauses":
		return "Clauses", ""
	case "cleanupCause":
		return "CleanupCause", "CleanupCauseField"
	case "cleanupPending":
		return "CleanupPending", ""
	case "closed":
		return "Closed", "ClosedField"
	case "closing":
		return "Closing", ""
	case "compareActive":
		return "CompareActive", ""
	case "compareOne":
		return "CompareOne", ""
	case "compareTwo":
		return "CompareTwo", ""
	case "completed":
		return "Completed", "CompletedField"
	case "config":
		return "Config", ""
	case "constantDeclarations":
		return "ConstantDeclarations", ""
	case "constants":
		return "Constants", ""
	case "constructed":
		return "Constructed", ""
	case "constructorUnwound":
		return "ConstructorUnwound", ""
	case "context":
		return "Context", "ContextField"
	case "copyAllocation":
		return "CopyAllocation", ""
	case "core":
		return "CoreField", ""
	case "count":
		return "Count", ""
	case "current":
		return "Current", ""
	case "debugFreeIndices":
		return "DebugFreeIndices", ""
	case "debugRefCount":
		return "DebugRefCount", ""
	case "decimal":
		return "Decimal", ""
	case "den":
		return "Den", ""
	case "denom":
		return "Denom", ""
	case "derived":
		return "Derived", ""
	case "descriptors":
		return "Descriptors", ""
	case "diagnostics":
		return "Diagnostics", ""
	case "display":
		return "Display", ""
	case "div":
		return "Div", ""
	case "divActive":
		return "DivActive", ""
	case "divTmp":
		return "DivTmp", ""
	case "divTmp1":
		return "DivTmp1", ""
	case "divTmp2":
		return "DivTmp2", ""
	case "divTmp3":
		return "DivTmp3", ""
	case "dividesNumeral":
		return "DividesNumeral", ""
	case "division":
		return "Division", ""
	case "divisionRaw":
		return "DivisionRaw", ""
	case "done":
		return "Done", ""
	case "endEntered":
		return "EndEntered", ""
	case "ended":
		return "Ended", ""
	case "entries":
		return "Entries", ""
	case "executing":
		return "Executing", ""
	case "exprIDs":
		return "ExprIDs", ""
	case "extendedRaw":
		return "ExtendedRaw", ""
	case "failed":
		return "Failed", ""
	case "failedNullRef":
		return "FailedNullRef", ""
	case "failedRef":
		return "FailedRef", ""
	case "failedTags":
		return "FailedTags", ""
	case "file":
		return "File", ""
	case "fileBad":
		return "FileBad", ""
	case "filePosition":
		return "FilePosition", ""
	case "first":
		return "First", ""
	case "format":
		return "Format", ""
	case "formatScopes":
		return "FormatScopes", ""
	case "formatScratch":
		return "FormatScratch", ""
	case "formattedSetHistories":
		return "FormattedSetHistories", ""
	case "formatting":
		return "Formatting", ""
	case "frame":
		return "Frame", ""
	case "free":
		return "Free", ""
	case "functions":
		return "Functions", ""
	case "gcdRaw":
		return "GcdRaw", ""
	case "generation":
		return "Generation", ""
	case "globalDescriptors":
		return "GlobalDescriptors", ""
	case "header":
		return "Header", ""
	case "headerBytes":
		return "HeaderBytes", ""
	case "headerCharged":
		return "HeaderCharged", ""
	case "heldGeneration":
		return "HeldGeneration", ""
	case "heldSlot":
		return "HeldSlot", ""
	case "index":
		return "Index", ""
	case "infIntMinusOne":
		return "InfIntMinusOneField", ""
	case "infIntOne":
		return "InfIntOneField", ""
	case "infIntZero":
		return "InfIntZeroField", ""
	case "infMinusOne":
		return "InfMinusOneField", ""
	case "infOne":
		return "InfOneField", ""
	case "infZero":
		return "InfZeroField", ""
	case "initialized":
		return "Initialized", ""
	case "inputCalls":
		return "InputCalls", ""
	case "intMin":
		return "IntMin", ""
	case "invTmp1":
		return "InvTmp1", ""
	case "invTmp2":
		return "InvTmp2", ""
	case "invTmp3":
		return "InvTmp3", ""
	case "invocation":
		return "Invocation", "InvocationField"
	case "issued":
		return "Issued", ""
	case "k":
		return "KField", ""
	case "key":
		return "Key", ""
	case "kind":
		return "Kind", ""
	case "lexical":
		return "Lexical", ""
	case "lifecycle":
		return "Lifecycle", ""
	case "limit":
		return "Limit", ""
	case "limits":
		return "Limits", ""
	case "live":
		return "Live", ""
	case "locals":
		return "Locals", ""
	case "lowLevel":
		return "LowLevel", ""
	case "lower":
		return "Lower", ""
	case "ltTmp1":
		return "LtTmp1", ""
	case "ltTmp2":
		return "LtTmp2", ""
	case "manager":
		return "Manager", ""
	case "memory":
		return "Memory", ""
	case "message":
		return "MessageField", ""
	case "minusOne":
		return "MinusOneField", ""
	case "mode":
		return "Mode", ""
	case "moduleDescriptions":
		return "ModuleDescriptions", ""
	case "moduleDescriptors":
		return "ModuleDescriptors", ""
	case "moduleParams":
		return "ModuleParams", ""
	case "mpfParent":
		return "MpfParent", ""
	case "mulCurr":
		return "MulCurr", ""
	case "mulMax":
		return "MulMax", ""
	case "mulMin":
		return "MulMin", ""
	case "nTmp":
		return "NTmp", ""
	case "names":
		return "Names", ""
	case "natural":
		return "Natural", ""
	case "next":
		return "Next", ""
	case "nodes":
		return "Nodes", ""
	case "norm":
		return "Norm", ""
	case "normActive":
		return "NormActive", ""
	case "normalization":
		return "Normalization", ""
	case "normalizationExecuting":
		return "NormalizationExecuting", ""
	case "num":
		return "Num", ""
	case "numeral":
		return "Numeral", ""
	case "numeralAcquired":
		return "NumeralAcquired", ""
	case "numerals":
		return "NumeralsField", ""
	case "numeric":
		return "Numeric", ""
	case "object":
		return "Object", ""
	case "offset":
		return "Offset", ""
	case "one":
		return "OneField", ""
	case "original":
		return "Original", ""
	case "outOfMemory":
		return "OutOfMemory", ""
	case "owned":
		return "Owned", ""
	case "owner":
		return "Owner", ""
	case "ownerBytes":
		return "OwnerBytes", ""
	case "ownerThread":
		return "OwnerThread", ""
	case "p":
		return "P", ""
	case "pages":
		return "Pages", ""
	case "parameters":
		return "Parameters", ""
	case "parent":
		return "Parent", ""
	case "parentContext":
		return "ParentContext", ""
	case "payload":
		return "Payload", ""
	case "pending":
		return "Pending", "PendingField"
	case "pendingCell":
		return "PendingCell", ""
	case "perfectSquare":
		return "PerfectSquare", ""
	case "perfectSquareGeneration":
		return "PerfectSquareGeneration", ""
	case "power":
		return "Power", ""
	case "powerActive":
		return "PowerActive", ""
	case "powerGeneration":
		return "PowerGeneration", ""
	case "powerRaw":
		return "PowerRaw", ""
	case "powerWrappers":
		return "PowerWrappers", ""
	case "present":
		return "PresentField", ""
	case "published":
		return "Published", ""
	case "q":
		return "Q", ""
	case "rat":
		return "Rat", ""
	case "rational":
		return "Rational", ""
	case "rationalSetHistories":
		return "RationalSetHistories", ""
	case "raw":
		return "Raw", ""
	case "rawArithmetic":
		return "RawArithmetic", ""
	case "rawDisplay":
		return "RawDisplay", ""
	case "rawGeneration":
		return "RawGeneration", ""
	case "receipt":
		return "Receipt", ""
	case "refCount":
		return "RefCount", ""
	case "references":
		return "References", ""
	case "refs":
		return "Refs", ""
	case "region":
		return "Region", ""
	case "register":
		return "Register", ""
	case "released":
		return "Released", ""
	case "requested":
		return "Requested", ""
	case "result":
		return "Result", ""
	case "retainedNext":
		return "RetainedNext", ""
	case "retired":
		return "Retired", ""
	case "rootRaw":
		return "RootRaw", ""
	case "roots":
		return "Roots", ""
	case "roundScope":
		return "RoundScope", ""
	case "roundSqrtScope":
		return "RoundSqrtScope", ""
	case "second":
		return "Second", ""
	case "selectIntTmp1":
		return "SelectIntTmp1", ""
	case "selectIntTmp2":
		return "SelectIntTmp2", ""
	case "selectSmallTmp":
		return "SelectSmallTmp", ""
	case "selectSmallTmp1":
		return "SelectSmallTmp1", ""
	case "selectSmallTmp2":
		return "SelectSmallTmp2", ""
	case "self":
		return "Self", ""
	case "setFrames":
		return "SetFrames", ""
	case "size":
		return "Size", ""
	case "slot":
		return "Slot", ""
	case "smt2Display":
		return "Smt2Display", ""
	case "sourceCause":
		return "SourceCause", ""
	case "sourceDelDone":
		return "SourceDelDone", ""
	case "sourceEnded":
		return "SourceEndedField", ""
	case "sourceFinalizeStage":
		return "SourceFinalizeStage", ""
	case "sourceFinalized":
		return "SourceFinalized", ""
	case "sourceFinalizing":
		return "SourceFinalizing", ""
	case "sourceFrames":
		return "SourceFrames", ""
	case "sourceIndex":
		return "SourceIndex", ""
	case "sourceStage":
		return "SourceStage", ""
	case "stage":
		return "Stage", ""
	case "stderr":
		return "StderrField", ""
	case "steps":
		return "Steps", ""
	case "stream":
		return "Stream", ""
	case "superseded":
		return "Superseded", ""
	case "symbols":
		return "Symbols", "SymbolsField"
	case "synchronized":
		return "Synchronized", ""
	case "table":
		return "Table", ""
	case "tag":
		return "Tag", ""
	case "tagRuntime":
		return "TagRuntime", ""
	case "tags":
		return "Tags", ""
	case "temporary":
		return "Temporary", ""
	case "text":
		return "Text", ""
	case "texts":
		return "Texts", ""
	case "tmp":
		return "Tmp", ""
	case "tmp2":
		return "Tmp2", ""
	case "traceFile":
		return "TraceFile", ""
	case "traceStream":
		return "TraceStream", ""
	case "transferred":
		return "Transferred", ""
	case "two64":
		return "Two64", ""
	case "u":
		return "U", ""
	case "universes":
		return "Universes", ""
	case "unpublished":
		return "Unpublished", ""
	case "upper":
		return "Upper", ""
	case "used":
		return "Used", ""
	case "val":
		return "Val", ""
	case "value":
		return "Value", ""
	case "valueCalls":
		return "ValueCalls", ""
	case "values":
		return "Values", ""
	case "variables":
		return "Variables", ""
	case "verification":
		return "Verification", ""
	case "verificationActive":
		return "VerificationActive", ""
	case "verificationInvocation":
		return "VerificationInvocation", ""
	case "words":
		return "Words", ""
	case "workspace":
		return "Workspace", ""
	case "z":
		return "Z", ""
	case "zero":
		return "ZeroField", ""
	}
	return "", ""
}
func testFieldBySourceName(v reflect.Value, name string) reflect.Value {
	f := v.FieldByName(name)
	if f.IsValid() {
		return f
	}
	a, b := testExportedFieldNames(name)
	if a != "" {
		if f = v.FieldByName(a); f.IsValid() {
			return f
		}
	}
	if b != "" {
		return v.FieldByName(b)
	}
	return f
}
func testTypeFieldBySourceName(t reflect.Type, name string) (reflect.StructField, bool) {
	f, ok := t.FieldByName(name)
	if ok {
		return f, ok
	}
	a, b := testExportedFieldNames(name)
	if a != "" {
		if f, ok = t.FieldByName(a); ok {
			return f, ok
		}
	}
	if b != "" {
		return t.FieldByName(b)
	}
	return f, ok
}
