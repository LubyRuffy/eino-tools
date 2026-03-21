package shared

type LimitedBuffer struct {
	limit     int
	truncated bool
	buf       []byte
}

func NewLimitedBuffer(limit int) *LimitedBuffer {
	if limit <= 0 {
		limit = 256 * 1024
	}
	return &LimitedBuffer{
		limit: limit,
		buf:   make([]byte, 0, min(limit, 8192)),
	}
}

func (b *LimitedBuffer) Write(p []byte) (int, error) {
	if b.truncated {
		return len(p), nil
	}
	remaining := b.limit - len(b.buf)
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf = append(b.buf, p[:remaining]...)
		b.truncated = true
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *LimitedBuffer) String() string {
	return string(b.buf)
}

func (b *LimitedBuffer) Truncated() bool {
	return b.truncated
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
