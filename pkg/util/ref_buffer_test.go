// Source: src/util/ref_buffer.h:29-171.
package util

import (
	"reflect"
	"testing"
)

type refBufferValue struct{ id int }

type refBufferEvents struct{ values []string }

func (events *refBufferEvents) inc(value any) {
	events.values = append(events.values, "i"+refBufferID(value))
}
func (events *refBufferEvents) dec(value any) {
	events.values = append(events.values, "d"+refBufferID(value))
}
func refBufferID(value any) string {
	if nilScopedPointer(value) {
		return "nil"
	}
	return string(rune('0' + value.(*refBufferValue).id))
}

func TestRefBufferCallbackOrderPushSetPopReset(t *testing.T) {
	events := &refBufferEvents{}
	buffer := NewRefBuffer("manager", events.inc, events.dec)
	a, b := &refBufferValue{id: 1}, &refBufferValue{id: 2}
	buffer.PushBack(a)
	buffer.PushBack(b)
	buffer.Set(0, a)
	buffer.Set(1, a)
	buffer.PopBack()
	buffer.Reset()
	want := []string{"i1", "i2", "i1", "d1", "i1", "d2", "d1", "d1"}
	if !reflect.DeepEqual(events.values, want) || !buffer.Empty() || buffer.Manager() != "manager" {
		t.Fatal("callback order", events.values)
	}
	buffer.Close()
}

func TestRefBufferResizeSetXShrinkAndNilCallbacks(t *testing.T) {
	events := &refBufferEvents{}
	buffer := NewRefBuffer(nil, events.inc, events.dec)
	a, b := &refBufferValue{id: 1}, &refBufferValue{id: 2}
	buffer.Resize(3)
	if buffer.Size() != 3 || buffer.Get(0) != nil {
		t.Fatal("nil growth")
	}
	buffer.SetX(4, a)
	buffer.Set(0, b)
	buffer.Shrink(1)
	want := []string{"i1", "dnil", "i2", "dnil", "dnil", "dnil", "dnil", "d1"}
	if !reflect.DeepEqual(events.values, want) || buffer.Size() != 1 || buffer.Get(0) != b {
		t.Fatal("resize callbacks", events.values, buffer.Size())
	}
	buffer.Close()
	if events.values[len(events.values)-1] != "d2" {
		t.Fatal("close callback", events.values)
	}
}

func TestRefBufferSelfAppendCloneAssignAndRawReferences(t *testing.T) {
	events := &refBufferEvents{}
	buffer := NewRefBuffer(events, events.inc, events.dec)
	a, b := &refBufferValue{id: 1}, &refBufferValue{id: 2}
	buffer.Append([]any{a, b})
	buffer.AppendBuffer(buffer)
	if buffer.Size() != 4 || buffer.Get(0) != a || buffer.Get(2) != a {
		t.Fatal("captured self append")
	}
	clone := buffer.Clone()
	if clone.Manager() != events || clone.Size() != 4 || &clone.values[0] == &buffer.values[0] {
		t.Fatal("clone")
	}
	*clone.BackRef() = b
	if clone.Back() != b {
		t.Fatal("raw back reference")
	}
	clone.Assign(clone)
	other := NewRefBuffer(events, events.inc, events.dec)
	other.Assign(buffer)
	if other.Size() != 4 {
		t.Fatal("assignment")
	}
	other.Close()
	clone.Close()
	buffer.Close()
}

func TestRefBufferUnmanagedAndPreconditionPanics(t *testing.T) {
	buffer := NewRefBuffer(nil, nil, nil)
	value := &refBufferValue{id: 1}
	buffer.PushBack(value)
	copyOwner := *buffer
	for name, operation := range map[string]func(){
		"copy":       copyOwner.Reset,
		"nonpointer": func() { buffer.PushBack(1) },
		"get":        func() { buffer.Get(2) },
		"shrink":     func() { buffer.Shrink(2) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("missing panic")
				}
			}()
			operation()
		})
	}
	buffer.Close()
}
