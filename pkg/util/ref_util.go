// Source: src/util/ref_util.h:26-88.
package util

// RefMapEntry is one key/value pair supplied to the reference-release helpers.
// Callers choose the entry order used by their concrete container.
type RefMapEntry struct {
	Key   any
	Value any
}

func requireRefCleanup(dec func(any), reset func()) {
	if dec == nil {
		panic("g3: reference cleanup requires a decrement callback")
	}
	if reset == nil {
		panic("g3: reference cleanup requires a reset callback")
	}
}

// DecRefMapKeyValues decrements each key and then its value before resetting
// the source map. Entries must be supplied in the concrete map's iteration
// order.
func DecRefMapKeyValues(decKey, decValue func(any), entries []RefMapEntry, reset func()) {
	requireRefCleanup(decKey, reset)
	if decValue == nil {
		panic("g3: reference cleanup requires a value decrement callback")
	}
	for _, entry := range entries {
		decKey(entry.Key)
		decValue(entry.Value)
	}
	reset()
}

// DecRefMapKeyValuesSame uses one decrement callback for keys and values.
func DecRefMapKeyValuesSame(dec func(any), entries []RefMapEntry, reset func()) {
	DecRefMapKeyValues(dec, dec, entries, reset)
}

// DecRefMapKeys decrements every key before resetting the source map.
func DecRefMapKeys(dec func(any), entries []RefMapEntry, reset func()) {
	requireRefCleanup(dec, reset)
	for _, entry := range entries {
		dec(entry.Key)
	}
	reset()
}

// DecRefMapValues decrements every value before resetting the source map.
func DecRefMapValues(dec func(any), entries []RefMapEntry, reset func()) {
	requireRefCleanup(dec, reset)
	for _, entry := range entries {
		dec(entry.Value)
	}
	reset()
}

// DecRefCollectionValues decrements every value before resetting the source
// collection.
func DecRefCollectionValues(dec func(any), values []any, reset func()) {
	requireRefCleanup(dec, reset)
	for _, value := range values {
		dec(value)
	}
	reset()
}
