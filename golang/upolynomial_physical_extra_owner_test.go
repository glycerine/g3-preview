// Source: src/math/polynomial/upolynomial.cpp; src/util/mpq.h (ownership audit adaptation). Licensed under the MIT license.
package g3

import (
	"fmt"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
)

// Every entry is supplied by its actual caller owner; storage directories do
// not supply missing lexical roots.
type paramsPhysicalValueOwner struct {
	Context *ParamsContext
	Value   *ParamsRef
}

type upolyPhysicalExtraOwners struct {
	ParamsValues     []paramsPhysicalValueOwner
	RationalServices []*mpnum.RationalService
	SexprManagers    []*SexprManager
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

		addRat := func(v reflect.Value) { addInt(v.FieldByName("num")); addInt(v.FieldByName("den")) }
		addManager := func(v reflect.Value) {
			for _, f := range []string{"nTmp", "addTmp1", "addTmp2"} {
				addInt(v.FieldByName(f))
			}
			for _, f := range []string{"addMulTmp", "ltTmp1", "ltTmp2"} {
				addRat(v.FieldByName(f))
			}
			scope := v.FieldByName("lexical")
			for i := 0; i < int(scope.FieldByName("count").Uint()); i++ {
				addRat(scope.FieldByName("roots").Index(i))
			}
			raw := v.FieldByName("rawArithmetic")
			if raw.FieldByName("divActive").Bool() {
				addInt(raw.FieldByName("div"))
			}
			if raw.FieldByName("powerActive").Bool() {
				addRat(raw.FieldByName("power"))
			}
			if raw.FieldByName("addActive").Bool() {
				addInt(raw.FieldByName("addOne"))
				addInt(raw.FieldByName("addTwo"))
			}
			if raw.FieldByName("compareActive").Bool() {
				addRat(raw.FieldByName("compareOne"))
				addRat(raw.FieldByName("compareTwo"))
			}
			if raw.FieldByName("normActive").Bool() {
				addInt(raw.FieldByName("norm"))
			}
		}
		for _, owner := range x.ParamsValues {
			if owner.Context == nil || owner.Value == nil {
				panic(ErrParamsOwner)
			}
			c, v := owner.Context, owner.Value
			if c.self != c || c.closed || v.owner != c.owner {
				panic(ErrParamsOwner)
			}
			if v.slot == 0 {
				continue
			}
			if int(v.slot) > len(c.blocks) {
				panic(ErrParamsOwner)
			}
			b := &c.blocks[v.slot-1]
			if !b.live || b.generation != v.generation {
				panic(ErrParamsOwner)
			}
			// refs==0 is allowed only as a reached stopped destructor owner. The
			// explicitly supplied owning value, not the count, names its actual block.
			for i := uint32(0); i < b.size; i++ {
				e := c.entry(b, i)
				if e.kind == ParamNumeral {
					addRat(reflect.ValueOf(&e.rat).Elem().FieldByName("raw"))
				}
			}
		}
		for _, s := range x.RationalServices {
			v := reflect.ValueOf(s).Elem()
			addManager(v.FieldByName("manager").Elem())
			for _, f := range []string{"zero", "one", "minusOne"} {
				addRat(v.FieldByName(f))
			}
			cache := v.FieldByName("cache")
			for i := 0; i < cache.Len(); i++ {
				addRat(cache.Index(i))
			}
			scope := v.FieldByName("lexical")
			for i := 0; i < int(scope.FieldByName("count").Uint()); i++ {
				addRat(scope.FieldByName("roots").Index(i))
			}
		}
		for _, s := range x.SexprManagers {
			for _, page := range s.pages {
				for i := range page.nodes {
					r := &page.nodes[i]
					if r.live && (r.kind == SexprNumeral || r.kind == SexprBVNumeral) {
						addRat(reflect.ValueOf(r.numeral).FieldByName("raw"))
					}
				}
			}
		}
		for _, n := range x.Integers {
			addInt(reflect.ValueOf(n).Elem())
		}
		for _, r := range x.Rationals {
			v := reflect.ValueOf(r).Elem()
			addInt(v.FieldByName("num"))
			addInt(v.FieldByName("den"))
		}
		for _, m := range x.RationalManagers {
			addManager(reflect.ValueOf(m).Elem())
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
