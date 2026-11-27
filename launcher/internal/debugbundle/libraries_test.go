package debugbundle

import (
	"errors"
	"strings"
	"testing"
)

/*
LuigiFan45's and Lee Harrold's bundles held a replay_srv.so crash loop with a
steamclient link that resolved, and nothing that said which library the server
lacked. The summary says what ldd says.
*/
func TestLibrariesNote(t *testing.T) {
	tests := []struct {
		name    string
		missing []string
		err     error
		standIn string
		want    string
	}{
		{name: "all found", want: "all found by ldd"},
		{name: "stood in", standIn: "/usr/lib/libcurl.so.4", want: "all found by ldd, libcurl-gnutls.so.4 -> /usr/lib/libcurl.so.4"},
		{name: "missing", missing: []string{"libcurl-gnutls.so.4"}, want: "MISSING libcurl-gnutls.so.4"},
		{name: "not checked", err: errors.New("ldd: exit status 1"), want: "NOT CHECKED: ldd: exit status 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := librariesNote(tt.missing, tt.err, tt.standIn); !strings.HasPrefix(got, tt.want) {
				t.Errorf("librariesNote = %q, want it to start with %q", got, tt.want)
			}
		})
	}
}
