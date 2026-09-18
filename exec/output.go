package exec

import "context"

// OutputListener receives one stdout or stderr chunk while a command is
// still running. stream is "stdout" or "stderr". The slice is not reused
// after the call returns. The listener must not block for long: it sits on
// the process's write path.
type OutputListener func(stream string, chunk []byte)

type outputListenerKey struct{}

// WithOutputListener attaches a live output callback to ctx. Execute
// delivers chunks as they are written, not after Wait.
func WithOutputListener(ctx context.Context, fn OutputListener) context.Context {
	if ctx == nil || fn == nil {
		return ctx
	}
	return context.WithValue(ctx, outputListenerKey{}, fn)
}

func outputListenerFrom(ctx context.Context) OutputListener {
	if ctx == nil {
		return nil
	}
	fn, _ := ctx.Value(outputListenerKey{}).(OutputListener)
	return fn
}

// liveWriter copies each Write to the listener. The pipe reuses its buffer,
// so the callback gets a snapshot, not an alias.
type liveWriter struct {
	stream string
	fn     OutputListener
}

func (w liveWriter) Write(p []byte) (int, error) {
	if w.fn != nil && len(p) > 0 {
		w.fn(w.stream, append([]byte(nil), p...))
	}
	return len(p), nil
}
