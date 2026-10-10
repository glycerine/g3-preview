// Native numeral output boundary for retained numeric stream consumers.
package util

func (out *NumericTextStream) AppendNumeralText(text string) error {
	return out.AppendCString(text)
}
