package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// RunConfig holds the options for deploying a new container.
type RunConfig struct {
	Image  string
	Name   string
	Ports  []RunPort
	Mounts []RunMount
	Env    []string // "KEY=VALUE"
}

type RunPort struct{ Host, Container string }
type RunMount struct{ Host, Container string }

// postJSON sends a JSON POST via an existing *http.Client and decodes the response.
// Uses the same pattern as the existing get/post/delete methods.
func postJSON(ctx context.Context, client *http.Client, url string, body interface{}, out interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(buf)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// validPortPair reports whether a port mapping row is filled in. A row that is left
// entirely blank means "no mapping" and is skipped; a half-filled row is a user error
// rather than a silent omission.
func validPortPair(p RunPort) bool {
	return strings.TrimSpace(p.Host) != "" && strings.TrimSpace(p.Container) != ""
}

// validatePort rejects the values Docker Engine 29.1.3 and newer refuse: an empty
// port, a non-numeric port, and port 0. Valid ports are 1-65535.
func validatePort(side, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s port is required", side)
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%s port %q is not a number", side, value)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("%s port %d is out of range (1-65535)", side, n)
	}
	return nil
}

// validateRunConfig catches the parts of a deploy request the daemon would otherwise
// reject with an opaque error. Blank port and mount rows are skipped, so an empty
// form still deploys an unmapped container.
func validateRunConfig(cfg RunConfig) error {
	if strings.TrimSpace(cfg.Image) == "" {
		return fmt.Errorf("image is required")
	}

	for _, p := range cfg.Ports {
		host, container := strings.TrimSpace(p.Host), strings.TrimSpace(p.Container)
		switch {
		case host == "" && container == "":
			continue // blank row — no mapping
		case host == "":
			return fmt.Errorf("host port is required when a container port is set")
		case container == "":
			return fmt.Errorf("container port is required when a host port is set")
		}
		if err := validatePort("host", host); err != nil {
			return err
		}
		if err := validatePort("container", container); err != nil {
			return err
		}
	}

	for _, m := range cfg.Mounts {
		host, container := strings.TrimSpace(m.Host), strings.TrimSpace(m.Container)
		switch {
		case host == "" && container == "":
			continue // blank row — no mount
		case host == "":
			return fmt.Errorf("mount host path is required when a container path is set")
		case container == "":
			return fmt.Errorf("mount container path is required when a host path is set")
		}
	}

	return nil
}

// buildCreateBody builds the container create request body.
//
// ExposedPorts and HostConfig.PortBindings are omitted entirely when there are no port
// mappings: Engine 29 deprecates back-filling an empty PortBindings map, and that
// compatibility behaviour is removed in API 1.53.
func buildCreateBody(cfg RunConfig) (map[string]interface{}, error) {
	if err := validateRunConfig(cfg); err != nil {
		return nil, err
	}

	portBindings := map[string]interface{}{}
	exposedPorts := map[string]interface{}{}
	for _, p := range cfg.Ports {
		if !validPortPair(p) {
			continue
		}
		key := strings.TrimSpace(p.Container) + "/tcp"
		exposedPorts[key] = struct{}{}
		portBindings[key] = []map[string]string{{"HostPort": strings.TrimSpace(p.Host)}}
	}

	var binds []string
	for _, m := range cfg.Mounts {
		if strings.TrimSpace(m.Host) == "" || strings.TrimSpace(m.Container) == "" {
			continue
		}
		binds = append(binds, m.Host+":"+m.Container)
	}

	env := cfg.Env
	if env == nil {
		env = []string{}
	}

	hostConfig := map[string]interface{}{
		"Binds": binds,
	}
	if len(portBindings) > 0 {
		hostConfig["PortBindings"] = portBindings
	}

	body := map[string]interface{}{
		"Image":      cfg.Image,
		"Env":        env,
		"HostConfig": hostConfig,
	}
	if len(portBindings) > 0 {
		body["ExposedPorts"] = exposedPorts
	}
	return body, nil
}

// RunContainer creates and starts a container via the Docker socket API.
func (d *Docker) RunContainer(ctx context.Context, cfg RunConfig) (string, error) {
	path := "/containers/create"
	if cfg.Name != "" {
		path += "?name=" + cfg.Name
	}
	var resp struct {
		ID string `json:"Id"`
	}
	body, err := buildCreateBody(cfg)
	if err != nil {
		return "", err
	}
	if err := postJSON(ctx, d.client, d.url(path), body, &resp); err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	if resp.ID == "" {
		return "", fmt.Errorf("no container ID returned")
	}
	if err := d.post(ctx, fmt.Sprintf("/containers/%s/start", resp.ID)); err != nil {
		return resp.ID, fmt.Errorf("start: %w", err)
	}
	return resp.ID, nil
}

// RunContainer creates and starts a container via the Podman socket API.
func (p *Podman) RunContainer(ctx context.Context, cfg RunConfig) (string, error) {
	path := "/containers/create"
	if cfg.Name != "" {
		path += "?name=" + cfg.Name
	}
	var resp struct {
		ID string `json:"Id"`
	}
	body, err := buildCreateBody(cfg)
	if err != nil {
		return "", err
	}
	if err := postJSON(ctx, p.client, p.url(path), body, &resp); err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	if resp.ID == "" {
		return "", fmt.Errorf("no container ID returned")
	}
	if err := p.post(ctx, fmt.Sprintf("/containers/%s/start", resp.ID)); err != nil {
		return resp.ID, fmt.Errorf("start: %w", err)
	}
	return resp.ID, nil
}

// RunCLIPreview returns the equivalent CLI command for Learn Mode.
// It mirrors buildCreateBody's filtering so the preview matches what is deployed.
func RunCLIPreview(runtime string, cfg RunConfig) string {
	var sb strings.Builder
	sb.WriteString(runtime + " run -d")
	if cfg.Name != "" {
		sb.WriteString(" --name " + cfg.Name)
	}
	for _, p := range cfg.Ports {
		if validPortPair(p) {
			sb.WriteString(fmt.Sprintf(" -p %s:%s", p.Host, p.Container))
		}
	}
	for _, m := range cfg.Mounts {
		if m.Host != "" && m.Container != "" {
			sb.WriteString(fmt.Sprintf(" -v %s:%s", m.Host, m.Container))
		}
	}
	for _, e := range cfg.Env {
		sb.WriteString(" -e " + e)
	}
	sb.WriteString(" " + cfg.Image)
	return sb.String()
}
