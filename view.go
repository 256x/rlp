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
		return styleAccent.Render("[rlp]") + styleDim.Render(" [space] select/search")
	}

	prefix := styleAccent.Render("[rlp]") + " "
	prefixW := lipgloss.Width(prefix)
	avail := m.width - prefixW

	station := m.current.Name
	if m.current.Country != "" {
		station += " [" + abbrevCountry(m.current.Country) + "]"
	}

	const sep = "  |  "
	var line string
	if m.trackTitle == "" {
		line = truncate(station, avail)
	} else {
		stationW := runewidth.StringWidth(station)
		trackW := runewidth.StringWidth(m.trackTitle)
		if stationW+len(sep)+trackW <= avail {
			line = station + sep + m.trackTitle
		} else if remaining := avail - stationW - len(sep); remaining > 4 {
			line = station + sep + runewidth.Truncate(m.trackTitle, remaining, "…")
		} else {
			line = truncate(station, avail)
		}
	}

	if m.playing {
		return prefix + grad.Render(line)
	}
	return prefix + line
}

func (m model) renderWithSearchPopup() string {
	popup := m.buildSearchPopup()
	hints := renderHints([]string{"←→", "mode"}, []string{"↑↓", "select"}, []string{"type", "filter"}, []string{"enter", "search"}, []string{"esc", "back"})
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
	hints := renderHints([]string{"↑↓", "move"}, []string{"enter", "play"}, []string{"esc", "back"})
	if m.stationLoading {
		popup := renderPopupBox("stations", []string{"loading..."}, nil, -1, -1, 0, "", false, m.width, m.height)
		return overlay(hints, popup, m.width, m.height)
	}
	items := make([]string, len(m.stations))
	rightLabels := make([]string, len(m.stations))
	for i, s := range m.stations {
		items[i] = s.Name
		if s.Country != "" {
			rightLabels[i] = abbrevCountry(s.Country)
		}
	}
	if len(items) == 0 {
		items = []string{"(no stations)"}
		rightLabels = nil
	}
	popup := renderPopupBox("stations", items, rightLabels, m.stationCursor, len(m.stations), m.stationStart, "", false, m.width, m.height)
	return overlay(hints, popup, m.width, m.height)
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

func renderHints(pairs ...[]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = styleAccent.Render(p[0]) + styleDim.Render(":"+p[1])
	}
	return strings.Join(parts, styleDim.Render("  "))
}

var countryAbbr = map[string]string{
	"The United Kingdom of Great Britain and Northern Ireland": "UK",
	"United Kingdom":                "UK",
	"United States of America":      "US",
	"The United States Of America":  "US",
	"The United States of America":  "US",
	"Germany":                       "DE",
	"France":                        "FR",
	"Japan":                         "JP",
	"Canada":                        "CA",
	"Australia":                     "AU",
	"Brazil":                        "BR",
	"Netherlands":                   "NL",
	"Spain":                         "ES",
	"Italy":                         "IT",
	"Russia":                        "RU",
	"Poland":                        "PL",
	"Sweden":                        "SE",
	"Norway":                        "NO",
	"Denmark":                       "DK",
	"Finland":                       "FI",
	"Switzerland":                   "CH",
	"Austria":                       "AT",
	"Belgium":                       "BE",
	"Portugal":                      "PT",
	"Mexico":                        "MX",
	"Argentina":                     "AR",
	"China":                         "CN",
	"India":                         "IN",
	"South Korea":                   "KR",
	"Korea, Republic of":            "KR",
	"Turkey":                        "TR",
	"Ukraine":                       "UA",
	"Czech Republic":                "CZ",
	"Czechia":                       "CZ",
	"Hungary":                       "HU",
	"Romania":                       "RO",
	"Greece":                        "GR",
	"Bulgaria":                      "BG",
	"Croatia":                       "HR",
	"Slovakia":                      "SK",
	"Serbia":                        "RS",
	"Ireland":                       "IE",
	"New Zealand":                   "NZ",
	"South Africa":                  "ZA",
	"Israel":                        "IL",
	"Thailand":                      "TH",
	"Indonesia":                     "ID",
	"Malaysia":                      "MY",
	"Philippines":                   "PH",
	"Vietnam":                       "VN",
	"Pakistan":                      "PK",
	"Bangladesh":                    "BD",
	"Iran":                          "IR",
	"Egypt":                         "EG",
	"Nigeria":                       "NG",
	"Colombia":                      "CO",
	"Chile":                         "CL",
	"Peru":                          "PE",
	"Cuba":                          "CU",
	"Iceland":                       "IS",
	"Luxembourg":                    "LU",
	"Slovenia":                      "SI",
	"Lithuania":                     "LT",
	"Latvia":                        "LV",
	"Estonia":                       "EE",
	"Belarus":                       "BY",
	"Georgia":                       "GE",
	"Armenia":                       "AM",
	"Azerbaijan":                    "AZ",
	"Kazakhstan":                    "KZ",
	"Taiwan":                        "TW",
	"Taiwan, Province of China":     "TW",
	"Hong Kong":                     "HK",
	"Singapore":                     "SG",
	"Saudi Arabia":                  "SA",
	"United Arab Emirates":          "AE",
	"Morocco":                       "MA",
	"Algeria":                       "DZ",
	"Tunisia":                       "TN",
	"Kenya":                         "KE",
	"Ethiopia":                      "ET",
	"Ghana":                         "GH",
}

func abbrevCountry(name string) string {
	if abbr, ok := countryAbbr[name]; ok {
		return abbr
	}
	lower := strings.ToLower(name)
	for k, v := range countryAbbr {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return name
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
