// Source: src/math/polynomial/polynomial.cpp:236-544.
package polynomial

import (
	"errors"
	"testing"
)

func TestMonomialNativeTableGrowthRenameAndOrder(t *testing.T) {
	m := monomialFixture(t)
	const variables = 257
	const count = 4097
	for i := 0; i < variables; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	owned := make([]Monomial, 0, count)
	t.Cleanup(func() {
		m.config.Budget.FailAfter(-1)
		for _, a := range owned {
			serviceMust(t, m.DecRef(a))
		}
	})
	for i := 0; i < count; i++ {
		a, e := m.MkVariable(uint32(i%variables), uint32(i/variables+1))
		serviceMust(t, e)
		serviceMust(t, m.IncRef(a))
		owned = append(owned, a)
	}
	m.config.Budget.FailAfter(0)
	before, used := m.memory.Stats(), m.config.Budget.Used()
	var table monomialTable
	ids := make([]uint32, count)
	for i, a := range owned {
		id, inserted, e := table.insert(m, monomialKey{monomialObjectKey, a.slot})
		serviceMust(t, e)
		if !inserted {
			t.Fatal("new native row", i)
		}
		ids[i] = id
	}
	i := 0
	for id, entry := range table.entries.All() {
		if id != ids[i] || entry.key != (monomialKey{monomialObjectKey, owned[i].slot}) {
			t.Fatal("insertion order", i, id, entry)
		}
		i++
	}
	if i != count {
		t.Fatal("native rows", i)
	}
	for i, a := range owned {
		k := monomialKey{monomialObjectKey, a.slot}
		id, inserted, e := table.insert(m, k)
		serviceMust(t, e)
		if inserted || id != ids[i] {
			t.Fatal("duplicate identity", i, id)
		}
		if i%3 == 0 {
			serviceMust(t, table.erase(m, k))
		}
	}
	i = 0
	for _, entry := range table.entries.All() {
		for i%3 == 0 {
			i++
		}
		if entry.key != (monomialKey{monomialObjectKey, owned[i].slot}) {
			t.Fatal("survivor order", i, entry)
		}
		i++
	}
	table.cleanup()
	if table.entries.Len() != 0 || table.buckets != nil {
		t.Fatal("native cleanup retained rows")
	}
	permutation := make([]uint32, variables)
	for i := range permutation {
		permutation[i] = uint32(variables - 1 - i)
	}
	for attempt := 0; attempt < 2; attempt++ {
		serviceMust(t, m.Rename(permutation))
		for i, a := range owned {
			v := uint32(i % variables)
			if attempt == 0 {
				v = uint32(variables-1) - v
			}
			monomialEqualPowers(t, m, a, []MonomialPower{{v, uint32(i/variables + 1)}})
			got, e := m.MkVariable(v, uint32(i/variables+1))
			serviceMust(t, e)
			if got != a {
				t.Fatal("renamed canonical identity", i, got, a)
			}
		}
		if m.renameTable.entries.Len() != 0 || m.renameFound != nil {
			t.Fatal("lexical native scratch retained")
		}
	}
	if m.memory.Stats() != before || m.config.Budget.Used() != used {
		t.Fatal("native table/rename changed Runtime accounting")
	}
	monomialPhysicalAudit(t, m)
}

