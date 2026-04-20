package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	stylePopupBorder lipgloss.Style
	styleSelected    lipgloss.Style
	styleTitle       lipgloss.Style
	styleFilter      lipgloss.Style
	styleAccent      lipgloss.Style
	styleDim         lipgloss.Style
	styleTab         lipgloss.Style
	styleTabActive   lipgloss.Style
)

func initStyles() {
	accent := lipgloss.Color("#84a0c6")
	stylePopupBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 1)
	styleSelected = lipgloss.NewStyle().
		Background(accent).
		Foreground(lipgloss.Color("#c6c8d1")).
		Bold(true)
	styleTitle = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)
	styleFilter = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89b8c2"))
	styleAccent = lipgloss.NewStyle().
		Foreground(accent)
	styleDim = lipgloss.NewStyle().
		Faint(true)
	styleTab = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6b7089"))
	styleTabActive = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)
}

func (m model) View() string {
	if !m.ready {
		return ""
	}
	switch m.screen {
	case screenSearch:
		return m.renderWithSearchPopup()
	case screenStation:
		return m.renderWithStationPopup()
	}
	return m.renderPlayerLine()
}

func (m model) renderPlayerLine() string {
	status := m.currentStatus()
	if status != "" {
		return truncate(status, m.width)
	}
	if m.current == nil {
		return styleAccent.Render("-") + styleDim.Render(" rlp - [space] select/search")
	}
	spinFrames := []string{"-", "\\", "|", "/"}
	sym := spinFrames[m.spinFrame%len(spinFrames)]
	if !m.connecting {
		sym = "-"
	}
	prefix := styleAccent.Render(sym) + " "
	info := m.current.Name
	if m.current.Country != "" {
		info += " @ " + m.current.Country
	}
	suffix := ""
	if m.current.Bitrate > 0 {
		suffix = styleDim.Render(fmt.Sprintf("  %dkbps", m.current.Bitrate))
	}
	prefixW := lipgloss.Width(prefix)
	suffixW := lipgloss.Width(suffix)
	return prefix + truncate(info, m.width-prefixW-suffixW) + suffix
}

func (m model) renderWithSearchPopup() string {
	popup := m.buildSearchPopup()
	hints := styleDim.Render("←→:mode  ↑↓:move  type:filter  enter:select  esc:back")
	return overlay(hints, popup, m.width, m.height)
}

func (m model) buildSearchPopup() string {
	maxW := m.width - 4
	if maxW > 60 {
		maxW = 60
	}
	if maxW < 20 {
		maxW = 20
	}
	innerW := maxW - 4

	var sb strings.Builder

	// tabs
	tabs := []struct {
		key   string
		label string
		mode  searchMode
	}{
		{"1", "Genre", modeGenre},
		{"2", "Country", modeCountry},
		{"3", "Language", modeLanguage},
		{"/", "Name", modeName},
	}
	var tabParts []string
	for _, t := range tabs {
		label := t.key + ":" + t.label
		if m.searchMode == t.mode {
			tabParts = append(tabParts, styleTabActive.Render(label))
		} else {
			tabParts = append(tabParts, styleTab.Render(label))
		}
	}
	tabLine := strings.Join(tabParts, styleDim.Render("  "))
	sb.WriteString(tabLine)
	sb.WriteString("\n")
	sb.WriteString(styleDim.Render(strings.Repeat("─", innerW)))
	sb.WriteString("\n")

	if m.searchLoading {
		sb.WriteString(styleDim.Render("loading..."))
		content := strings.TrimRight(sb.String(), "\n")
		return stylePopupBorder.Width(maxW).Render(content)
	}

	// filter input
	indicator := styleFilter.Render("❯ ")
	f := runewidth.Truncate(m.searchFilter, innerW-2, "…")
	sb.WriteString(indicator + f + "█")
	sb.WriteString("\n")
	sb.WriteString(styleDim.Render(strings.Repeat("─", innerW)))
	sb.WriteString("\n")

	if m.searchMode == modeName {
		sb.WriteString(styleDim.Render("type to search, enter to find stations"))
		content := strings.TrimRight(sb.String(), "\n")
		return stylePopupBorder.Width(maxW).Render(content)
	}

	// list
	items := m.filteredItems
	if len(items) == 0 {
		sb.WriteString(styleDim.Render("(no match)"))
		content := strings.TrimRight(sb.String(), "\n")
		return stylePopupBorder.Width(maxW).Render(content)
	}

	maxRows := listMaxRows(m.height)
	cursor := m.searchCursor
	start := m.searchStart
	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	const prefixW = 2
	textW := innerW - prefixW

	for i := start; i < end; i++ {
		text := items[i]
		if textW > 0 {
			text = runewidth.Truncate(text, textW, "…")
			text = runewidth.FillRight(text, textW)
		}
		if i == cursor {
			sb.WriteString(styleSelected.Render("❯ " + text))
		} else {
			sb.WriteString("  " + text)
		}
		sb.WriteString("\n")
	}

	// scroll indicator
	if len(items) > 1 {
		scroll := fmt.Sprintf("%d/%d", cursor+1, len(items))
		sb.WriteString(styleDim.Render(strings.Repeat(" ", innerW-len(scroll)) + scroll))
	}

	content := strings.TrimRight(sb.String(), "\n")
	return stylePopupBorder.Width(maxW).Render(content)
}

