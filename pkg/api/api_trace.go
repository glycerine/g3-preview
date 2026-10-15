// Source: src/api/api_context.cpp:408-419; src/api/z3_api.h:5295-5305.
package api

import "github.com/glycerine/g3/pkg/util/diagnostic"

// EnableTrace changes the process trace tags shared by shell, parameters and
// retained solver owners. TraceTags applies the source C-string boundary and
// copies the tag into its Go-owned map.
func EnableTrace(tag string) {
	diagnostic.BorrowProcessTraceTags().Enable(tag)
}

// DisableTrace removes a tag from the same process trace owner.
func DisableTrace(tag string) {
	diagnostic.BorrowProcessTraceTags().Disable(tag)
}
