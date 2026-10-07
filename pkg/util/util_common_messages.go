// Source: src/util/common_msgs.h:22-39; src/util/common_msgs.cpp:21-27.
package util

// Each variable represents both the source static pointer binding and its
// Z3_*_MSG macro alias. Source pointers may be reassigned; a reached message
// lookup reads this shared binding. Go strings retain immutable message bytes.
var (
	Z3CanceledMessage          = "canceled"
	Z3MaxMemoryMessage         = "max. memory exceeded"
	Z3MaxScopesMessage         = "max. scopes exceeded"
	Z3MaxStepsMessage          = "max. steps exceeded"
	Z3MaxFramesMessage         = "max. frames exceeded"
	Z3NoProofGenerationMessage = "component does not support proof generation"
	Z3MaxResourceMessage       = "max. resource limit exceeded"
)
