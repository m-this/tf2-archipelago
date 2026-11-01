package sigmodload

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Refusal
		ok   bool
	}{
		{
			name: "keys, a binary differs",
			line: `[SM] Unable to load extension "sigsegv.ext.2.tf2.dll": server.dll is TimeDateStamp 0x6ad30000 SizeOfImage 0xcb7000, the address table for ServerVersion 11191711 was made from 0x6ad26f14 0xcb6000, and the server is 11200000`,
			want: Refusal{Lib: "server"},
			ok:   true,
		},
		{
			name: "keys, the DLL name is case-folded",
			line: `SoundEmitterSystem.dll is TimeDateStamp 0x1 SizeOfImage 0x2, the address table for ServerVersion 1 was made from 0x3 0x4, and the server is 5`,
			want: Refusal{Lib: "soundemittersystem"},
			ok:   true,
		},
		{
			name: "keys, no identity for a DLL",
			line: `the address table has addresses in vscript.dll and no identity for it`,
			want: Refusal{Lib: "vscript"},
			ok:   true,
		},
		{
			name: "keys, DLL not loaded",
			line: `the address table has addresses in engine.dll, which is not loaded`,
			want: Refusal{Lib: "engine"},
			ok:   true,
		},
		{
			name: "keys, identity unreadable",
			line: `the address table's identity of tier0.dll is "zz", not TimeDateStamp and SizeOfImage`,
			want: Refusal{Lib: "tier0"},
			ok:   true,
		},
		{
			name: "keys, no PE header",
			line: `dedicated.dll at 0x10000000 has no PE32 header`,
			want: Refusal{Lib: "dedicated"},
			ok:   true,
		},
		{
			name: "no keys, ServerVersion only",
			line: `[SM] Unable to load extension "sigsegv.ext.2.tf2.dll": this SigMod build is for TF2 ServerVersion 11191711, and the server is 11200000`,
			want: Refusal{Table: "11191711", Server: "11200000"},
			ok:   true,
		},
		{name: "extension loaded", line: `SigMod: address table made for ServerVersion 11191711, the server is 11200000; the 6 binaries it reads are the same`},
		{name: "unrelated line", line: `server.dll loaded`},
		{name: "DLL named without the table", line: `server.dll is TimeDateStamp of the day`},
		{name: "empty", line: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := Parse(test.line)
			if ok != test.ok || got != test.want {
				t.Fatalf("Parse(%q) = %+v, %v; want %+v, %v", test.line, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name    string
		refusal Refusal
		want    []string
	}{
		{"names the DLL", Refusal{Lib: "server"}, []string{"server.dll", "Update the launcher"}},
		{"names both builds", Refusal{Table: "1", Server: "2"}, []string{"build 1", "on 2", "Update the launcher"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.refusal.Message()
			for _, part := range test.want {
				if !strings.Contains(got, part) {
					t.Errorf("Message() = %q, missing %q", got, part)
				}
			}
		})
	}
}
