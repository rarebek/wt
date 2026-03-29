package codec

// Raw is a no-op codec that passes bytes through unchanged.
// Useful when messages are already serialized or when using raw binary protocols.
type Raw struct{}

func (Raw) Marshal(v any) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}
	return nil, nil
}

func (Raw) Unmarshal(data []byte, v any) error {
	if p, ok := v.(*[]byte); ok {
		*p = data
	}
	return nil
}

func (Raw) Name() string { return "raw" }
