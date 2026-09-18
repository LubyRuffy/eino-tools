package exec

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExecuteReportsStdoutChunksBeforeTheCommandFinishes(t *testing.T) {
	// A host that only sees output after Wait cannot show a live exec row.
	first := make(chan string, 1)
	var mu sync.Mutex
	var chunks []string
	ctx := WithOutputListener(context.Background(), func(stream string, chunk []byte) {
		if stream != "stdout" {
			return
		}
		mu.Lock()
		chunks = append(chunks, string(chunk))
		mu.Unlock()
		select {
		case first <- string(chunk):
		default:
		}
	})

	tl, err := New(Config{})
	require.NoError(t, err)

	done := make(chan struct{})
	var payload map[string]interface{}
	var runErr error
	go func() {
		defer close(done)
		payload, runErr = tl.Execute(ctx, Params{
			Command:   "printf a; sleep 0.3; printf b",
			TimeoutMS: 5000,
		})
	}()

	select {
	case got := <-first:
		require.Equal(t, "a", got)
	case <-done:
		t.Fatal("command finished before any stdout chunk — output is still buffered until Wait")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first stdout chunk")
	}

	<-done
	require.NoError(t, runErr)
	require.Contains(t, payload["stdout"], "ab")
	mu.Lock()
	defer mu.Unlock()
	joined := ""
	for _, c := range chunks {
		joined += c
	}
	require.Contains(t, joined, "ab")
}

func TestExecuteReportsStderrChunksBeforeTheCommandFinishes(t *testing.T) {
	first := make(chan string, 1)
	ctx := WithOutputListener(context.Background(), func(stream string, chunk []byte) {
		if stream != "stderr" {
			return
		}
		select {
		case first <- string(chunk):
		default:
		}
	})

	tl, err := New(Config{})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = tl.Execute(ctx, Params{
			Command:   "printf a >&2; sleep 0.3; printf b >&2",
			TimeoutMS: 5000,
		})
	}()

	select {
	case got := <-first:
		require.Equal(t, "a", got)
	case <-done:
		t.Fatal("command finished before any stderr chunk")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first stderr chunk")
	}
	<-done
}

func TestExecuteWithoutListenerStillCapturesOutput(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)
	payload, runErr := tl.Execute(context.Background(), Params{
		Command:   "printf ab",
		TimeoutMS: 2000,
	})
	require.NoError(t, runErr)
	require.Contains(t, payload["stdout"], "ab")
}
