// Source: src/smt/diff_logic.h sparse graph construction, incremental edge
// activation, feasible assignments, negative-cycle explanations and scopes.
package smtcore

import (
	"errors"
	"math/big"
)

var errSMTDiffGraphDomain = errors.New("smt: invalid difference-logic graph operation")

type smtDiffSearchMark uint8

const (
	smtDiffUnmarked smtDiffSearchMark = iota
	smtDiffFound
	smtDiffProcessed
)

type smtRationalDiffEdge struct {
	source, target int32
	weight         big.Rat
	timestamp      uint32
	explanation    SMTLiteral
	enabled        bool
}

type smtRationalDiffAssignmentTrail struct {
	variable int32
	oldValue big.Rat
}

type smtRationalDiffScope struct {
	edges, enabled int
	timestamp      uint32
}

// SMTRationalDiffGraph is the concrete rational dl_graph instantiation used
// by integer difference logic. Rational values are copied into graph-owned
// big.Rat values; explanations are borrowed SMT literals.
type SMTRationalDiffGraph struct {
	assignment        []big.Rat
	assignmentTrail   []smtRationalDiffAssignmentTrail
	edges             []smtRationalDiffEdge
	outEdges, inEdges [][]int32
	scopes            []smtRationalDiffScope
	gamma             []big.Rat
	mark              []smtDiffSearchMark
	parent            []int32
	visited           []int32
	heap              smtRationalDiffHeap
	timestamp         uint32
	lastEnabledEdge   int32
	enabledEdges      []int32
	activity          []uint32
	propagationCost   uint32
	numRelax          uint32
}

func NewSMTRationalDiffGraph() *SMTRationalDiffGraph {
	g := &SMTRationalDiffGraph{lastEnabledEdge: -1}
	g.heap.graph = g
	return g
}

func (g *SMTRationalDiffGraph) validVariable(v int32) bool {
	return g != nil && v >= 0 && int(v) < len(g.assignment)
}

func (g *SMTRationalDiffGraph) validEdge(id int32) bool {
	return g != nil && id >= 0 && int(id) < len(g.edges)
}

// InitVariable creates all missing source IDs through v and resets v. As in
// the source, an already initialized variable may be reset only while it has
// no incident edges.
func (g *SMTRationalDiffGraph) InitVariable(v int32) error {
	if g == nil || v < 0 {
		return errSMTDiffGraphDomain
	}
	for int(v) >= len(g.assignment) {
		g.assignment = append(g.assignment, big.Rat{})
		g.outEdges = append(g.outEdges, nil)
		g.inEdges = append(g.inEdges, nil)
		g.gamma = append(g.gamma, big.Rat{})
		g.mark = append(g.mark, smtDiffUnmarked)
		g.parent = append(g.parent, -1)
	}
	if len(g.outEdges[v]) != 0 || len(g.inEdges[v]) != 0 || g.mark[v] != smtDiffUnmarked {
		return errSMTDiffGraphDomain
	}
	g.assignment[v].SetInt64(0)
	return nil
}

func (g *SMTRationalDiffGraph) AddEdge(source, target int32, weight *big.Rat, explanation SMTLiteral) (int32, error) {
	if !g.validVariable(source) || !g.validVariable(target) || weight == nil {
		return -1, errSMTDiffGraphDomain
	}
	id := int32(len(g.edges))
	edge := smtRationalDiffEdge{source: source, target: target, timestamp: g.timestamp, explanation: explanation}
	edge.weight.Set(weight)
	g.edges = append(g.edges, edge)
	g.activity = append(g.activity, 0)
	g.outEdges[source] = append(g.outEdges[source], id)
	g.inEdges[target] = append(g.inEdges[target], id)
	return id, nil
}

func (g *SMTRationalDiffGraph) edgeFeasible(edge *smtRationalDiffEdge) bool {
	if !edge.enabled {
		return true
	}
	var difference big.Rat
	difference.Sub(&g.assignment[edge.target], &g.assignment[edge.source])
	return difference.Cmp(&edge.weight) <= 0
}

func (g *SMTRationalDiffGraph) IsFeasible() bool {
	if g == nil {
		return false
	}
	for index := range g.edges {
		if !g.edgeFeasible(&g.edges[index]) {
			return false
		}
	}
	return true
}

func (g *SMTRationalDiffGraph) setGamma(edge *smtRationalDiffEdge, result *big.Rat) {
	result.Sub(&g.assignment[edge.source], &g.assignment[edge.target])
	result.Add(result, &edge.weight)
}

func (g *SMTRationalDiffGraph) accumulateAssignment(variable int32, increment *big.Rat) {
	trail := smtRationalDiffAssignmentTrail{variable: variable}
	trail.oldValue.Set(&g.assignment[variable])
	g.assignmentTrail = append(g.assignmentTrail, trail)
	g.assignment[variable].Add(&g.assignment[variable], increment)
}

