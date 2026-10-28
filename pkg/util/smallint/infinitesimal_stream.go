package smallint

// Source: src/util/inf_s_integer.h:250-262.

import utext "github.com/glycerine/g3/pkg/util/text"

// AppendTo retains the source's separate ostream insertions. A field width is
// consumed by the first integer for a finite value, or by "(" otherwise.
func (n Infinitesimal) AppendTo(out *utext.NumericTextStream) error {
	if out == nil {
		return utext.ErrNumericStreamOwner
	}
	if n.epsilon == 0 {
		return out.AppendInt32(int32(n.whole))
	}
	if err := out.AppendCString("("); err != nil {
		return err
	}
	if err := out.AppendInt32(int32(n.whole)); err != nil {
		return err
	}
	if n.epsilon < 0 {
		if err := out.AppendCString(" -e*"); err != nil {
			return err
		}
	} else if err := out.AppendCString(" +e*"); err != nil {
		return err
	}
	if err := out.AppendInt32(int32(n.epsilon)); err != nil {
		return err
	}
	return out.AppendCString(")")
}
