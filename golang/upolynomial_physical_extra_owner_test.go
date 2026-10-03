// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"fmt"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
)

// Every entry is supplied by its actual caller owner; storage directories do
// not supply missing lexical roots.
type upolyPhysicalExtraOwners struct {
	Services         []*mpnum.UPolynomialFactorizationService
	Integers         []*mpnum.Int
	Rationals        []*mpnum.Rational
	RationalManagers []*mpnum.RationalManager
	BinaryValues     []*mpnum.BinaryRational
	BinaryManagers   []*mpnum.BinaryRationalManager
	BinaryVectors    []*mpnum.BinaryRationalVector
	Sequences        []*mpnum.UPolynomialSequence
}

func upolyPhysicalExtraNumerals(extras []upolyPhysicalExtraOwners, addInt func(reflect.Value)) {
	for _, x := range extras {
		for _, n := range x.Integers {
			addInt(reflect.ValueOf(n).Elem())
		}
		for _, r := range x.Rationals {
			v := reflect.ValueOf(r).Elem()
			addInt(v.FieldByName("num"))
			addInt(v.FieldByName("den"))
		}
		for _, m := range x.RationalManagers {
			v := reflect.ValueOf(m).Elem()
			for _, f := range []string{"nTmp", "addTmp1", "addTmp2"} {
				addInt(v.FieldByName(f))
			}
			for _, f := range []string{"addMulTmp", "ltTmp1", "ltTmp2"} {
				r := v.FieldByName(f)
				addInt(r.FieldByName("num"))
				addInt(r.FieldByName("den"))
			}
		}
		for _, b := range x.BinaryValues {
			addInt(reflect.ValueOf(b).Elem().FieldByName("num"))
		}
		for _, m := range x.BinaryManagers {
			v := reflect.ValueOf(m).Elem()
			for _, f := range []string{"tmp", "tmp2", "selectIntTmp1", "selectIntTmp2", "selectSmallTmp", "divTmp1", "divTmp2", "divTmp3"} {
				addInt(v.FieldByName(f))
			}
			for _, f := range []string{"addMulTmp", "selectSmallTmp1", "selectSmallTmp2"} {
				addInt(v.FieldByName(f).FieldByName("num"))
			}
		}
		for _, b := range x.BinaryVectors {
			state := reflect.ValueOf(b).Elem().FieldByName("state")
			values := state.FieldByName("values")
			for i := uint64(0); i < state.FieldByName("initialized").Uint(); i++ {
				addInt(values.Index(int(i)).FieldByName("num"))
			}
		}
	}
}

func upolyPhysicalExtraVectors(extras []upolyPhysicalExtraOwners, add func(reflect.Value, reflect.Value, string) error) error {
	for _, x := range extras {
		for i, s := range x.Sequences {
			v := reflect.ValueOf(s).Elem()
			if e := add(v.FieldByName("manager").Elem(), v.FieldByName("state").FieldByName("coefficients"), fmt.Sprintf("sequence%d", i)); e != nil {
				return e
			}
		}
	}
	return nil
}