func (g *SMTRationalDiffGraph) undoAssignments() {
	for index := len(g.assignmentTrail); index > 0; index-- {
		trail := &g.assignmentTrail[index-1]
		g.assignment[trail.variable].Set(&trail.oldValue)
	}
	g.assignmentTrail = g.assignmentTrail[:0]
}

func (g *SMTRationalDiffGraph) resetMarks() {
	for _, variable := range g.visited {
		g.mark[variable] = smtDiffUnmarked
	}
	g.visited = g.visited[:0]
}

func (g *SMTRationalDiffGraph) makeFeasible(id int32) bool {
	edge := &g.edges[id]
	root, target := edge.source, edge.target
	g.gamma[root].SetInt64(0)
	g.setGamma(edge, &g.gamma[target])
	g.mark[target] = smtDiffProcessed
	g.parent[target] = id
	g.visited = append(g.visited, target)
	g.accumulateAssignment(target, &g.gamma[target])

	source := target
	for {
		g.propagationCost++
		if g.mark[root] != smtDiffUnmarked {
			g.heap.reset()
			g.resetMarks()
			g.undoAssignments()
			return false
		}
		for _, edgeID := range g.outEdges[source] {
			candidate := &g.edges[edgeID]
			if !candidate.enabled {
				continue
			}
			var gamma big.Rat
			g.setGamma(candidate, &gamma)
			if gamma.Sign() >= 0 {
				continue
			}
			next := candidate.target
			switch g.mark[next] {
			case smtDiffUnmarked:
				g.gamma[next].Set(&gamma)
				g.mark[next] = smtDiffFound
				g.parent[next] = edgeID
				g.visited = append(g.visited, next)
				g.heap.insert(next)
			case smtDiffFound:
				if gamma.Cmp(&g.gamma[next]) < 0 {
					g.gamma[next].Set(&gamma)
					g.parent[next] = edgeID
					g.heap.decreased(next)
				}
			}
		}
		if g.heap.empty() {
			g.resetMarks()
			g.assignmentTrail = g.assignmentTrail[:0]
			return true
		}
		source = g.heap.eraseMin()
		g.mark[source] = smtDiffProcessed
		g.accumulateAssignment(source, &g.gamma[source])
	}
}

// EnableEdge returns false precisely when this activation closes a negative
// cycle. The edge remains enabled so NegativeCycleExplanations can inspect it.
func (g *SMTRationalDiffGraph) EnableEdge(id int32) (bool, error) {
	if !g.validEdge(id) {
		return false, errSMTDiffGraphDomain
	}
	edge := &g.edges[id]
	if edge.enabled {
		return true, nil
	}
	edge.enabled = true
	edge.timestamp = g.timestamp
	g.lastEnabledEdge = id
	g.timestamp++
	feasible := true
	if !g.edgeFeasible(edge) {
		feasible = g.makeFeasible(id)
	}
	g.enabledEdges = append(g.enabledEdges, id)
	return feasible, nil
}

func (g *SMTRationalDiffGraph) findRelaxedEdge(id int32, gamma *big.Rat) int32 {
	edge := &g.edges[id]
	weight := new(big.Rat).Set(&edge.weight)
	selected := id
	for _, candidateID := range g.outEdges[edge.source] {
		candidate := &g.edges[candidateID]
		if candidate.target != edge.target || !candidate.enabled || candidate.weight.Cmp(weight) <= 0 {
			continue
		}
		var delta, adjusted big.Rat
		delta.Sub(&candidate.weight, weight)
		adjusted.Add(gamma, &delta)
		if adjusted.Sign() < 0 {
			selected = candidateID
			gamma.Set(&adjusted)
			weight.Set(&candidate.weight)
			g.numRelax++
		}
	}
	return selected
}

func (g *SMTRationalDiffGraph) NegativeCycleExplanations(tryRelax bool) ([]SMTLiteral, error) {
	if !g.validEdge(g.lastEnabledEdge) || g.edgeFeasible(&g.edges[g.lastEnabledEdge]) {
		return nil, errSMTDiffGraphDomain
	}
	last := g.lastEnabledEdge
	edge := &g.edges[last]
	var gamma big.Rat
	gamma.Set(&g.gamma[edge.source])
	result := make([]SMTLiteral, 0, 4)
	id := last
	for steps := 0; ; steps++ {
		if steps > len(g.edges) || !g.validEdge(id) {
			return nil, errSMTDiffGraphDomain
		}
		selected := id
		if tryRelax {
			selected = g.findRelaxedEdge(id, &gamma)
		}
		g.activity[id]++
		result = append(result, g.edges[selected].explanation)
		id = g.parent[g.edges[id].source]
		if id == last {
			break
		}
	}
	return result, nil
}

