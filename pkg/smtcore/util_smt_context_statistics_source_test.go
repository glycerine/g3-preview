package smtcore

// Source: src/smt/smt_statistics.h:26-54; src/smt/smt_statistics.cpp:24-26;
// src/smt/smt_context.h:70.

import (
	"reflect"
	"testing"
)

func TestSMTContextStatisticsConstructorAndLiteralResetAllFields(t *testing.T) {
	s := NewSMTContextStatistics()
	if s != (SMTContextStatistics{}) {
		t.Fatal("source constructor reset", s)
	}
	// Distinct nonzero source words ensure reset consumes every field, including
	// the tail NumChecks; a partial prefix reset cannot satisfy this independent trace.
	v := reflect.ValueOf(&s).Elem()
	if v.NumField() != 22 {
		t.Fatal("literal declaration inventory", v.NumField())
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() != reflect.Uint32 {
			t.Fatal("pointer-free source unsigned member", i)
		}
		v.Field(i).SetUint(uint64(0x80000001 + uint32(i)))
	}
	s.Reset()
	if s != (SMTContextStatistics{}) {
		t.Fatal("memset source all members", s)
	}
	s.Reset()
	if s != (SMTContextStatistics{}) {
		t.Fatal("repeat zero reset", s)
	}
}
func TestSMTContextStatisticsInlineCopiesAndUnsignedProducerWords(t *testing.T) {
	s := NewSMTContextStatistics()
	s.NumChecks = ^uint32(0)
	s.NumChecks++
	if s.NumChecks != 0 {
		t.Fatal("actual unsigned source storage wraps")
	}
	s.NumConflicts = 9
	s.NumDynAck = 13
	s.MaxGeneration = 17
	copy := s
	copy.Reset()
	if s.NumConflicts != 9 || s.NumDynAck != 13 || s.MaxGeneration != 17 || copy != (SMTContextStatistics{}) {
		t.Fatal("source scalar value copy/reset", s, copy)
	}
	if reflect.TypeOf(s).Size() != 22*4 {
		t.Fatal("actual declared Go scalar geometry")
	}
}
