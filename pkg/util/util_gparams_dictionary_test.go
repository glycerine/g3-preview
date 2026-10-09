package util

import "testing"

func TestGparamsDictionariesUseRequiredGoOrdering(t *testing.T) {
	var descriptors gparamsDescriptorDictionary
	descriptors.init()
	for i := uint32(0); i < 8; i++ {
		descriptors.set(NumericalSymbol(i), i+1)
	}
	descriptors.set(NumericalSymbol(2), 99)
	if got, ok := descriptors.get(NumericalSymbol(2)); !ok || got != 99 || descriptors.size() != 8 {
		t.Fatal(got, ok, descriptors.size())
	}
	want := uint32(0)
	for key := range descriptors.values.All() {
		if key != NumericalSymbol(want) {
			t.Fatal("descriptor registration order", key, want)
		}
		want++
	}

	var refs gparamsRefDictionary
	refs.init()
	refs.set(NumericalSymbol(1), 4)
	refs.set(NumericalSymbol(1), 9)
	if got, ok := refs.get(NumericalSymbol(1)); !ok || got != 9 || refs.size() != 1 {
		t.Fatal(got, ok, refs.size())
	}
	refs.reset()
	if refs.size() != 0 {
		t.Fatal(refs.size())
	}

	var descriptions gparamsDescriptionDictionary
	descriptions.init()
	text := GlobalParamCString{Offset: 7, Present: true}
	descriptions.set(NumericalSymbol(3), text)
	if got, ok := descriptions.get(NumericalSymbol(3)); !ok || got != text {
		t.Fatal(got, ok)
	}
}
