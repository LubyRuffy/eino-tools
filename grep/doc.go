// Package grep provides the current `grep` tool implementation.
//
// Repository-wide searches skip default build/VCS/dependency directories,
// honor gitignore rules, and skip unreadable, binary, or overlong files
// instead of aborting the whole walk.
package grep
