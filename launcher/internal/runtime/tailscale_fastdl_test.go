package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/tailscalefastdl"
)

func TestTailscaleFastDLOffChangesNothing(t *testing.T) {
	s := settings.Settings{FastDLPort: 27080, SrcdsDownloadURL: "https://example.test/tf"}
	got, err := prepareTailscaleFastDLWith(context.Background(), s, func(string) {}, func(context.Context, int) (tailscalefastdl.Result, error) {
		t.Fatal("configure called while Tailscale FastDL was off")
		return tailscalefastdl.Result{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, s) {
		t.Fatalf("settings changed: got %#v, want %#v", got, s)
	}
}

func TestTailscaleFastDLIsReappliedAtEveryStart(t *testing.T) {
	s := settings.Settings{TailscaleFastDL: true}
	var port int
	var said string
	got, err := prepareTailscaleFastDLWith(context.Background(), s, func(message string) { said = message }, func(_ context.Context, gotPort int) (tailscalefastdl.Result, error) {
		port = gotPort
		return tailscalefastdl.Result{URL: "https://host.example.ts.net/tf"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if port != 27080 || got.FastDLPort != 27080 || got.SrcdsDownloadURL != "https://host.example.ts.net/tf" {
		t.Fatalf("prepared settings = %#v; configured port = %d", got, port)
	}
	if !strings.Contains(said, "ready") {
		t.Fatalf("message = %q", said)
	}
}

func TestTailscaleFastDLFailureStopsStart(t *testing.T) {
	s := settings.Settings{TailscaleFastDL: true, FastDLPort: 27080}
	want := errors.New("signed out")
	_, err := prepareTailscaleFastDLWith(context.Background(), s, func(string) {}, func(context.Context, int) (tailscalefastdl.Result, error) {
		return tailscalefastdl.Result{}, want
	})
	var startErr *TailscaleFastDLStartError
	if !errors.As(err, &startErr) || !errors.Is(err, want) || startErr.ApprovalURL != "" {
		t.Fatalf("error = %#v", err)
	}
	if !strings.Contains(err.Error(), "server was not started") {
		t.Fatalf("error did not explain the outcome: %v", err)
	}
}

func TestTailscaleFastDLApprovalIsCarriedToTheInterface(t *testing.T) {
	s := settings.Settings{TailscaleFastDL: true, FastDLPort: 27080}
	wantURL := "https://login.tailscale.com/f/funnel?node=abc123"
	_, err := prepareTailscaleFastDLWith(context.Background(), s, func(string) {}, func(context.Context, int) (tailscalefastdl.Result, error) {
		return tailscalefastdl.Result{}, &tailscalefastdl.ApprovalRequiredError{URL: wantURL}
	})
	var startErr *TailscaleFastDLStartError
	if !errors.As(err, &startErr) || startErr.ApprovalURL != wantURL {
		t.Fatalf("error = %#v", err)
	}
}

func TestStoppingTailscaleFastDLRemovesItsRoute(t *testing.T) {
	s := settings.Settings{TailscaleFastDL: true}
	called := false
	var said string
	stopTailscaleFastDLWith(context.Background(), s, func(_ context.Context, message string) { said = message }, func(context.Context) error {
		called = true
		return nil
	})
	if !called {
		t.Fatal("Disable was not called")
	}
	if !strings.Contains(said, "removed") {
		t.Fatalf("message = %q", said)
	}
}

func TestStoppingTailscaleFastDLOffDoesNothing(t *testing.T) {
	stopTailscaleFastDLWith(context.Background(), settings.Settings{}, func(context.Context, string) {
		t.Fatal("cleanup spoke while FastDL was off")
	}, func(context.Context) error {
		t.Fatal("Disable was called while FastDL was off")
		return nil
	})
}

func TestStoppingTailscaleFastDLReportsCleanupFailure(t *testing.T) {
	want := errors.New("permission denied")
	var said string
	stopTailscaleFastDLWith(context.Background(), settings.Settings{TailscaleFastDL: true}, func(_ context.Context, message string) {
		said = message
	}, func(context.Context) error {
		return want
	})
	if !strings.Contains(said, want.Error()) {
		t.Fatalf("message = %q", said)
	}
}
