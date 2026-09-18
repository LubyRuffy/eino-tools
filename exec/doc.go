// Package exec provides the current `exec` tool implementation.
//
// Commands run with a non-interactive bash (`--noprofile --norc -c`).
// The user's `$SHELL` and rc files are not sourced. Timeout kills the
// still-running command process group and does not reap background jobs
// after the shell has already exited.
package exec
