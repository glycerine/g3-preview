package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSexprPolynomialNativeFramesNestedBIGOwnership(t *testing.T) {
	c, sx, sy, q, p, out := sxPolyOwner(t)
	x := sxPolySym(t, sx, sy, "x")
	expr := sxPolyNum(t, sx, q, "1208925819614629174706177") // 2^80 + 1
	const count = 257
	for i := 0; i < count; i++ {
		expr = sxPolyApp(t, sx, sy, "+", x, expr)
	}
	z := p.IntegerManager()
	var expected util.Int
	sxMust(t, z.SetLimbs(&expected, []uint32{1, 0, 1 << 16}))
	defer func() { sxMust(t, z.Del(&expected)) }()
	for run := 0; run < 2; run++ {
		sxMust(t, c.Convert(expr, out))
		if n, e := p.Size(out); e != nil || n != 2 {
			t.Fatal("independent nested sum degree", n, e)
		}
		var constant, coefficient util.Int
		sxMust(t, p.ReadCoefficient(out, 0, &constant))
		sxMust(t, p.ReadCoefficient(out, 1, &coefficient))
		same, e := z.Compare(&constant, &expected)
		sxMust(t, e)
		value, e := z.Int32(&coefficient)
		sxMust(t, e)
		if same != 0 || value != count || constant.NativeCellToken() == 0 {
			t.Fatal("nested sum lost BIG constant or x coefficient", same, value)
		}
		sxMust(t, z.Del(&constant))
		sxMust(t, z.Del(&coefficient))
		if len(c.frames) != count || c.depth != 0 || c.cause != nil {
			t.Fatal("native frame extent or cleanup", len(c.frames), c.depth, c.cause)
		}
		for i, frame := range c.frames {
			if frame.live || frame.argument != (UPolynomial{}) {
				t.Fatal("completed lexical frame retained owning vector", i, frame)
			}
		}
		upolyHenselAuditPhysical(t, []*UPolynomialManager{p}, nil, upolyPhysicalExtraOwners{Integers: []*util.Int{&expected}, RationalServices: []*util.RationalService{q}, SexprManagers: []*util.SexprManager{sx}})
	}
}
