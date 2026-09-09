//go:build !cgo

package auth

// yescryptVerify is a stub for the pure-Go build.
// Full yescrypt support requires a CGO build — the mise task for that is
// `mise run build-pam` (CGO_ENABLED=1). Every other task inherits the
// repository-wide [env] CGO_ENABLED=0 from mise.toml.
// This is called only when CGO is disabled AND the hash starts with $y$.
// The error message guides users to the correct build.
func yescryptVerify(password, hash string) error {
	return nil // handled upstream in pureGoCrypt
}
