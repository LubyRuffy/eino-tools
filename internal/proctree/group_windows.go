//go:build windows

package proctree

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Group is a Windows job that owns the command and the children it starts.
// The job is not kill-on-close: a shell that has already exited may leave
// background jobs, matching the Unix group. Interrupt and Kill are what
// stop a command that is still running.
type Group struct {
	job windows.Handle
}

// New creates the job and starts cmd suspended so no child can escape
// before AfterStart assigns the job and resumes the primary thread.
func New(cmd *exec.Cmd) (*Group, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create process job: %w", err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_SUSPENDED | windows.CREATE_NEW_PROCESS_GROUP,
	}
	return &Group{job: job}, nil
}

// AfterStart assigns the started process to the job and lets it run.
func (g *Group) AfterStart(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("command has no process")
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("open process: %w", err)
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(g.job, process); err != nil {
		return fmt.Errorf("assign process job: %w", err)
	}
	if err := resume(uint32(cmd.Process.Pid)); err != nil {
		return err
	}
	return nil
}

// Interrupt sends a console break to the new process group.
func (g *Group) Interrupt(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid))
}

// Kill terminates every process still in the job, then the root process.
func (g *Group) Kill(cmd *exec.Cmd) {
	if g != nil && g.job != 0 {
		_ = windows.TerminateJobObject(g.job, 1)
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// Close releases the job handle. It does not terminate survivors.
func (g *Group) Close() {
	if g == nil || g.job == 0 {
		return
	}
	_ = windows.CloseHandle(g.job)
	g.job = 0
}

func resume(pid uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fmt.Errorf("snapshot process threads: %w", err)
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	found := false
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != pid {
			continue
		}
		thread, openErr := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if openErr != nil {
			return fmt.Errorf("open process thread: %w", openErr)
		}
		_, resumeErr := windows.ResumeThread(thread)
		closeErr := windows.CloseHandle(thread)
		if resumeErr != nil {
			return fmt.Errorf("resume process: %w", resumeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close process thread: %w", closeErr)
		}
		found = true
	}
	if !found {
		if err != nil && !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			return fmt.Errorf("find process thread: %w", err)
		}
		return fmt.Errorf("process %d has no suspended thread", pid)
	}
	return nil
}
