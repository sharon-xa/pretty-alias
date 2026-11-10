package shell

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    Shell
		wantErr bool
	}{
		{"bash", Bash, false},
		{"zsh", Zsh, false},
		{"fish", Fish, false},
		{"/bin/bash", Bash, false},
		{"/usr/bin/zsh", Zsh, false},
		{"/usr/local/bin/fish", Fish, false},
		{"  /usr/bin/fish\n", Fish, false},
		{"/usr/bin/ZSH", Zsh, false},

		// substring matches must not count (the old code used strings.Contains)
		{"/home/zsh-fan/bin/mysh", "", true},
		{"/usr/bin/zsh-wrapper", "", true},
		{"/opt/fish/bin/sh", "", true},
		{"bashful", "", true},

		{"/bin/sh", "", true},
		{"/usr/bin/nu", "", true},
		{"", "", true},
		{"/", "", true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if tt.wantErr {
			var ue *UnsupportedError
			if !errors.As(err, &ue) {
				t.Errorf("Parse(%q) error is %T, want *UnsupportedError", tt.in, err)
			}
		}
	}
}

func TestDetectMatchesParse(t *testing.T) {
	if got, err := Detect("/usr/bin/fish"); err != nil || got != Fish {
		t.Errorf("Detect = %q, %v", got, err)
	}
}

func TestErrorMessagesAreActionable(t *testing.T) {
	_, err := Parse("")
	if err == nil || !contains(err.Error(), "--shell") {
		t.Errorf("empty $SHELL error should mention --shell: %v", err)
	}
	_, err = Parse("/bin/sh")
	if err == nil || !contains(err.Error(), "bash, zsh, fish") || !contains(err.Error(), "/bin/sh") {
		t.Errorf("unsupported error should name the shell and the supported ones: %v", err)
	}
}

func TestConfigFile(t *testing.T) {
	tests := []struct {
		sh   Shell
		want string
	}{
		{Bash, "/home/u/.bashrc"},
		{Zsh, "/home/u/.zshrc"},
		{Fish, "/home/u/.config/fish/config.fish"},
	}
	for _, tt := range tests {
		if got := tt.sh.ConfigFile("/home/u"); got != tt.want {
			t.Errorf("%s.ConfigFile = %q, want %q", tt.sh, got, tt.want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
