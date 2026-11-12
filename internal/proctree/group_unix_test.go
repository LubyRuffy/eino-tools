//go:build unix

package proctree

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestCommandBecomesItsOwnGroupAndStops(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "30")
	group, err := New(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer group.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	if err := group.AfterStart(cmd); err != nil {
		t.Fatal(err)
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil || pgid != cmd.Process.Pid {
		t.Fatalf("pgid=%d pid=%d err=%v", pgid, cmd.Process.Pid, err)
	}
	group.Interrupt(cmd)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("interrupt left the process running")
	}
}

func TestKillStopsTheProcessGroup(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "30")
	group, err := New(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer group.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := group.AfterStart(cmd); err != nil {
		t.Fatal(err)
	}
	group.Kill(cmd)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("kill left the process running")
	}
}

func TestSignalWithoutAProcessOrALivePidDoesNothing(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "30")
	group, err := New(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer group.Close()
	group.Interrupt(cmd)
	group.Kill(cmd)
	group.Interrupt(nil)
	group.Kill(nil)

	proc, err := os.FindProcess(1 << 30)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Process = proc
	group.Interrupt(cmd)
	group.Kill(cmd)
}
