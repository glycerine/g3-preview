// Source: src/math/polynomial/upolynomial.cpp; src/util/mpq.h (ownership audit adaptation).
package polynomial

import (
	"fmt"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Every entry is supplied by its actual caller owner; storage directories do
// not supply missing lexical roots.
type paramsPhysicalValueOwner struct {
	Context *util.ParamsContext
	Value   *util.ParamsRef
}

type paramsPhysicalRationalOwner struct {
	Service *util.RationalService
	Value   *util.OwnedRational
}

type upolyPhysicalExtraOwners struct {
	CallerBlocks         []umemory.Block // actual directly returned caller allocations only
	ASTManagers          []*ast.ASTManager
	DebugContexts        []*util.DebugContext
	ParamsRationalValues []paramsPhysicalRationalOwner
	ParamsValues         []paramsPhysicalValueOwner
	RationalServices     []*util.RationalService
	SexprManagers        []*util.SexprManager
	Services             []*UPolynomialFactorizationService
	Integers             []*util.Int
	Rationals            []*util.Rational
	RationalManagers     []*util.RationalManager
	BinaryValues         []*util.BinaryRational
	BinaryManagers       []*util.BinaryRationalManager
	BinaryVectors        []*BinaryRationalVector
	Sequences            []*UPolynomialSequence
}

func upolyPhysicalExtraNumerals(extras []upolyPhysicalExtraOwners, addInt func(reflect.Value)) {
	for _, x := range extras {

		addRat := func(v reflect.Value) {
			addInt(testFieldBySourceName(v, "num"))
			addInt(testFieldBySourceName(v, "den"))
		}
		addManager := func(v reflect.Value) {
			for _, f := range []string{"nTmp", "addTmp1", "addTmp2"} {
				addInt(testFieldBySourceName(v, f))
			}
			for _, f := range []string{"addMulTmp", "ltTmp1", "ltTmp2"} {
				addRat(testFieldBySourceName(v, f))
			}
			scope := testFieldBySourceName(v, "lexical")
			for i := 0; i < int(testFieldBySourceName(scope, "count").Uint()); i++ {
				addRat(testFieldBySourceName(scope, "roots").Index(i))
			}
			raw := testFieldBySourceName(v, "rawArithmetic")
			if testFieldBySourceName(raw, "divActive").Bool() {
				addInt(testFieldBySourceName(raw, "div"))
			}
			if testFieldBySourceName(raw, "powerActive").Bool() {
				addRat(testFieldBySourceName(raw, "power"))
			}
			if testFieldBySourceName(raw, "addActive").Bool() {
				addInt(testFieldBySourceName(raw, "addOne"))
				addInt(testFieldBySourceName(raw, "addTwo"))
			}
			if testFieldBySourceName(raw, "compareActive").Bool() {
				addRat(testFieldBySourceName(raw, "compareOne"))
				addRat(testFieldBySourceName(raw, "compareTwo"))
			}
			if testFieldBySourceName(raw, "normActive").Bool() {
				addInt(testFieldBySourceName(raw, "norm"))
			}
		}
		// Only the explicitly supplied owner supplies roots. Its initialized
		// placement records and acquired lexical value are concrete acquisition
		// fields, never discoveries from the numeric allocator directory.
		for _, owner := range x.ASTManagers {
			if owner == nil || owner.Self != owner || owner.Closed {
				panic(ast.ErrASTHandle)
			}
			for _, page := range owner.Parameters {
				if page == nil {
					continue
				}
				for i := range page {
					value := &page[i]
					if value.Live && value.Kind == ast.ASTParameterRational {
						addRat(testFieldBySourceName(reflect.ValueOf(&value.Rational).Elem(), "raw"))
					}
				}
			}
			for f := owner.ArrayCardinality; f != nil; f = f.Next {
				if f.Self != f || f.Owner != owner || f.Invocation == 0 {
					panic(ast.ErrASTHandle)
				}
				for i, owned := range f.Acquired {
					if owned {
						addRat(testFieldBySourceName(reflect.ValueOf(&f.Values[i]).Elem(), "raw"))
					}
				}
			}
			if owner.LowLevel.Active && owner.LowLevel.NumeralAcquired {
				addRat(testFieldBySourceName(reflect.ValueOf(&owner.LowLevel.Numeral).Elem(), "raw"))
			}
		}
		for _, owner := range x.ParamsRationalValues {
			if owner.Service == nil || owner.Value == nil {
				panic(util.ErrRationalServiceOwner)
			}
			service := reflect.ValueOf(owner.Service).Elem()
			value := reflect.ValueOf(owner.Value).Elem()
			if testFieldBySourceName(service, "self").Pointer() != reflect.ValueOf(owner.Service).Pointer() || testFieldBySourceName(value, "ownerID").Uint() == 0 || testFieldBySourceName(value, "ownerID").Uint() != testFieldBySourceName(service, "ownerID").Uint() {
				panic(util.ErrRationalServiceOwner)
			}
			addRat(testFieldBySourceName(value, "raw"))
		}
		paramsContexts := make(map[*util.ParamsContext]bool)
		for _, owner := range x.ParamsValues {
			if owner.Context == nil || owner.Value == nil {
				panic(util.ErrParamsOwner)
			}
			c, v := owner.Context, owner.Value
			if !paramsContexts[c] {
				paramsContexts[c] = true
				for i, frame := range c.SetFrames {
					if !frame.Active {
						continue
					}
					if !frame.Acquired || frame.Invocation == 0 || i >= len(c.Blocks) || frame.Generation != c.Blocks[i].Generation || frame.Next.Kind != util.ParamNumeral {
						panic(util.ErrParamsOwner)
					}
					raw := reflect.ValueOf(&frame.Next.Rat).Elem()
					if testFieldBySourceName(raw, "ownerID").Uint() != testFieldBySourceName(reflect.ValueOf(c.NumeralsField).Elem(), "ownerID").Uint() {
						panic(util.ErrRationalServiceOwner)
					}
					addRat(testFieldBySourceName(raw, "raw"))
				}
				for _, frame := range c.SourceFrames {
					if !frame.Active {
						continue
					}
					if frame.Invocation == 0 || frame.HeldSlot == 0 || int(frame.HeldSlot) > len(c.Blocks) {
						panic(util.ErrParamsOwner)
					}
					held := &c.Blocks[frame.HeldSlot-1]
					if !held.Live || held.Generation != frame.HeldGeneration {
						panic(util.ErrParamsOwner)
					}
					for i := uint32(0); i < held.Size; i++ {
						entry := c.Entry(held, i)
						if entry.Kind == util.ParamNumeral {
							addRat(testFieldBySourceName(reflect.ValueOf(&entry.Rat).Elem(), "raw"))
						}
					}
				}
			}
			if c.Self != c || c.Closed || v.Owner != c.Owner {
				panic(util.ErrParamsOwner)
			}
			if v.Slot == 0 {
				continue
			}
			if int(v.Slot) > len(c.Blocks) {
				panic(util.ErrParamsOwner)
			}
			b := &c.Blocks[v.Slot-1]
			if !b.Live || b.Generation != v.Generation {
				panic(util.ErrParamsOwner)
			}
			// refs==0 is allowed only as a reached stopped destructor owner. The
			// explicitly supplied owning value, not the count, names its actual block.
			for i := uint32(0); i < b.Size; i++ {
				e := c.Entry(b, i)
				if e.Kind == util.ParamNumeral {
					addRat(testFieldBySourceName(reflect.ValueOf(&e.Rat).Elem(), "raw"))
				}
			}
		}
		for _, s := range x.RationalServices {
			v := reflect.ValueOf(s).Elem()
			addManager(testFieldBySourceName(v, "manager").Elem())
			for _, f := range []string{"zero", "one", "minusOne"} {
				addRat(testFieldBySourceName(v, f))
			}
			cache := testFieldBySourceName(v, "cache")
			for i := 0; i < cache.Len(); i++ {
				addRat(cache.Index(i))
			}
			scope := testFieldBySourceName(v, "lexical")
			for i := 0; i < int(testFieldBySourceName(scope, "count").Uint()); i++ {
				addRat(testFieldBySourceName(scope, "roots").Index(i))
			}
		}
		for _, s := range x.SexprManagers {
			for _, page := range s.Pages {
				for i := range page.Nodes {
					r := &page.Nodes[i]
					if r.Live && (r.Kind == util.SexprNumeral || r.Kind == util.SexprBVNumeral) {
						addRat(testFieldBySourceName(reflect.ValueOf(r.Numeral), "raw"))
					}
				}
			}
		}
		for _, n := range x.Integers {
			addInt(reflect.ValueOf(n).Elem())
		}
		for _, r := range x.Rationals {
			v := reflect.ValueOf(r).Elem()
			addInt(testFieldBySourceName(v, "num"))
			addInt(testFieldBySourceName(v, "den"))
		}
		for _, m := range x.RationalManagers {
			addManager(reflect.ValueOf(m).Elem())
		}
		for _, b := range x.BinaryValues {
			addInt(testFieldBySourceName(reflect.ValueOf(b).Elem(), "num"))
		}
		for _, m := range x.BinaryManagers {
			v := reflect.ValueOf(m).Elem()
			for _, f := range []string{"tmp", "tmp2", "selectIntTmp1", "selectIntTmp2", "selectSmallTmp", "divTmp1", "divTmp2", "divTmp3"} {
				addInt(testFieldBySourceName(v, f))
			}
			for _, f := range []string{"addMulTmp", "selectSmallTmp1", "selectSmallTmp2"} {
				addInt(testFieldBySourceName(testFieldBySourceName(v, f), "num"))
			}
		}
		for _, b := range x.BinaryVectors {
			state := testFieldBySourceName(reflect.ValueOf(b).Elem(), "state")
			values := testFieldBySourceName(state, "values")
			for i := 0; i < values.Len(); i++ {
				addInt(testFieldBySourceName(values.Index(int(i)), "num"))
			}
		}
	}
}

func upolyPhysicalExtraVectors(extras []upolyPhysicalExtraOwners, add func(reflect.Value, reflect.Value, string) error) error {
	for _, x := range extras {
		for i, s := range x.Sequences {
			v := reflect.ValueOf(s).Elem()
			if e := add(testFieldBySourceName(v, "manager").Elem(), testFieldBySourceName(testFieldBySourceName(v, "state"), "coefficients"), fmt.Sprintf("sequence%d", i)); e != nil {
				return e
			}
		}
	}
	return nil
}
