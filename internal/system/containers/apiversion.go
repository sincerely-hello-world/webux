package containers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
)

// socketBaseURL is the pseudo-host used for every request. Connections are actually
// established against the runtime's unix socket by the transport's DialContext, so
// the host part is never resolved.
const socketBaseURL = "http://localhost"

// apiVersionHeader is advertised by the /_ping endpoint of both Docker and Podman.
const apiVersionHeader = "API-Version"

// Fallback API versions, used when a runtime cannot be asked for its version or
// answers with something unusable.
//
//   - Docker: Engine 29.0–29.2 reject anything below 1.44, so 1.44 is the lowest
//     prefix that is safe across the range of supported daemons.
//   - Podman: v4.0.0 is the prefix this package has always used for Podman's
//     Docker-compatible endpoints.
const (
	dockerFallbackVersion = "v1.44"
	podmanFallbackVersion = "v4.0.0"
)

var (
	// apiVersionRe matches a "major.minor" version as advertised by a daemon.
	apiVersionRe = regexp.MustCompile(`^\d+\.\d+$`)
	// dockerCompatVersionRe matches Docker-compatible ("1.<minor>") versions.
	dockerCompatVersionRe = regexp.MustCompile(`^1\.\d+$`)
)

// pingDaemon probes url and reports whether the runtime is alive and healthy.
//
// Unlike a bare http.Client.Do this fails on a non-200 response, so a daemon that
// rejects our requests — for example with "client version is too old" — is not
// mistaken for a working one. The body is drained so the connection can be reused.
func pingDaemon(ctx context.Context, client *http.Client, url string) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ping %s: HTTP %d", url, resp.StatusCode)
	}
	return resp.Header, nil
}

// negotiateAPIVersion asks the runtime which Docker-compatible API version it serves
// and returns the URL prefix to use for subsequent requests (for example "v1.52").
//
// The unversioned /_ping endpoint is used for the probe: every Engine version serves
// it, and a request without a version prefix is handled by the daemon at its own
// newest version, so the probe itself can never be rejected as "too old".
//
// It returns fallback unless the runtime advertises a well-formed version that accept
// endorses *and* that the daemon actually routes, so an unusable answer can never be
// adopted. accept may be nil to allow any advertised version; Podman passes a checker
// that only allows Docker-compatible ("1.x") values, because Podman also advertises
// its own native version, which is not a valid prefix for the endpoints used here.
func negotiateAPIVersion(ctx context.Context, client *http.Client, runtime, fallback string, accept func(string) bool) string {
	hdr, err := pingDaemon(ctx, client, socketBaseURL+"/_ping")
	if err != nil {
		slog.Debug("container runtime: API version negotiation unavailable",
			"runtime", runtime, "err", err, "fallback", fallback)
		return fallback
	}

	advertised := hdr.Get(apiVersionHeader)
	if !apiVersionRe.MatchString(advertised) || (accept != nil && !accept(advertised)) {
		slog.Info("container runtime: using fallback API version",
			"runtime", runtime, "advertised", advertised, "fallback", fallback)
		return fallback
	}

	candidate := "v" + advertised
	// Confirm the daemon routes the advertised prefix before committing to it.
	if _, err := pingDaemon(ctx, client, socketBaseURL+"/"+candidate+"/_ping"); err != nil {
		slog.Warn("container runtime: advertised API version is not routable, using fallback",
			"runtime", runtime, "advertised", advertised, "fallback", fallback, "err", err)
		return fallback
	}

	slog.Info("container runtime: negotiated API version", "runtime", runtime, "version", candidate)
	return candidate
}

// isDockerCompatVersion reports whether v is a Docker-compatible API version ("1.x").
func isDockerCompatVersion(v string) bool { return dockerCompatVersionRe.MatchString(v) }
