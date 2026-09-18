// Package fsutil contains internal helpers for base_dir handling, path resolution,
// and walk skip policy (default build/VCS/dependency directories plus gitignore).
//
// base_dir is only an anchor for resolving relative paths; resolved paths are
// NOT restricted to stay inside base_dir. AllowedPaths-style sandboxing was
// removed deliberately — do not reintroduce an allowedPaths parameter here.
package fsutil
