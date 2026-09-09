package containers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeTransport drives pingDaemon/negotiateAPIVersion without a real unix socket.
type fakeTransport struct {
	seen    []string
	handler func(*http.Request) (*http.Response, error)
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.seen = append(f.seen, req.URL.Path)
	return f.handler(req)
}

func newFakeClient(h func(*http.Request) (*http.Response, error)) (*http.Client, *fakeTransport) {
	ft := &fakeTransport{handler: h}
	return &http.Client{Transport: ft}, ft
}

func fakeResponse(status int, headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader("OK")),
	}
}

func TestPingDaemonStatusHandling(t *testing.T) {
	tests := []struct {
		doc       string
		status    int
		transport error
		wantErr   bool
	}{
		{doc: "200 is healthy", status: http.StatusOK},
		{doc: "400 is not healthy", status: http.StatusBadRequest, wantErr: true},
		{doc: "404 is not healthy", status: http.StatusNotFound, wantErr: true},
		{doc: "500 is not healthy", status: http.StatusInternalServerError, wantErr: true},
		{doc: "transport error is not healthy", transport: errors.New("no such socket"), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			client, _ := newFakeClient(func(*http.Request) (*http.Response, error) {
				if tc.transport != nil {
					return nil, tc.transport
				}
				return fakeResponse(tc.status, nil), nil
			})

			_, err := pingDaemon(context.Background(), client, socketBaseURL+"/_ping")
			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestNegotiateAPIVersion(t *testing.T) {
	tests := []struct {
		doc         string
		advertised  string
		pingStatus  int
		versionedOK bool
		accept      func(string) bool
		fallback    string
		want        string
	}{
		{
			doc: "docker adopts the advertised version", advertised: "1.52", versionedOK: true,
			fallback: dockerFallbackVersion, want: "v1.52",
		},
		{
			// Regression guard: a Docker 24 daemon (max API 1.43) must keep working.
			// Negotiating must not silently bump the prefix to the 1.44 fallback.
			doc: "older engine keeps its own version", advertised: "1.43", versionedOK: true,
			fallback: dockerFallbackVersion, want: "v1.43",
		},
		{
			doc: "missing header falls back", advertised: "", versionedOK: true,
			fallback: dockerFallbackVersion, want: dockerFallbackVersion,
		},
		{
			doc: "non-numeric header falls back", advertised: "abc", versionedOK: true,
			fallback: dockerFallbackVersion, want: dockerFallbackVersion,
		},
		{
			doc: "three-part version falls back", advertised: "1.52.0", versionedOK: true,
			fallback: dockerFallbackVersion, want: dockerFallbackVersion,
		},
		{
			doc: "podman native version falls back", advertised: "5.2.1", versionedOK: true,
			accept: isDockerCompatVersion, fallback: podmanFallbackVersion, want: podmanFallbackVersion,
		},
		{
			doc: "podman compat version is adopted", advertised: "1.44", versionedOK: true,
			accept: isDockerCompatVersion, fallback: podmanFallbackVersion, want: "v1.44",
		},
		{
			doc: "unroutable prefix falls back", advertised: "1.52", versionedOK: false,
			fallback: dockerFallbackVersion, want: dockerFallbackVersion,
		},
		{
			doc: "unreachable daemon falls back", advertised: "1.52", pingStatus: http.StatusInternalServerError,
			versionedOK: true, fallback: dockerFallbackVersion, want: dockerFallbackVersion,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			client, ft := newFakeClient(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/_ping" {
					if tc.pingStatus != 0 && tc.pingStatus != http.StatusOK {
						return fakeResponse(tc.pingStatus, nil), nil
					}
					if tc.advertised == "" {
						return fakeResponse(http.StatusOK, nil), nil
					}
					return fakeResponse(http.StatusOK, map[string]string{apiVersionHeader: tc.advertised}), nil
				}
				if !tc.versionedOK {
					return fakeResponse(http.StatusNotFound, nil), nil
				}
				return fakeResponse(http.StatusOK, nil), nil
			})

			got := negotiateAPIVersion(context.Background(), client, "test", tc.fallback, tc.accept)
			if got != tc.want {
				t.Errorf("negotiateAPIVersion() = %q, want %q", got, tc.want)
			}
			// The very first probe must be version-less — a versioned probe could be
			// rejected as "too old" before we learn what the daemon accepts.
			if len(ft.seen) == 0 || ft.seen[0] != "/_ping" {
				t.Errorf("first probe path = %v, want /_ping first", ft.seen)
			}
		})
	}
}

func TestURLUsesNegotiatedVersion(t *testing.T) {
	d := &Docker{client: http.DefaultClient, apiVer: "v1.52"}
	if got, want := d.url("/containers/json"), "http://localhost/v1.52/containers/json"; got != want {
		t.Errorf("Docker.url() = %q, want %q", got, want)
	}

	p := &Podman{client: http.DefaultClient, apiVer: podmanFallbackVersion}
	if got, want := p.url("/containers/json"), "http://localhost/v4.0.0/containers/json"; got != want {
		t.Errorf("Podman.url() = %q, want %q", got, want)
	}
}
