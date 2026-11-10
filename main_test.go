package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type mockTransport func(*http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func setupCLI(t *testing.T, privateKey, publicKey, failedPath string) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile("token", []byte("test-token"), 0600); err != nil {
		t.Fatal(err)
	}
	oldArgs, oldFlags, oldTransport := os.Args, flag.CommandLine, http.DefaultTransport
	t.Cleanup(func() {
		os.Args, flag.CommandLine, http.DefaultTransport = oldArgs, oldFlags, oldTransport
	})
	os.Args = []string{"norrWGConfig", "fi", "example.test"}
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	http.DefaultTransport = mockTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == failedPath {
			return nil, errors.New("simulated API failure")
		}
		var body string
		switch req.URL.Path {
		case "/v1/countries":
			body = `[{"id":73,"code":"FI","name":"Finland"}]`
		case "/v1/users/services/credentials":
			key, _ := json.Marshal(privateKey)
			body = fmt.Sprintf(`{"nordlynx_private_key":%s}`, key)
		case "/v1/servers/recommendations":
			key, _ := json.Marshal(publicKey)
			body = fmt.Sprintf(`[{"hostname":"127.0.0.1","technologies":[{"identifier":"wireguard_udp","metadata":[{"value":%s}]}],"locations":[{"country":{"code":"FI"}}]}]`, key)
		default:
			return nil, fmt.Errorf("unexpected request path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
}

func TestAPIFailureOrInvalidKeyPreservesConfig(t *testing.T) {
	validKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	shortKey := base64.StdEncoding.EncodeToString([]byte("too short"))
	cases := []struct {
		name, privateKey, publicKey, failedPath string
	}{
		{"private API failure", validKey, validKey, "/v1/users/services/credentials"},
		{"server API failure", validKey, validKey, "/v1/servers/recommendations"},
		{"empty private key", "", validKey, ""},
		{"empty public key", validKey, "", ""},
		{"malformed private key", "not-base64!", validKey, ""},
		{"malformed public key", validKey, "not-base64!", ""},
		{"short private key", shortKey, validKey, ""},
		{"short public key", validKey, shortKey, ""},
	}
	for _, tc := range cases {
		for _, exists := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing=%t", tc.name, exists), func(t *testing.T) {
				setupCLI(t, tc.privateKey, tc.publicKey, tc.failedPath)
				const original = "existing working configuration\n"
				if exists {
					if err := os.WriteFile("fi.conf", []byte(original), 0600); err != nil {
						t.Fatal(err)
					}
				}
				var failure any
				func() {
					defer func() { failure = recover() }()
					main()
				}()
				if failure == nil {
					t.Fatal("expected generation to fail")
				}
				data, err := os.ReadFile("fi.conf")
				if exists {
					if err != nil || string(data) != original {
						t.Fatalf("existing config was changed: read error %v", err)
					}
				} else if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("failed generation created a config: read error %v", err)
				}
			})
		}
	}
}

func TestSuccessfulGenerationTruncatesConfig(t *testing.T) {
	privateKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
	publicKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	setupCLI(t, privateKey, publicKey, "")
	if err := os.WriteFile("fi.conf", []byte(strings.Repeat("stale content\n", 100)), 0600); err != nil {
		t.Fatal(err)
	}
	main()
	data, err := os.ReadFile("fi.conf")
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(template, privateKey, publicKey, "fi", "example.test")
	if string(data) != want {
		t.Fatal("generated config differs from expected content or contains stale bytes")
	}
}
