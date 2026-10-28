package rel

// Source: src/muz/rel/dl_instruction.cpp unary-singleton, total-relation,
// and signature-assertion instructions.

import "testing"

func TestFiniteRelationConstructionInstructions(t *testing.T) {
	context := NewExecutionContext(nil)
	t.Cleanup(func() { _ = context.Close() })
	empty, err := NewSignature(nil)
	if err != nil {
		t.Fatal(err)
	}
	totalSignature, err := NewSignature([]uint64{2, 3})
	if err != nil {
		t.Fatal(err)
	}
	singleton, err := NewUnarySingletonInstruction(4, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	block := NewInstructionBlock()
	t.Cleanup(func() { _ = block.Close() })
	for _, instruction := range []Instruction{
		singleton,
		NewTotalInstruction(empty, 1),
		NewTotalInstruction(totalSignature, 2),
		NewAssertSignatureInstruction(totalSignature, 2),
	} {
		if err = block.Append(instruction); err != nil {
			t.Fatal(err)
		}
	}
	complete, err := block.Perform(context)
	if err != nil || !complete {
		t.Fatal("construct instructions", complete, err)
	}
	instructionWantFacts(t, instructionFacts(t, context, 0), Fact{2})
	instructionWantFacts(t, instructionFacts(t, context, 1), Fact{})
	instructionWantFacts(t, instructionFacts(t, context, 2),
		Fact{0, 0}, Fact{0, 1}, Fact{0, 2}, Fact{1, 0}, Fact{1, 1}, Fact{1, 2})
	stats, err := context.Statistics()
	if err != nil || stats.UnarySingleton != 1 || stats.Total != 2 {
		t.Fatal("construct statistics", stats, err)
	}
	wrong, err := NewSignature([]uint64{2, 2})
	if err != nil {
		t.Fatal(err)
	}
	assertion := NewAssertSignatureInstruction(wrong, 2)
	t.Cleanup(func() { _ = assertion.Close() })
	if complete, err = assertion.Perform(context); err == nil || complete {
		t.Fatal("wrong signature assertion accepted", complete, err)
	}
}

func TestUnarySingletonInstructionRejectsOutOfDomainValue(t *testing.T) {
	if instruction, err := NewUnarySingletonInstruction(2, 2, 0); err == nil || instruction != nil {
		t.Fatal("out-of-domain singleton accepted", instruction, err)
	}
}
