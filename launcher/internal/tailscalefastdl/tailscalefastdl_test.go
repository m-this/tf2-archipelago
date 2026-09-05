package tailscalefastdl

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestConfigurePublishesOnlyTheLoopbackServer(t *testing.T) {
	var calls [][]string
	run := func(_ context.Context, executable string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{executable}, args...))
		if reflect.DeepEqual(args, []string{"status", "--json"}) {
			return []byte(`{"BackendState":"Running","Self":{"DNSName":"host.example.ts.net."}}`), nil
		}
		return nil, nil
	}

	got, err := configure(context.Background(), "tailscale", 27080, run)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://host.example.ts.net/tf" {
		t.Fatalf("result = %#v", got)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %#v", calls)
	}
	joined := strings.Join(calls[1], " ")
	if joined != "tailscale funnel --yes --bg --https=443 --set-path=/tf http://127.0.0.1:27080/tf" {
		t.Errorf("funnel call = %s", joined)
	}
}

func TestConfigureExplainsSignedOutTailscale(t *testing.T) {
	run := func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"BackendState":"NeedsLogin","Self":{}}`), nil
	}
	_, err := configure(context.Background(), "tailscale", 27080, run)
	if err == nil || !strings.Contains(err.Error(), "sign in") {
		t.Fatalf("error = %v", err)
	}
}

func TestConfigureCarriesFunnelInstructions(t *testing.T) {
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if reflect.DeepEqual(args, []string{"status", "--json"}) {
			return []byte(`{"BackendState":"Running","Self":{"DNSName":"host.example.ts.net"}}`), nil
		}
		return nil, errors.New("Access denied; use an Administrator terminal")
	}
	_, err := configure(context.Background(), "tailscale", 27080, run)
	if err == nil || !strings.Contains(err.Error(), "Administrator") {
		t.Fatalf("error = %v", err)
	}
}

func TestAuthorizeReturnsTheBrowserApproval(t *testing.T) {
	run := func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("Funnel is disabled; visit https://login.tailscale.com/f/funnel?node=abc123")
	}
	got, err := authorize(context.Background(), "tailscale", run)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApprovalURL != "https://login.tailscale.com/f/funnel?node=abc123" || got.Ready {
		t.Fatalf("authorization = %#v", got)
	}
}

func TestAuthorizeRemovesItsPublicSetupPath(t *testing.T) {
	var calls [][]string
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	got, err := authorize(context.Background(), "tailscale", run)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Ready || got.ApprovalURL != "" {
		t.Fatalf("authorization = %#v", got)
	}
	if len(calls) != 2 || calls[1][len(calls[1])-1] != "off" {
		t.Fatalf("calls = %#v", calls)
	}
}
