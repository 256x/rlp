package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

var mpvProc *exec.Cmd
var mpvGeneration int64

func stopStation() {
	atomic.AddInt64(&mpvGeneration, 1)
	if mpvProc != nil {
		_ = mpvProc.Process.Kill()
		mpvProc = nil
	}
	killSavedMpv()
}

func savePID(pid int) {
	path := cachePath("mpv.pid")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

func isMpvRunning() bool {
	data, err := os.ReadFile(cachePath("mpv.pid"))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// signal 0 checks if process exists without killing it
	return p.Signal(syscall.Signal(0)) == nil
}

func killSavedMpv() {
	data, err := os.ReadFile(cachePath("mpv.pid"))
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
	_ = os.Remove(cachePath("mpv.pid"))
}

func playCmd(s Station) tea.Cmd {
	return func() tea.Msg {
		stopStation()
		_ = os.Remove(mpvSocket)
		cmd := exec.Command("mpv", "--no-video", "--no-terminal", "--really-quiet",
			"--input-ipc-server="+mpvSocket, s.URL)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			return statusMsg{Text: "failed to play: " + err.Error()}
		}
		mpvProc = cmd
		gen := atomic.AddInt64(&mpvGeneration, 1)
		savePID(cmd.Process.Pid)
		_ = SaveCurrentStation(s)
		go func() {
			cmd.Wait()
			if atomic.LoadInt64(&mpvGeneration) == gen && rlpProgram != nil {
				rlpProgram.Send(mpvExitedMsg{})
			}
		}()
		return playStartedMsg{Station: s}
	}
}
