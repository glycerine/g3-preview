// Source: src/util/ref_util.h:26-88.
package util

import (
	"reflect"
	"testing"
)

func TestDecRefMapKeyValuesOrderAndReset(t *testing.T) {
	entries := []RefMapEntry{{Key: "k1", Value: "v1"}, {Key: nil, Value: "v2"}}
	events := make([]string, 0, 5)
	DecRefMapKeyValues(
		func(value any) { events = append(events, "key:"+refCleanupName(value)) },
		func(value any) { events = append(events, "value:"+refCleanupName(value)) },
		entries,
		func() { events = append(events, "reset") },
	)
	want := []string{"key:k1", "value:v1", "key:<nil>", "value:v2", "reset"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestDecRefMapKeyValuesSameManager(t *testing.T) {
	entries := []RefMapEntry{{Key: "k1", Value: "v1"}, {Key: "k2", Value: "v2"}}
	events := make([]string, 0, 5)
	DecRefMapKeyValuesSame(
		func(value any) { events = append(events, value.(string)) },
		entries,
		func() { events = append(events, "reset") },
	)
	want := []string{"k1", "v1", "k2", "v2", "reset"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestDecRefMapKeysValuesAndCollection(t *testing.T) {
	entries := []RefMapEntry{{Key: "k1", Value: "v1"}, {Key: "k2", Value: "v2"}}
	var events []string
	reset := func() { events = append(events, "reset") }
	dec := func(value any) { events = append(events, value.(string)) }

	DecRefMapKeys(dec, entries, reset)
	DecRefMapValues(dec, entries, reset)
	DecRefCollectionValues(dec, []any{"c1", "c2"}, reset)
	want := []string{"k1", "k2", "reset", "v1", "v2", "reset", "c1", "c2", "reset"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestDecRefCleanupEmptyAndInterruptedBehavior(t *testing.T) {
	resets := 0
	DecRefMapKeys(func(any) {}, nil, func() { resets++ })
	DecRefMapValues(func(any) {}, nil, func() { resets++ })
	DecRefCollectionValues(func(any) {}, nil, func() { resets++ })
	if resets != 3 {
		t.Fatalf("empty cleanup resets = %d, want 3", resets)
	}

	entries := []RefMapEntry{{Key: "k1", Value: "v1"}, {Key: "k2", Value: "v2"}}
	events := make([]string, 0, 2)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("decrement callback did not panic")
			}
		}()
		DecRefMapKeyValues(
			func(value any) { events = append(events, value.(string)) },
			func(any) { panic("stop") },
			entries,
			func() { events = append(events, "reset") },
		)
	}()
	if want := []string{"k1"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("interrupted events = %v, want %v", events, want)
	}
}

func TestDecRefCleanupRejectsMissingCallbacksBeforeEffects(t *testing.T) {
	entries := []RefMapEntry{{Key: "k", Value: "v"}}
	for name, operation := range map[string]func(){
		"key decrement":   func() { DecRefMapKeyValues(nil, func(any) {}, entries, func() {}) },
		"value decrement": func() { DecRefMapKeyValues(func(any) {}, nil, entries, func() {}) },
		"same decrement":  func() { DecRefMapKeyValuesSame(nil, entries, func() {}) },
		"keys reset":      func() { DecRefMapKeys(func(any) {}, entries, nil) },
		"values reset":    func() { DecRefMapValues(func(any) {}, entries, nil) },
		"collection reset": func() {
			DecRefCollectionValues(func(any) {}, []any{"value"}, nil)
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("missing callback did not panic")
				}
			}()
			operation()
		})
	}
}

func refCleanupName(value any) string {
	if value == nil {
		return "<nil>"
	}
	return value.(string)
}