func (g *SMTRationalDiffGraph) Push() error {
	if g == nil {
		return errSMTDiffGraphDomain
	}
	g.scopes = append(g.scopes, smtRationalDiffScope{len(g.edges), len(g.enabledEdges), g.timestamp})
	return nil
}

func (g *SMTRationalDiffGraph) Pop(count uint32) error {
	if g == nil || count == 0 || int(count) > len(g.scopes) {
		return errSMTDiffGraphDomain
	}
	level := len(g.scopes) - int(count)
	scope := g.scopes[level]
	for index := len(g.enabledEdges); index > scope.enabled; index-- {
		g.edges[g.enabledEdges[index-1]].enabled = false
	}
	g.enabledEdges = g.enabledEdges[:scope.enabled]
	g.timestamp = scope.timestamp
	for len(g.edges) > scope.edges {
		id := int32(len(g.edges) - 1)
		edge := &g.edges[id]
		g.outEdges[edge.source] = g.outEdges[edge.source][:len(g.outEdges[edge.source])-1]
		g.inEdges[edge.target] = g.inEdges[edge.target][:len(g.inEdges[edge.target])-1]
		g.edges = g.edges[:id]
		g.activity = g.activity[:id]
	}
	g.scopes = g.scopes[:level]
	return nil
}

func (g *SMTRationalDiffGraph) Assignment(variable int32) (*big.Rat, error) {
	if !g.validVariable(variable) {
		return nil, errSMTDiffGraphDomain
	}
	return new(big.Rat).Set(&g.assignment[variable]), nil
}

func (g *SMTRationalDiffGraph) EdgeSource(id int32) (int32, error) {
	if !g.validEdge(id) {
		return -1, errSMTDiffGraphDomain
	}
	return g.edges[id].source, nil
}

func (g *SMTRationalDiffGraph) EdgeTarget(id int32) (int32, error) {
	if !g.validEdge(id) {
		return -1, errSMTDiffGraphDomain
	}
	return g.edges[id].target, nil
}

func (g *SMTRationalDiffGraph) EdgeWeight(id int32) (*big.Rat, error) {
	if !g.validEdge(id) {
		return nil, errSMTDiffGraphDomain
	}
	return new(big.Rat).Set(&g.edges[id].weight), nil
}

func (g *SMTRationalDiffGraph) EdgeExplanation(id int32) (SMTLiteral, error) {
	if !g.validEdge(id) {
		return SMTNullLiteral, errSMTDiffGraphDomain
	}
	return g.edges[id].explanation, nil
}

func (g *SMTRationalDiffGraph) EdgeEnabled(id int32) (bool, error) {
	if !g.validEdge(id) {
		return false, errSMTDiffGraphDomain
	}
	return g.edges[id].enabled, nil
}

// SetToZero translates every assignment by the selected value. Difference
// constraints are translation invariant, so feasibility is preserved.
func (g *SMTRationalDiffGraph) SetToZero(variable int32) error {
	if !g.validVariable(variable) || !g.IsFeasible() {
		return errSMTDiffGraphDomain
	}
	if g.assignment[variable].Sign() == 0 {
		return nil
	}
	var offset big.Rat
	offset.Set(&g.assignment[variable])
	for index := range g.assignment {
		g.assignment[index].Sub(&g.assignment[index], &offset)
	}
	return nil
}

// SetPairToZero implements the source helper for disconnected variables. If
// translation does not zero both values, zero-weight edges join the components.
func (g *SMTRationalDiffGraph) SetPairToZero(first, second int32) error {
	if !g.validVariable(first) || !g.validVariable(second) || !g.IsFeasible() {
		return errSMTDiffGraphDomain
	}
	selected := second
	if g.assignment[first].Sign() != 0 {
		selected = first
	}
	if err := g.SetToZero(selected); err != nil {
		return err
	}
	if g.assignment[first].Sign() == 0 && g.assignment[second].Sign() == 0 {
		return nil
	}
	zero := new(big.Rat)
	forward, err := g.AddEdge(first, second, zero, SMTNullLiteral)
	if err != nil {
		return err
	}
	if feasible, err := g.EnableEdge(forward); err != nil || !feasible {
		if err != nil {
			return err
		}
		return errSMTDiffGraphDomain
	}
	backward, err := g.AddEdge(second, first, zero, SMTNullLiteral)
	if err != nil {
		return err
	}
	if feasible, err := g.EnableEdge(backward); err != nil || !feasible {
		if err != nil {
			return err
		}
		return errSMTDiffGraphDomain
	}
	return nil
}

