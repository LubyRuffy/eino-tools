//go:build unix

package proctree

import (
	"os/exec"
	"syscall"
)

// Group is the process group a command joins before it starts.
// Closing it does not signal anyone: a finished shell may leave
// background jobs, and only an explicit interrupt or kill stops the group.
type Group struct{}

// New puts cmd in its own process group. The caller still starts it.
func New(cmd *exec.Cmd) (*Group, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &Group{}, nil
}

// AfterStart is the Windows job-assignment step. Unix already set the group.
func (g *Group) AfterStart(cmd *exec.Cmd) error { return nil }

// Interrupt asks the group to stop.
func (g *Group) Interrupt(cmd *exec.Cmd) { signal(cmd, syscall.SIGTERM) }

// Kill stops the group immediately.
func (g *Group) Kill(cmd *exec.Cmd) { signal(cmd, syscall.SIGKILL) }

// Close releases nothing on Unix.
func (g *Group) Close() {}

func signal(cmd *exec.Cmd, sig syscall.Signal) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return
	}
	_ = syscall.Kill(-pgid, sig)
}
