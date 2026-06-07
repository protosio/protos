package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListenAddr = ":8080"
	defaultTimeout    = 8 * time.Second
	defaultMaxBytes   = 4096
	maxProbeBytes     = 1024 * 1024
)

type server struct {
	id         string
	httpClient *http.Client
}

type probeResponse struct {
	OK             bool   `json:"ok"`
	ID             string `json:"id"`
	Target         string `json:"target"`
	StatusCode     int    `json:"status_code,omitempty"`
	BytesRead      int    `json:"bytes_read,omitempty"`
	DurationMillis int64  `json:"duration_ms"`
	Error          string `json:"error,omitempty"`
}

func main() {
	listenAddr := flag.String("listen", envOrDefault("PROTOS_E2E_PROBE_LISTEN", defaultListenAddr), "HTTP listen address")
	id := flag.String("id", envOrDefault("PROTOS_E2E_PROBE_ID", hostname()), "probe instance id")
	flag.Parse()

	s := &server{
		id: *id,
		httpClient: &http.Client{
			Timeout: defaultTimeout + time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/probe", s.handleProbe)

	httpServer := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "probe server failed: %v\n", err)
		os.Exit(1)
	}
}

func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"id": s.id,
	})
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"id": s.id,
	})
}

func (s *server) handleProbe(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	timeout := durationParam(r, "timeout_ms", defaultTimeout, 30*time.Second)
	maxBytes := intParam(r, "max_bytes", defaultMaxBytes, maxProbeBytes)
	resp := probeResponse{ID: s.id, Target: target}

	var statusCode int
	if err := validateTarget(target); err != nil {
		resp.Error = err.Error()
		statusCode = http.StatusBadRequest
	} else {
		probeCtx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, target, nil)
		if err != nil {
			resp.Error = fmt.Sprintf("build request: %v", err)
			statusCode = http.StatusBadRequest
		} else {
			statusCode = s.runProbe(req, maxBytes, &resp)
		}
	}

	resp.DurationMillis = time.Since(start).Milliseconds()
	writeJSON(w, statusCode, resp)
}

func (s *server) runProbe(req *http.Request, maxBytes int, out *probeResponse) int {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		out.Error = err.Error()
		return http.StatusBadGateway
	}
	defer resp.Body.Close()

	out.StatusCode = resp.StatusCode
	limited := io.LimitReader(resp.Body, int64(maxBytes))
	body, err := io.ReadAll(limited)
	if err != nil {
		out.Error = fmt.Sprintf("read response: %v", err)
		return http.StatusBadGateway
	}
	out.BytesRead = len(body)
	out.OK = resp.StatusCode >= 200 && resp.StatusCode < 300 && len(body) > 0
	if !out.OK {
		out.Error = fmt.Sprintf("target returned status %d with %d byte(s)", resp.StatusCode, len(body))
		return http.StatusBadGateway
	}
	return http.StatusOK
}

func validateTarget(target string) error {
	if target == "" {
		return fmt.Errorf("target is required")
	}
	parsed, err := url.ParseRequestURI(target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported target URL scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("target URL host is required")
	}
	return nil
}

func durationParam(r *http.Request, name string, fallback time.Duration, max time.Duration) time.Duration {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback
	}
	ms, err := strconv.Atoi(value)
	if err != nil || ms <= 0 {
		return fallback
	}
	duration := time.Duration(ms) * time.Millisecond
	if duration > max {
		return max
	}
	return duration
}

func intParam(r *http.Request, name string, fallback int, max int) int {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		http.Error(w, fmt.Sprintf("encode json: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

func envOrDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func hostname() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "protos-e2e-probe"
	}
	return host
}