// FindDirectlySubsumed returns disabled parallel edges whose bounds are no
// stronger than the selected edge. It preserves source adjacency order.
func (g *SMTRationalDiffGraph) FindDirectlySubsumed(id int32) ([]int32, error) {
	if !g.validEdge(id) {
		return nil, errSMTDiffGraphDomain
	}
	edge := &g.edges[id]
	result := make([]int32, 0)
	candidates := g.outEdges[edge.source]
	useOutgoing := len(candidates) < len(g.inEdges[edge.target])
	if !useOutgoing {
		candidates = g.inEdges[edge.target]
	}
	for _, candidateID := range candidates {
		g.propagationCost++
		candidate := &g.edges[candidateID]
		matches := candidate.target == edge.target
		if !useOutgoing {
			matches = candidate.source == edge.source
		}
		if candidateID != id && !candidate.enabled && matches && candidate.weight.Cmp(&edge.weight) >= 0 {
			result = append(result, candidateID)
		}
	}
	return result, nil
}

// FindTwoHopSubsumed extends direct subsumption through one enabled incoming
// edge at the source and one enabled outgoing edge at the destination.
func (g *SMTRationalDiffGraph) FindTwoHopSubsumed(id int32) ([]int32, error) {
	if !g.validEdge(id) {
		return nil, errSMTDiffGraphDomain
	}
	result, err := g.FindDirectlySubsumed(id)
	if err != nil {
		return nil, err
	}
	edge := &g.edges[id]
	var combined big.Rat
	for _, prefixID := range g.inEdges[edge.source] {
		prefix := &g.edges[prefixID]
		if !prefix.enabled || prefix.source == edge.target {
			continue
		}
		combined.Add(&prefix.weight, &edge.weight)
		for _, candidateID := range g.outEdges[prefix.source] {
			g.propagationCost++
			candidate := &g.edges[candidateID]
			if !candidate.enabled && candidate.target == edge.target && candidate.weight.Cmp(&combined) >= 0 {
				result = append(result, candidateID)
			}
		}
	}
	for _, suffixID := range g.outEdges[edge.target] {
		suffix := &g.edges[suffixID]
		if !suffix.enabled || suffix.target == edge.source {
			continue
		}
		combined.Add(&suffix.weight, &edge.weight)
		for _, candidateID := range g.inEdges[suffix.target] {
			g.propagationCost++
			candidate := &g.edges[candidateID]
			if !candidate.enabled && candidate.source == edge.source && candidate.weight.Cmp(&combined) >= 0 {
				result = append(result, candidateID)
			}
		}
	}
	return result, nil
}

func (g *SMTRationalDiffGraph) NumNodes() int {
	if g == nil {
		return 0
	}
	return len(g.assignment)
}
func (g *SMTRationalDiffGraph) NumEdges() int {
	if g == nil {
		return 0
	}
	return len(g.edges)
}

// Concrete indexed minimum heap ordered by the graph gamma values.
type smtRationalDiffHeap struct {
	graph     *SMTRationalDiffGraph
	variables []int32
	positions []int
}

func (h *smtRationalDiffHeap) empty() bool { return len(h.variables) == 0 }
func (h *smtRationalDiffHeap) ensure(v int32) {
	for int(v) >= len(h.positions) {
		h.positions = append(h.positions, -1)
	}
}
func (h *smtRationalDiffHeap) less(i, j int) bool {
	return h.graph.gamma[h.variables[i]].Cmp(&h.graph.gamma[h.variables[j]]) < 0
}
func (h *smtRationalDiffHeap) swap(i, j int) {
	h.variables[i], h.variables[j] = h.variables[j], h.variables[i]
	h.positions[h.variables[i]], h.positions[h.variables[j]] = i, j
}
func (h *smtRationalDiffHeap) up(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if !h.less(index, parent) {
			break
		}
		h.swap(index, parent)
		index = parent
	}
}
func (h *smtRationalDiffHeap) down(index int) {
	for {
		child := index*2 + 1
		if child >= len(h.variables) {
			return
		}
		if child+1 < len(h.variables) && h.less(child+1, child) {
			child++
		}
		if !h.less(child, index) {
			return
		}
		h.swap(index, child)
		index = child
	}
}
func (h *smtRationalDiffHeap) insert(v int32) {
	h.ensure(v)
	if h.positions[v] >= 0 {
		return
	}
	h.positions[v] = len(h.variables)
	h.variables = append(h.variables, v)
	h.up(len(h.variables) - 1)
}
func (h *smtRationalDiffHeap) decreased(v int32) {
	h.ensure(v)
	if h.positions[v] >= 0 {
		h.up(h.positions[v])
	}
}
func (h *smtRationalDiffHeap) eraseMin() int32 {
	result := h.variables[0]
	last := len(h.variables) - 1
	h.swap(0, last)
	h.variables = h.variables[:last]
	h.positions[result] = -1
	if len(h.variables) != 0 {
		h.down(0)
	}
	return result
}
func (h *smtRationalDiffHeap) reset() {
	for _, v := range h.variables {
		h.positions[v] = -1
	}
	h.variables = h.variables[:0]
}
