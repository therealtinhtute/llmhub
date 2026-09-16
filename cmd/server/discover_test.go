package main

import "testing"

// TestArgvEnablesBoolFlag mirrors upstream main_test.go (13af6c002bd0): the
// pre-scan must detect -discover-json/--discover-json in every spelling so the
// startup banner stays off stdout, and must stop at "--" or the first
// non-flag argument instead of scanning the whole argv.
func TestArgvEnablesBoolFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{name: "bare long flag", args: []string{"--discover-json"}, flag: "discover-json", want: true},
		{name: "assigned true", args: []string{"--discover-json=true"}, flag: "discover-json", want: true},
		{name: "assigned false", args: []string{"--discover-json=false"}, flag: "discover-json", want: false},
		{name: "single dash", args: []string{"-discover-json"}, flag: "discover-json", want: true},
		{name: "does not match timeout", args: []string{"--discover-timeout", "3"}, flag: "discover", want: false},
		{name: "bare discover", args: []string{"--discover"}, flag: "discover", want: true},
		{name: "stops at terminator", args: []string{"--", "--discover-json"}, flag: "discover-json", want: false},
		{name: "stops at non-flag", args: []string{"foo", "--discover-json"}, flag: "discover-json", want: false},
		{name: "skips value of discover-config", args: []string{"--discover-config", "config.yaml", "--discover-json"}, flag: "discover-json", want: true},
		{name: "value flag eats next token", args: []string{"--discover-config", "--discover-json"}, flag: "discover-json", want: false},
		{name: "value flag eats terminator", args: []string{"--discover-config", "--", "--discover-json"}, flag: "discover-json", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := argvEnablesBoolFlag(tt.args, tt.flag); got != tt.want {
				t.Fatalf("argvEnablesBoolFlag(%v, %q) = %t, want %t", tt.args, tt.flag, got, tt.want)
			}
		})
	}
}

// TestEarlyCommandDispatchDiscover wires `llmhub discover` into
// dispatchEarlyCommand. The -h path exits through flag.ErrHelp -> usage code 2
// without touching the network.
func TestEarlyCommandDispatchDiscover(t *testing.T) {
	if code, ok := dispatchEarlyCommand([]string{"discover", "-h"}); !ok || code != 2 {
		t.Fatalf("dispatchEarlyCommand(discover -h) = (%d, %t), want (2, true)", code, ok)
	}
}
