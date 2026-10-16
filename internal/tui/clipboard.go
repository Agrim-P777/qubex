package tui

import (
	"bytes"
	"os/exec"
	"runtime"

	tea "charm.land/bubbletea/v2"
)

func writeClipboard(s string) tea.Cmd {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = bytes.NewBufferString(s)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return tea.SetClipboard(s)
}
