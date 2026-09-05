package runtime

import (
	"context"
	"reflect"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func TestTailscaleFastDLOffChangesNothing(t *testing.T) {
	s := settings.Settings{FastDLPort: 27080, SrcdsDownloadURL: "https://example.test/tf"}
	got := prepareTailscaleFastDL(context.Background(), s, func(string) {})
	if !reflect.DeepEqual(got, s) {
		t.Fatalf("settings changed: got %#v, want %#v", got, s)
	}
}
