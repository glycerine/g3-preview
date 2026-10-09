// Source: src/util/gparams.cpp; src/util/dictionary.h; src/util/map.h.
package util

import "github.com/glycerine/insdict"

// Module descriptors are rendered by traversal, so their dictionary preserves
// registration order. The other two dictionaries are lookup-only.
type gparamsDescriptorDictionary struct {
	values *insdict.Dict[Symbol, uint32]
}

type gparamsRefDictionary struct {
	values map[Symbol]uint32
}

type gparamsDescriptionDictionary struct {
	values map[Symbol]GlobalParamCString
}

func (d *gparamsDescriptorDictionary) init() {
	d.values = insdict.NewDictFunc[Symbol, uint32](func(k Symbol) uint64 {
		h := k.owner ^ uint64(k.Value)<<32 | uint64(k.Tag)
		return h ^ uint64(k.Generation)*0x9e3779b97f4a7c15
	})
}
func (d *gparamsDescriptorDictionary) get(k Symbol) (uint32, bool) {
	if d.values == nil {
		return 0, false
	}
	return d.values.Get2(k)
}
func (d *gparamsDescriptorDictionary) set(k Symbol, descriptor uint32) {
	if d.values == nil {
		d.init()
	}
	d.values.Put(k, descriptor)
}
func (d *gparamsDescriptorDictionary) reset() {
	if d.values != nil {
		d.values.Clear()
	}
}
func (d *gparamsDescriptorDictionary) close() { d.values = nil }
func (d *gparamsDescriptorDictionary) size() uint32 {
	if d.values == nil {
		return 0
	}
	return uint32(d.values.Len())
}

func (d *gparamsRefDictionary) init() { d.values = make(map[Symbol]uint32) }
func (d *gparamsRefDictionary) get(k Symbol) (uint32, bool) {
	v, ok := d.values[k]
	return v, ok
}
func (d *gparamsRefDictionary) set(k Symbol, cell uint32) {
	if d.values == nil {
		d.init()
	}
	d.values[k] = cell
}
func (d *gparamsRefDictionary) reset() { clear(d.values) }
func (d *gparamsRefDictionary) close() { d.values = nil }
func (d *gparamsRefDictionary) size() uint32 {
	return uint32(len(d.values))
}

func (d *gparamsDescriptionDictionary) init() {
	d.values = make(map[Symbol]GlobalParamCString)
}
func (d *gparamsDescriptionDictionary) get(k Symbol) (GlobalParamCString, bool) {
	v, ok := d.values[k]
	return v, ok
}
func (d *gparamsDescriptionDictionary) set(k Symbol, text GlobalParamCString) {
	if d.values == nil {
		d.init()
	}
	d.values[k] = text
}
func (d *gparamsDescriptionDictionary) reset() { clear(d.values) }
func (d *gparamsDescriptionDictionary) close() { d.values = nil }
func (d *gparamsDescriptionDictionary) size() uint32 {
	return uint32(len(d.values))
}
