// Source: src/math/subpaving/subpaving_types.h:30-31 and the concrete MPFF
// throws in subpaving.cpp:213, subpaving_t_def.h:237,570.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrSubpavingFailure = errors.New("subpaving: source exception")

type sourceFailure struct{ message string }

func (e *sourceFailure) Error() string { return e.message }
func (e *sourceFailure) Unwrap() error { return ErrSubpavingFailure }

// The source fpoint wrapper catches only its numeral-manager exception and
// replaces it with a message-free subpaving::exception.
func sourceFastFailure(err error) error {
	if errors.Is(err, numeral.ErrFastFloatDomain) {
		return ErrSubpavingFailure
	}
	return err
}
