package arena

// Words is the uint32 payload specialization used for literal words and limbs.
// Capacity and PageSize are expressed in words, never bytes.
type Words = Vectors[uint32]

func NewWords(config VectorConfig) (*Words, error) { return NewVectors[uint32](config) }
