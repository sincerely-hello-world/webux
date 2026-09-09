package containers

import (
	"testing"
)

func TestValidateRunConfig(t *testing.T) {
	tests := []struct {
		doc     string
		cfg     RunConfig
		wantErr bool
	}{
		{
			doc: "image only, no mappings",
			cfg: RunConfig{Image: "nginx:latest"},
		},
		{
			doc: "blank image rejected",
			cfg: RunConfig{Image: "   "}, wantErr: true,
		},
		{
			doc: "valid mapping",
			cfg: RunConfig{Image: "nginx", Ports: []RunPort{{Host: "8080", Container: "80"}}},
		},
		{
			doc: "blank port row skipped",
			cfg: RunConfig{Image: "nginx", Ports: []RunPort{{Host: "", Container: ""}}},
		},
		{
			doc: "half-filled port row rejected",
			cfg: RunConfig{Image: "nginx", Ports: []RunPort{{Host: "8080", Container: ""}}}, wantErr: true,
		},
		{
			// Docker Engine 29.1.3+ rejects a mapping that targets container port 0.
			doc: "container port 0 rejected",
			cfg: RunConfig{Image: "nginx", Ports: []RunPort{{Host: "8080", Container: "0"}}}, wantErr: true,
		},
		{
			doc: "host port 0 rejected",
			cfg: RunConfig{Image: "nginx", Ports: []RunPort{{Host: "0", Container: "80"}}}, wantErr: true,
		},
		{
			doc: "non-numeric port rejected",
			cfg: RunConfig{Image: "nginx", Ports: []RunPort{{Host: "http", Container: "80"}}}, wantErr: true,
		},
		{
			doc: "out of range port rejected",
			cfg: RunConfig{Image: "nginx", Ports: []RunPort{{Host: "70000", Container: "80"}}}, wantErr: true,
		},
		{
			doc: "blank mount row skipped",
			cfg: RunConfig{Image: "nginx", Mounts: []RunMount{{}}},
		},
		{
			doc: "valid mount",
			cfg: RunConfig{Image: "nginx", Mounts: []RunMount{{Host: "/data", Container: "/data"}}},
		},
		{
			doc: "half-filled mount row rejected",
			cfg: RunConfig{Image: "nginx", Mounts: []RunMount{{Host: "/data"}}}, wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			err := validateRunConfig(tc.cfg)
			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestBuildCreateBodyOmitsEmptyPortBindings(t *testing.T) {
	// Engine 29 deprecates back-filling an empty PortBindings map (removed in API
	// 1.53), so the keys must be absent rather than empty when there are no mappings.
	body, err := buildCreateBody(RunConfig{Image: "nginx:latest"})
	if err != nil {
		t.Fatalf("buildCreateBody() error = %v", err)
	}
	if _, ok := body["ExposedPorts"]; ok {
		t.Error("ExposedPorts must be omitted when there are no port mappings")
	}
	hostConfig, ok := body["HostConfig"].(map[string]interface{})
	if !ok {
		t.Fatal("HostConfig missing from request body")
	}
	if _, ok := hostConfig["PortBindings"]; ok {
		t.Error("PortBindings must be omitted when there are no port mappings")
	}
}

func TestBuildCreateBodyIncludesPortBindings(t *testing.T) {
	body, err := buildCreateBody(RunConfig{
		Image: "nginx:latest",
		Ports: []RunPort{{Host: "8080", Container: "80"}},
		Mounts: []RunMount{
			{Host: "/data", Container: "/data"},
			{}, // blank row must be skipped
		},
		Env: []string{"TZ=UTC"},
	})
	if err != nil {
		t.Fatalf("buildCreateBody() error = %v", err)
	}

	exposed, ok := body["ExposedPorts"].(map[string]interface{})
	if !ok || len(exposed) != 1 {
		t.Fatalf("ExposedPorts = %#v, want one entry", body["ExposedPorts"])
	}
	if _, ok := exposed["80/tcp"]; !ok {
		t.Errorf("ExposedPorts missing 80/tcp: %#v", exposed)
	}

	hostConfig := body["HostConfig"].(map[string]interface{})
	bindings, ok := hostConfig["PortBindings"].(map[string]interface{})
	if !ok || len(bindings) != 1 {
		t.Fatalf("PortBindings = %#v, want one entry", hostConfig["PortBindings"])
	}
	if _, ok := bindings["80/tcp"]; !ok {
		t.Errorf("PortBindings missing 80/tcp: %#v", bindings)
	}

	binds, ok := hostConfig["Binds"].([]string)
	if !ok || len(binds) != 1 || binds[0] != "/data:/data" {
		t.Errorf("Binds = %#v, want [/data:/data]", hostConfig["Binds"])
	}
}

func TestRunCLIPreviewMatchesDeployedMappings(t *testing.T) {
	cfg := RunConfig{
		Image: "nginx:latest",
		Name:  "web",
		Ports: []RunPort{
			{Host: "8080", Container: "80"},
			{},                           // blank row
			{Host: "", Container: "443"}, // half-filled row
		},
		Mounts: []RunMount{{Host: "/data", Container: "/data"}},
		Env:    []string{"TZ=UTC"},
	}

	want := "docker run -d --name web -p 8080:80 -v /data:/data -e TZ=UTC nginx:latest"
	if got := RunCLIPreview("docker", cfg); got != want {
		t.Errorf("RunCLIPreview() = %q, want %q", got, want)
	}
}
