// Native numeral output boundary for retained numeric stream consumers.
package text

func (out *NumericTextStream) AppendNumeralText(text string) error {
	return out.AppendCString(text)
}