func (m model) renderWithStationPopup() string {
	if m.stationLoading {
		popup := renderPopupBox("stations", []string{"loading..."}, nil, -1, -1, 0, "", false, m.width, m.height)
		return overlay(m.renderPlayerLine(), popup, m.width, m.height)
	}
	items := make([]string, len(m.stations))
	rightLabels := make([]string, len(m.stations))
	for i, s := range m.stations {
		items[i] = s.Name
		parts := []string{}
		if s.Country != "" {
			parts = append(parts, s.Country)
		}
		if s.Bitrate > 0 {
			parts = append(parts, fmt.Sprintf("%dkbps", s.Bitrate))
		}
		rightLabels[i] = strings.Join(parts, " ")
	}
	if len(items) == 0 {
		items = []string{"(no stations)"}
		rightLabels = nil
	}
	popup := renderPopupBox("stations", items, rightLabels, m.stationCursor, len(m.stations), m.stationStart, "", false, m.width, m.height)
	return overlay(m.renderPlayerLine(), popup, m.width, m.height)
}

func renderPopupBox(title string, items, rightLabels []string, cursor, total, start int, filter string, filterActive bool, termW, termH int) string {
	maxW := termW - 4
	if maxW > 60 {
		maxW = 60
	}
	if maxW < 20 {
		maxW = 20
	}

	maxRows := listMaxRows(termH)

	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	innerW := maxW - 4
	faint := lipgloss.NewStyle().Faint(true)

	var sb strings.Builder

	scrollStr := ""
	if total > 1 && cursor >= 0 {
		scrollStr = fmt.Sprintf("%d/%d", cursor+1, total)
	}
	titleMax := innerW
	if scrollStr != "" {
		titleMax = innerW - runewidth.StringWidth(scrollStr) - 1
	}
	titleRendered := styleTitle.Render(runewidth.Truncate(title, titleMax, ""))
	if scrollStr != "" {
		titleW := lipgloss.Width(titleRendered)
		pad := innerW - titleW - runewidth.StringWidth(scrollStr)
		if pad < 1 {
			pad = 1
		}
		sb.WriteString(titleRendered + strings.Repeat(" ", pad) + faint.Render(scrollStr))
	} else {
		sb.WriteString(titleRendered)
	}
	sb.WriteString("\n")

	if filterActive || filter != "" {
		indicator := "❯ "
		if filterActive {
			indicator = styleFilter.Render("❯ ")
		}
		f := runewidth.Truncate(filter, innerW-2, "…")
		sb.WriteString(indicator + f)
		if filterActive {
			sb.WriteString("█")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(faint.Render(strings.Repeat("─", innerW)))
	sb.WriteString("\n")

	const prefixW = 2
	avail := innerW - prefixW

	for i := start; i < end; i++ {
		rl := ""
		if rightLabels != nil && i < len(rightLabels) {
			rl = rightLabels[i]
		}
		name := items[i]
		nameW := runewidth.StringWidth(name)
		rlW := runewidth.StringWidth(rl)

		var displayName, displayRl string
		if rl == "" {
			displayName = runewidth.FillRight(name, avail)
		} else if nameW+1+rlW <= avail {
			pad := avail - nameW - rlW
			displayName = name + strings.Repeat(" ", pad)
			displayRl = rl
		} else {
			remaining := avail - nameW - 1
			if remaining >= 1 {
				displayRl = runewidth.Truncate(rl, remaining, "…")
				displayName = name + " "
			} else {
				displayName = runewidth.FillRight(name, avail)
			}
		}

		if i == cursor {
			sb.WriteString(styleSelected.Render("❯ " + displayName + displayRl))
		} else {
			sb.WriteString("  " + displayName)
			if displayRl != "" {
				sb.WriteString(faint.Render(displayRl))
			}
		}
		sb.WriteString("\n")
	}

	content := strings.TrimRight(sb.String(), "\n")
	return stylePopupBorder.Width(maxW).Render(content)
}

func overlay(base, popup string, termW, termH int) string {
	if termW <= 0 || termH <= 0 {
		return base
	}

	popupLines := strings.Split(popup, "\n")
	popupH := len(popupLines)

	popupW := 0
	for _, l := range popupLines {
		if w := lipgloss.Width(l); w > popupW {
			popupW = w
		}
	}

	startRow := (termH - popupH) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (termW - popupW) / 2
	if startCol < 0 {
		startCol = 0
	}

	blank := strings.Repeat(" ", termW)
	var sb strings.Builder

	for row := 0; row < termH-1; row++ {
		if row >= startRow && row < startRow+popupH {
			line := popupLines[row-startRow]
			lineW := lipgloss.Width(line)
			rightPad := termW - startCol - lineW
			if rightPad < 0 {
				rightPad = 0
			}
			sb.WriteString(strings.Repeat(" ", startCol))
			sb.WriteString(line)
			sb.WriteString(strings.Repeat(" ", rightPad))
		} else {
			sb.WriteString(blank)
		}
		if row < termH-2 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(base)
	return sb.String()
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 1 {
		return "…"
	}
	return string(runes[:maxRunes-1]) + "…"
}
