// Source: src/util/mpz.cpp:1385-1410; src/util/mpn.cpp:371-399; src/util/buffer.h.
package util

// SourceCharacterBuffer is the actual source character buffer embedded in a
// caller's lexical frame. Primitive storage and its cleanup remain in util.
type SourceCharacterBuffer = sourceCharBuffer

func (s *sourceCharBuffer) ReadPhysical(r *Runtime, i uint64) (byte, error) {
	return s.readPhysical(r, i)
}
func (s *sourceCharBuffer) Resize(r *Runtime, n uint32) error { return s.resize(r, n) }
func (s *sourceCharBuffer) Finalize(r *Runtime) error         { return s.finalize(r) }
func (s *sourceCharBuffer) Pending() Block                    { return s.PendingField }
func (s *sourceCharBuffer) ReleasePublished(r *Runtime) error {
	if s.Raw == (Block{}) {
		return nil
	}
	return r.Deallocate(&s.Raw)
}

type IntegerMagnitudeView struct {
	owner *IntegerManager
	limbs limbView
}

func (m *IntegerManager) BorrowMagnitude(a *Int) (IntegerMagnitudeView, error) {
	if e := m.check(a); e != nil {
		return IntegerMagnitudeView{}, e
	}
	if a.Cell == 0 {
		return IntegerMagnitudeView{}, ErrNumericOwner
	}
	c := m.Cell(a.Cell)
	size, e := c.Backing.ReadWord(m.Allocator, 0)
	if e != nil {
		return IntegerMagnitudeView{}, e
	}
	return IntegerMagnitudeView{m, limbView{allocator: m.Allocator, backing: c.Backing, size: size}}, nil
}
func (v IntegerMagnitudeView) Size() uint32 { return v.limbs.size }
func (v IntegerMagnitudeView) WriteCString(s *SourceCharacterBuffer, f *NaturalFormattingScope) error {
	if v.owner == nil || !v.owner.ValidOwner() || s == nil || f == nil {
		return ErrNumericOwner
	}
	return v.owner.Natural.sourceToCStringScoped(v.limbs, sourceDecimalDestination{character: s, assembly: v.owner.Natural}, uint64(s.pos), f)
}
func (s *naturalFormattingScope) Pending() bool       { return s.PendingField }
func (s *naturalFormattingScope) Completed() uint8    { return s.CompletedField }
func (s *naturalFormattingScope) CleanupCause() error { return s.cleanupCause }
func (s *naturalFormattingScope) CleanupAfterSource(m *NaturalManager) error {
	if m == nil || !m.ValidOwner() {
		return ErrNumericOwner
	}
	s.Active = false
	return m.cleanupFormattingScope(s)
}