func TestMonomialNativeTableCollisionsAndTemporaryPublication(t *testing.T) {
	m := monomialFixture(t)
	var table monomialTable
	m.config.Budget.FailAfter(0)
	t.Cleanup(func() { m.config.Budget.FailAfter(-1) })
	before, used := m.memory.Stats(), m.config.Budget.Used()
	for i := uint32(0); i < 3; i++ {
		serviceMust(t, m.tempInit(&m.temps[i], []MonomialPower{{i + 5, i + 7}}))
		m.temps[i].hash = 11 // Force a project-hash collision between distinct powers.
	}
	keys := []monomialKey{{monomialTemporaryKey, 0}, {monomialTemporaryKey, 1}, {monomialTemporaryKey, 2}}
	ids := make([]uint32, 3)
	for i, k := range keys {
		id, inserted, e := table.insert(m, k)
		serviceMust(t, e)
		if !inserted {
			t.Fatal("collision merged unequal powers", i)
		}
		ids[i] = id
	}
	powers, e := m.tempPowers(&m.temps[1])
	serviceMust(t, e)
	serviceMust(t, m.tempInit(&m.temps[3], powers))
	m.temps[3].hash = 11
	id, inserted, e := table.insert(m, monomialKey{monomialTemporaryKey, 3})
	serviceMust(t, e)
	if inserted || id != ids[1] {
		t.Fatal("equal colliding powers not interned", id, inserted)
	}
	serviceMust(t, table.setKey(ids[1], monomialKey{monomialTemporaryKey, 3}))
	got, e := table.key(ids[1])
	serviceMust(t, e)
	if got != (monomialKey{monomialTemporaryKey, 3}) {
		t.Fatal("stable publication", got)
	}
	// Owning cleanup uses the stored bucket even after a source key mutation.
	m.temps[3].hash = 29
	table.removeToken(got)
	if _, found := table.entries.Get2(ids[1]); found {
		t.Fatal("mutated temporary retained")
	}
	for _, i := range []int{0, 2} {
		id, found, e := table.find(m, keys[i])
		serviceMust(t, e)
		if !found || id != ids[i] {
			t.Fatal("collision peer lost", i, id, found)
		}
	}
	serviceMust(t, table.erase(m, keys[0]))
	serviceMust(t, table.erase(m, keys[2]))
	if table.entries.Len() != 0 || len(table.buckets) != 0 {
		t.Fatal("deleted collision bucket retained")
	}
	if m.memory.Stats() != before || m.config.Budget.Used() != used {
		t.Fatal("native collision storage changed Runtime accounting")
	}
	monomialPhysicalAudit(t, m)
}

func TestMonomialNativeTableSourceCleanupPendingCreation(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	m := monomialFixture(t)
	powers := make([]MonomialPower, 129)
	for i := range powers {
		powers[i] = MonomialPower{uint32(i), 1}
	}
	m.config.Budget.FailAfter(0)
	_, e := m.MkMonomial(powers)
	m.config.Budget.FailAfter(-1)
	if e == nil || !numericPhysicalReservationRefusal(e) || m.creation.stage == 0 {
		t.Fatal("real factory allocation must stop after native publication", e, m.creation)
	}
	e = m.SourceFinalize()
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) || m.sourceStage != 1 {
			t.Fatal("live table assertion", e, m.sourceStage)
		}
	} else {
		serviceMust(t, e)
		if !m.creation.tableRemoved {
			t.Fatal("source destruction lost removal receipt")
		}
	}
	serviceMust(t, m.CleanupCreation())
	serviceMust(t, m.SourceFinalize())
	monomialPhysicalAudit(t, m)
}

func TestMonomialNativeRenamePowerSorting(t *testing.T) {
	m := monomialFixture(t)
	powers := make([]MonomialPower, 129)
	permutation := make([]uint32, len(powers))
	want := make([]MonomialPower, len(powers))
	for i := range powers {
		_, e := m.MkVar()
		serviceMust(t, e)
		powers[i] = MonomialPower{uint32(i), uint32(i%11 + 1)}
		permutation[i] = uint32(i * 37 % len(powers))
		want[permutation[i]] = MonomialPower{permutation[i], powers[i].Degree}
	}
	a := monomialOwn(t, m, powers)
	m.config.Budget.FailAfter(0)
	t.Cleanup(func() { m.config.Budget.FailAfter(-1) })
	before, used := m.memory.Stats(), m.config.Budget.Used()
	serviceMust(t, m.Rename(permutation))
	monomialEqualPowers(t, m, a, want)
	duplicate, e := m.MkMonomial(want)
	serviceMust(t, e)
	if duplicate != a {
		t.Fatal("sorted powers changed canonical identity")
	}
	if m.memory.Stats() != before || m.config.Budget.Used() != used {
		t.Fatal("native sort changed Runtime accounting")
	}
	monomialPhysicalAudit(t, m)
}
