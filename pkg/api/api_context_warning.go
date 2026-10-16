// Source: src/api/api_context.cpp:364-368.
package api

import "github.com/glycerine/g3/pkg/util/diagnostic"

// ToggleWarningMessages sets the process warning gate used by API callers.
func ToggleWarningMessages(enabled bool) {
	_ = diagnostic.BorrowProcessWarningService().Enable(enabled)
}
