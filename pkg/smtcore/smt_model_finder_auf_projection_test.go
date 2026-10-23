package smtcore

import (
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
)

func TestSMTModelFinderAUFMonotoneProjectionNumericOrderAndFixedPoints(t *testing.T) {
	for _, domain := range []string{"integer", "real", "unsigned-bit-vector", "signed-bit-vector"} {
		t.Run(domain, func(t *testing.T) {
			m, config := smtModelGeneratorFixture(t)
			solver, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			proto, err := protomodel.NewProtoModel(m, config)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, proto.Close()) })
			makeNumeral := func(text string) ast.ASTHandle {
				var h ast.ASTHandle
				var err error
				switch domain {
				case "integer":
					h, err = m.ArithmeticIntegerNumeralText(text)
				case "real":
					h, err = m.ArithmeticRealNumeralText(text)
				default:
					_, err = m.RegisterBitVectorPlugin()
					astTestMust(t, err)
					v, e := strconv.ParseUint(text, 10, 32)
					astTestMust(t, e)
					h, err = m.BitVectorNumeralUnsigned(uint32(v), 4)
				}
				astTestMust(t, err)
				return h
			}
			samples := []string{"10", "0", "5"}
			inputs := []string{"-3", "0", "4", "5", "6", "9", "10", "20"}
			want := []string{"0", "0", "0", "5", "5", "5", "10", "10"}
			if domain == "real" {
				samples = []string{"10", "0", "5/2"}
				inputs = []string{"-1/2", "0", "2", "5/2", "7/2", "10", "101/10"}
				want = []string{"0", "0", "0", "5/2", "5/2", "10", "10"}
			} else if domain == "unsigned-bit-vector" {
				samples = []string{"15", "3", "8"}
				inputs = []string{"0", "3", "7", "8", "14", "15"}
				want = []string{"3", "3", "3", "8", "8", "15"}
			} else if domain == "signed-bit-vector" {
				// Four-bit encodings 8 and 15 denote -8 and -1.
				samples = []string{"3", "8", "15"}
				inputs = []string{"8", "14", "15", "0", "2", "3", "7"}
				want = []string{"8", "8", "15", "15", "15", "3", "3"}
			}
			first := makeNumeral(samples[0])
			sort, err := m.GetSort(first)
			astTestMust(t, err)
			graph := newSMTModelFinderAUFGraph(solver.kernel.internalizer.quantifierManager.modelFinder)
			graph.model = proto
			t.Cleanup(func() { astTestMust(t, graph.close()) })
			node, err := newSMTModelFinderAUFNode(0, sort)
			astTestMust(t, err)
			graph.nodes = append(graph.nodes, node)
			astTestMust(t, node.makeInstantiationSet(m))
			astTestMust(t, node.setMono())
			if domain == "signed-bit-vector" {
				astTestMust(t, node.setSigned())
			}
			for _, text := range samples {
				_, err := node.insert(makeNumeral(text), 0)
				astTestMust(t, err)
			}
			astTestMust(t, graph.makeMonotoneProjection(node))
			if node.projection.IsNull() {
				t.Fatal("monotone projection was not registered")
			}
			for i, input := range inputs {
				application, err := m.MkApp(node.projection, []ast.ASTHandle{makeNumeral(input)})
				astTestMust(t, err)
				value, err := graph.eval(application)
				astTestMust(t, err)
				expected := makeNumeral(want[i])
				if value != expected {
					t.Fatalf("projection(%s) = %v, want %s (%v)", input, value, want[i], expected)
				}
			}
		})
	}
}
