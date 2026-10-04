// Licensed under the MIT license.
// Concrete reslimit specialization of src/util/event_handler.h/cancel_eh.h.
// Original author: Leonardo de Moura.
package g3

type utilEventCaller uint32

const (
	utilUnsetCaller utilEventCaller = iota
	utilControlCCaller
	utilTimeoutCaller
	utilResourceLimitCaller
	utilAPIInterruptCaller
)

// This cold scope borrows its resource owner. Invoke and Close must be serialized
// by the caller, as in the source event handler. The underlying cancel counter
// supports asynchronous cancellation. Do not copy an active handler or close its
// owner first. The handler has no interface dispatch or type parameters.
type utilCancelHandler struct {
	owner               *utilReslimit
	caller              utilEventCaller
	wasCanceled, closed bool
}

func newUtilCancelHandler(r *utilReslimit) utilCancelHandler {
	r.checkOpen()
	return utilCancelHandler{owner: r}
}
func (h *utilCancelHandler) callerID() utilEventCaller { return h.caller }
func (h *utilCancelHandler) canceled() bool            { return h.wasCanceled }
func (h *utilCancelHandler) invoke(caller utilEventCaller) {
	if h.closed {
		panic("g3: closed cancel handler")
	}
	if !h.wasCanceled {
		h.caller = caller
		h.wasCanceled = true
		h.owner.incCancel()
	}
}

// Explicit deferred close replaces the single source RAII destructor. Repeated
// close is harmless and does not decrement another scope's cancellation.
func (h *utilCancelHandler) close() {
	if h.closed {
		return
	}
	if h.wasCanceled {
		h.owner.decCancel()
	}
	h.closed = true
}
