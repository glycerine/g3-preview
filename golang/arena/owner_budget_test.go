package arena

import (
	"errors"
	"reflect"
	"testing"
)

// These are cold test dispatchers; no interface/callback is retained per record.
type ownerBudgetCase struct {
	name      string
	expected  uint64
	construct func(*Budget) (func(), func() uint64, func() uint64, error)
}

func ownerBudgetCases() []ownerBudgetCase {
	slab := uint64(reflect.TypeOf(Slab32{}).Size())
	descriptor := uint64(reflect.TypeOf(descriptorSlab{}).Size())
	identity := uint64(reflect.TypeOf(regionIdentity{}).Size())
	return []ownerBudgetCase{
		{"Slab32", slab, func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewSlab32(Config{PageSize: 2, Budget: b})
			if e != nil {
				return nil, nil, nil, e
			}
			return s.Close, func() uint64 { s.Reset(); return s.Stats().OwnerBytes }, func() uint64 { return s.Stats().OwnerBytes }, nil
		}},
		{"Slab64", uint64(reflect.TypeOf(Slab64{}).Size()), func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewSlab64(Config{PageSize: 2, Budget: b})
			if e != nil {
				return nil, nil, nil, e
			}
			return s.Close, func() uint64 { s.Reset(); return s.Stats().OwnerBytes }, func() uint64 { return s.Stats().OwnerBytes }, nil
		}},
		{"descriptor", descriptor, func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := newDescriptorSlab(Config{PageSize: 2, Budget: b})
			if e != nil {
				return nil, nil, nil, e
			}
			return s.Close, func() uint64 { s.Reset(); return s.Stats().OwnerBytes }, func() uint64 { return s.Stats().OwnerBytes }, nil
		}},
		{"record test slab", uint64(reflect.TypeOf(recordSlab{}).Size()), func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := newRecordSlab(Config{PageSize: 2, Budget: b})
			if e != nil {
				return nil, nil, nil, e
			}
			return s.Close, func() uint64 { s.Reset(); return s.Stats().OwnerBytes }, func() uint64 { return s.Stats().OwnerBytes }, nil
		}},
		{"Words", uint64(reflect.TypeOf(Words{}).Size()) + descriptor, func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewWords(VectorConfig{PageSize: 2, Descriptors: Config{PageSize: 2, Budget: b}})
			if e != nil {
				return nil, nil, nil, e
			}
			stats := func() uint64 { v := s.Stats(); return v.OwnerBytes + v.Descriptors.OwnerBytes }
			return s.Close, func() uint64 { s.Reset(); return stats() }, stats, nil
		}},
		{"Vectors64", uint64(reflect.TypeOf(Vectors64{}).Size()) + descriptor, func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewVectors64(VectorConfig{PageSize: 2, Descriptors: Config{PageSize: 2, Budget: b}})
			if e != nil {
				return nil, nil, nil, e
			}
			stats := func() uint64 { v := s.Stats(); return v.OwnerBytes + v.Descriptors.OwnerBytes }
			return s.Close, func() uint64 { s.Reset(); return stats() }, stats, nil
		}},
		{"Vectors32Signed", uint64(reflect.TypeOf(Vectors32Signed{}).Size()) + descriptor, func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewVectors32Signed(VectorConfig{PageSize: 2, Descriptors: Config{PageSize: 2, Budget: b}})
			if e != nil {
				return nil, nil, nil, e
			}
			stats := func() uint64 { v := s.Stats(); return v.OwnerBytes + v.Descriptors.OwnerBytes }
			return s.Close, func() uint64 { s.Reset(); return stats() }, stats, nil
		}},
		{"Region32", uint64(reflect.TypeOf(Region32{}).Size()) + identity + slab, func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewRegion32(Config{PageSize: 2, Budget: b})
			if e != nil {
				return nil, nil, nil, e
			}
			stats := func() uint64 { v := s.Stats(); return v.OwnerBytes + v.Storage.OwnerBytes }
			return func() {
					if e := s.Close(nil); e != nil {
						panic(e)
					}
				}, func() uint64 {
					if e := s.Reset(nil); e != nil {
						panic(e)
					}
					return stats()
				}, stats, nil
		}},
		{"Region64", uint64(reflect.TypeOf(Region64{}).Size()) + identity + uint64(reflect.TypeOf(Slab64{}).Size()), func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewRegion64(Config{PageSize: 2, Budget: b})
			if e != nil {
				return nil, nil, nil, e
			}
			stats := func() uint64 { v := s.Stats(); return v.OwnerBytes + v.Storage.OwnerBytes }
			return func() {
					if e := s.Close(nil); e != nil {
						panic(e)
					}
				}, func() uint64 {
					if e := s.Reset(nil); e != nil {
						panic(e)
					}
					return stats()
				}, stats, nil
		}},
		{"Pool32", uint64(reflect.TypeOf(Pool32{}).Size()), func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewPool32(Pool32Config{PageSize: 2, Workers: 1, Budget: b}, Pool32Policy{})
			if e != nil {
				return nil, nil, nil, e
			}
			stats := func() uint64 { return s.Stats().OwnerBytes }
			return func() {
					if e := s.Close(); e != nil {
						panic(e)
					}
				}, func() uint64 {
					if e := s.Reset(); e != nil {
						panic(e)
					}
					return stats()
				}, stats, nil
		}},
		{"SmallBytes", uint64(reflect.TypeOf(SmallBytes{}).Size()) + descriptor, func(b *Budget) (func(), func() uint64, func() uint64, error) {
			s, e := NewSmallBytes(SmallBytesConfig{Descriptors: Config{PageSize: 2, Budget: b}})
			if e != nil {
				return nil, nil, nil, e
			}
			return s.Close, func() uint64 { s.Reset(); return s.Stats().OwnerBytes }, func() uint64 { return s.Stats().OwnerBytes }, nil
		}},
	}
}
func TestOwnerBudgetConstructorAndClose(t *testing.T) {
	for _, tc := range ownerBudgetCases() {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBudget(1 << 20)
			b.Reserve(7)
			close, reset, stats, e := tc.construct(b)
			if e != nil {
				t.Fatal(e)
			}
			backing := uint64(0)
			if tc.name == "Pool32" {
				backing = uint64(reflect.TypeOf(pool32Worker{}).Size()) + uint64(reflect.TypeOf(pool32Page{}).Size()) + 2*(4+uint64(reflect.TypeOf(pool32Meta{}).Size()))
			}
			if stats() != tc.expected || b.Used() != 7+tc.expected+backing {
				t.Fatal("owner header uncharged", stats(), tc.expected, b.Used())
			}
			retained := b.Used()
			if reset() != tc.expected || b.Used() != retained {
				t.Fatal("reset changed cold owner ownership")
			}
			close()
			close()
			if stats() != 0 || b.Used() != 7 {
				t.Fatal("close owner charge leaked/doubled", stats(), b.Used())
			}
			bounded := NewBudget(tc.expected + backing - 1)
			close, _, _, e = tc.construct(bounded)
			if !errors.Is(e, ErrBudget) || close != nil || bounded.Used() != 0 {
				t.Fatal("partial nested constructor budget unwind", e, bounded.Used())
			}
			close, _, _, e = tc.construct(NewBudget(0))
			if !errors.Is(e, ErrBudget) || close != nil {
				t.Fatal("zero-budget constructor accepted owner", e)
			}
		})
	}
}
func TestOwnerBudgetConstructorFailureUnwind(t *testing.T) {
	for _, tc := range ownerBudgetCases() {
		t.Run(tc.name, func(t *testing.T) {
			failures, successes := 0, 0
			for after := int64(0); after < 8; after++ {
				b := NewBudget(1 << 20)
				b.Reserve(7)
				b.FailAfter(after)
				close, _, _, e := tc.construct(b)
				if e != nil {
					failures++
					if !errors.Is(e, ErrInjectedFailure) || close != nil {
						t.Fatal("unexpected constructor failure", e)
					}
				} else {
					successes++
					b.FailAfter(-1)
					close()
				}
				if b.Used() != 7 {
					t.Fatalf("failure stage%d retained%d", after, b.Used()-7)
				}
			}
			if failures == 0 || successes == 0 {
				t.Fatal("failure sweep did not exercise both outcomes")
			}
		})
	}
}

