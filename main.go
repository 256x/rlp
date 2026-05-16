package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "1.0.0"

var rlpProgram *tea.Program

func main() {
	selectFlag := flag.Bool("select", false, "selection popup mode (used with tmux display-popup)")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println("rlp v" + version)
		return
	}

	initStyles()
	initGradient()
	m := newModel(*selectFlag)

	if !*selectFlag {
		defer killSavedMpv()
	}

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	p := tea.NewProgram(m, opts...)
	rlpProgram = p
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
