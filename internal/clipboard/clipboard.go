// Package clipboard copies text using the platform clipboard helper when
// available and falls back to an OSC 52 terminal escape sequence.
package clipboard

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type tool struct {
	name string
	args []string
}

var tools = []tool{
	{"wl-copy", nil},
	{"xclip", []string{"-selection", "clipboard"}},
	{"xsel", []string{"--input", "--clipboard"}},
	{"pbcopy", nil},
	{"clip.exe", nil},
}

// Copy places text on the clipboard. The returned string is the OSC 52 escape
// sequence that should be written to the terminal when no helper is installed.
func Copy(text string) (osc52 string, err error) {
	for _, t := range tools {
		path, lookErr := exec.LookPath(t.name)
		if lookErr != nil {
			continue
		}
		cmd := exec.Command(path, t.args...)
		cmd.Stdin = strings.NewReader(text)
		if runErr := cmd.Run(); runErr == nil {
			return "", nil
		}
	}
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("no clipboard helper available")
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return "\x1b]52;c;" + encoded + "\a", nil
}