func TestOwnerBudgetInvalidConfigurationUnwind(t *testing.T) {
	cases := []struct {
		name      string
		construct func(*Budget) error
	}{
		{"Slab32", func(b *Budget) error { _, e := NewSlab32(Config{PageSize: 3, Budget: b}); return e }},
		{"Slab64", func(b *Budget) error { _, e := NewSlab64(Config{PageSize: 3, Budget: b}); return e }},
		{"Words descriptor", func(b *Budget) error {
			_, e := NewWords(VectorConfig{Descriptors: Config{PageSize: 3, Budget: b}})
			return e
		}},
		{"Vectors64 descriptor", func(b *Budget) error {
			_, e := NewVectors64(VectorConfig{Descriptors: Config{PageSize: 3, Budget: b}})
			return e
		}},
		{"Vectors32Signed descriptor", func(b *Budget) error {
			_, e := NewVectors32Signed(VectorConfig{Descriptors: Config{PageSize: 3, Budget: b}})
			return e
		}},
		{"Region32 child", func(b *Budget) error { _, e := NewRegion32(Config{PageSize: 3, Budget: b}); return e }},
		{"Region64 child", func(b *Budget) error { _, e := NewRegion64(Config{PageSize: 3, Budget: b}); return e }},
		{"Pool32", func(b *Budget) error {
			_, e := NewPool32(Pool32Config{PageSize: 3, Budget: b}, Pool32Policy{})
			return e
		}},
		{"SmallBytes descriptor", func(b *Budget) error {
			_, e := NewSmallBytes(SmallBytesConfig{Descriptors: Config{PageSize: 3, Budget: b}})
			return e
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBudget(1 << 20)
			b.Reserve(7)
			b.FailAfter(0)
			if e := tc.construct(b); !errors.Is(e, ErrConfig) || b.Used() != 7 {
				t.Fatal("configuration error changed external charge", e, b.Used())
			}
		})
	}
}
