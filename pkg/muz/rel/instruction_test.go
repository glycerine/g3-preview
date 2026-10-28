package rel

// Source: src/muz/rel/dl_instruction.cpp execution_context, instruction_block,
// instr_dealloc, and instr_clone_move.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/statistics"
)

func instructionTable(t *testing.T, values ...Element) *HashTable {
	t.Helper()
	signature, err := NewSignature([]uint64{8})
	if err != nil {
		t.Fatal(err)
	}
	table := NewHashTable(signature)
	for _, value := range values {
		if _, err = table.Add(Fact{value}); err != nil {
			t.Fatal(err)
		}
	}
	return table
}

func TestInstructionBlockCloneMoveDeallocateOwnership(t *testing.T) {
	context := NewExecutionContext(nil)
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if err := context.SetRegister(0, instructionTable(t, 1, 2)); err != nil {
		t.Fatal(err)
	}
	if err := context.SetRegisterAnnotation(0, "seed"); err != nil {
		t.Fatal(err)
	}
	block := NewInstructionBlock()
	t.Cleanup(func() {
		if err := block.Close(); err != nil {
			t.Fatal(err)
		}
	})
	for _, instruction := range []Instruction{NewCloneInstruction(0, 1), NewMoveInstruction(1, 2), NewDeallocateInstruction(0)} {
		if err := block.Append(instruction); err != nil {
			t.Fatal(err)
		}
	}
	if err := block.MakeAnnotations(context); err != nil {
		t.Fatal(err)
	}
	complete, err := block.Perform(context)
	if err != nil || !complete {
		t.Fatal("instruction block", complete, err)
	}
	if value, _ := context.Register(0); value != nil {
		t.Fatal("deallocate retained source")
	}
	if value, _ := context.Register(1); value != nil {
		t.Fatal("move retained source")
	}
	value, err := context.Register(2)
	if err != nil || value == nil {
		t.Fatal("move lost target", err)
	}
	if count, err := value.Len(); err != nil || count != 2 {
		t.Fatal("clone/move rows", count, err)
	}
	if annotation, found, err := context.RegisterAnnotation(2); err != nil || !found || annotation != "seed" {
		t.Fatal("annotation", annotation, found, err)
	}
	if err = block.ProcessAllCosts(); err != nil {
		t.Fatal(err)
	}
	stats := statistics.New()
	if err = block.CollectStatistics(stats); err != nil || stats.Size() == 0 {
		t.Fatal("instruction statistics", stats.Size(), err)
	}
	var display bytes.Buffer
	if err = block.Display(&display); err != nil || display.String() != "clone 0 into 1\nmove 1 into 2\ndealloc 0\n" {
		t.Fatal("instruction display", display.String(), err)
	}
}

func TestInstructionBlockChecksTerminationBeforePerform(t *testing.T) {
	checks := 0
	context := NewExecutionContext(func() bool {
		checks++
		return checks == 2
	})
	t.Cleanup(func() { _ = context.Close() })
	if err := context.SetRegister(0, instructionTable(t, 3)); err != nil {
		t.Fatal(err)
	}
	block := NewInstructionBlock()
	t.Cleanup(func() { _ = block.Close() })
	if err := block.Append(NewCloneInstruction(0, 1)); err != nil {
		t.Fatal(err)
	}
	if err := block.Append(NewMoveInstruction(1, 2)); err != nil {
		t.Fatal(err)
	}
	complete, err := block.Perform(context)
	if err != nil || complete {
		t.Fatal("termination", complete, err)
	}
	if value, _ := context.Register(1); value == nil {
		t.Fatal("first instruction did not run")
	}
	if value, _ := context.Register(2); value != nil {
		t.Fatal("instruction ran after termination")
	}
}
