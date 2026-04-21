package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenPlayer screen = iota
	screenSearch
	screenStation
)

type searchMode int

const (
	modeGenre    searchMode = iota
	modeCountry
	modeLanguage
	modeName
)

// --- messages ---

type mpvExitedMsg struct{}
type playStartedMsg struct{ Station Station }
type connectedMsg struct{}
type vizTickMsg struct{}

type stationsLoadedMsg struct {
	Stations []Station
	Err      error
}

type listLoadedMsg struct {
	Mode  searchMode
	Items []string
	Err   error
}

type statusMsg struct{ Text string }
type tickMsg time.Time
type stationSelectedExternalMsg struct{ Station Station }
type trackTitleMsg struct{ Title string }

// --- mpv management ---

var mpvProc *exec.Cmd
var mpvGeneration int64

func stopStation() {
	atomic.AddInt64(&mpvGeneration, 1) // invalidate any running watcher
	if mpvProc != nil {
		_ = mpvProc.Process.Kill()
		mpvProc = nil
	}
	killSavedMpv()
}

// --- model ---

type model struct {
	width, height int
	ready         bool
	screen        screen
	selectMode    bool

	// search popup
	searchMode     searchMode
	searchFilter   string
	searchCursor   int
	searchStart    int
	searchLoading  bool
	countries      []string
	languages      []string
	filteredItems  []string

	// station popup
	stations       []Station
	stationCursor  int
	stationStart   int
	stationLoading bool

	// player
	current    *Station
	playing    bool
	connecting bool
	trackTitle string

	// status
	statusMsg    string
	statusExpiry time.Time
}

func newModel(selectMode bool) model {
	m := model{
		screen:        screenPlayer,
		selectMode:    selectMode,
		searchMode:    modeGenre,
		filteredItems: Genres,
	}
	if selectMode {
		m.screen = screenSearch
	} else {
		if s, err := LoadCurrentStation(); err == nil {
			if isMpvRunning() {
				m.current = &s
				m.playing = true
			}
		}
	}
	return m
}

func (m model) Init() tea.Cmd {
	if m.selectMode {
		return nil
	}
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// --- commands ---

func fetchList(mode searchMode) tea.Cmd {
	return func() tea.Msg {
		var cacheKey string
		switch mode {
		case modeCountry:
			cacheKey = "countries"
		case modeLanguage:
			cacheKey = "languages"
		default:
			return nil
		}

		if cached, err := LoadListCache(cacheKey); err == nil {
			return listLoadedMsg{Mode: mode, Items: cached}
		}

		var items []string
		var err error
		switch mode {
		case modeCountry:
			items, err = FetchCountries()
		case modeLanguage:
			items, err = FetchLanguages()
		}
		if err == nil {
			_ = SaveListCache(cacheKey, items)
		}
		return listLoadedMsg{Mode: mode, Items: items, Err: err}
	}
}

func fetchStationsCmd(mode searchMode, query string) tea.Cmd {
	return func() tea.Msg {
		var stations []Station
		var err error
		switch mode {
		case modeGenre:
			stations, err = FetchStationsByGenre(query)
		case modeCountry:
			stations, err = FetchStationsByCountry(query)
		case modeLanguage:
			stations, err = FetchStationsByLanguage(query)
		case modeName:
			stations, err = FetchStationsByName(query)
		}
		return stationsLoadedMsg{Stations: stations, Err: err}
	}
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

func vizTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return vizTickMsg{}
	})
}

func connectTimer() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return connectedMsg{}
	})
}

func trackPoll() tea.Cmd {
	return func() tea.Msg {
		title, _ := fetchIcyTitle()
		return trackTitleMsg{Title: title}
	}
}

func trackPollDelayed() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		title, _ := fetchIcyTitle()
		return trackTitleMsg{Title: title}
	})
}

func openSelectPopup() tea.Cmd {
	return func() tea.Msg {
		exe, err := os.Executable()
		if err != nil {
			exe = os.Args[0]
		}
		var cmd *exec.Cmd
		switch {
		case os.Getenv("TMUX") != "":
			cmd = exec.Command("tmux", "display-popup", "-E", "-w", "66", "-h", "22", exe, "--select")
		case os.Getenv("ZELLIJ") != "":
			cmd = exec.Command("zellij", "run", "--floating", "--close-on-exit", "--width", "66", "--height", "22", "--", exe, "--select")
		}
		if cmd != nil {
			_ = cmd.Run()
		}
		if s, err := LoadCurrentStation(); err == nil {
			return stationSelectedExternalMsg{Station: s}
		}
		return nil
	}
}

// --- update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tickMsg:
		if s, err := LoadCurrentStation(); err == nil {
			if m.current == nil || m.current.URL != s.URL {
				m.current = &s
				m.playing = true
			}
		}
		return m, tick()

	case playStartedMsg:
		s := msg.Station
		m.current = &s
		m.playing = true
		m.connecting = true
		m.trackTitle = ""
		return m, tea.Batch(vizTick(), connectTimer())

	case vizTickMsg:
		grad.Tick(m.connecting, m.playing)
		if m.connecting || m.playing {
			return m, vizTick()
		}

	case connectedMsg:
		m.connecting = false
		return m, trackPoll()

	case trackTitleMsg:
		m.trackTitle = msg.Title
		if m.playing {
			return m, trackPollDelayed()
		}

	case mpvExitedMsg:
		m.connecting = false
		m.playing = false
		m.trackTitle = ""
		m.setStatus("stream ended or unreachable")

	case stationSelectedExternalMsg:
		m.current = &msg.Station
		m.playing = true
		m.connecting = true
		m.trackTitle = ""
		return m, tea.Batch(vizTick(), connectTimer())

	case listLoadedMsg:
		m.searchLoading = false
		if msg.Err != nil {
			m.setStatus("error: " + msg.Err.Error())
		} else {
			switch msg.Mode {
			case modeCountry:
				m.countries = msg.Items
			case modeLanguage:
				m.languages = msg.Items
			}
			m.filteredItems = filterItems(msg.Items, m.searchFilter)
			m.searchCursor = 0
		}

	case stationsLoadedMsg:
		m.stationLoading = false
		if msg.Err != nil {
			m.setStatus("error: " + msg.Err.Error())
			m.screen = screenSearch
		} else if len(msg.Stations) == 0 {
			m.setStatus("no stations found")
			m.screen = screenSearch
		} else {
			m.stations = msg.Stations
			m.stationCursor = 0
			m.screen = screenStation
		}

	case statusMsg:
		if msg.Text != "" {
			m.setStatus(msg.Text)
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *model) setStatus(s string) {
	m.statusMsg = s
	m.statusExpiry = time.Now().Add(4 * time.Second)
}

func (m model) currentStatus() string {
	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		return m.statusMsg
	}
	return ""
}

// --- key handlers ---

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenPlayer:
		return m.handlePlayerKey(msg)
	case screenSearch:
		return m.handleSearchKey(msg)
	case screenStation:
		return m.handleStationKey(msg)
	}
	return m, nil
}

func (m model) handlePlayerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		stopStation()
		return m, tea.Quit
	case " ":
		if os.Getenv("TMUX") != "" || os.Getenv("ZELLIJ") != "" {
			return m, openSelectPopup()
		}
		return m, m.openSearch()
	case "k", "up":
		return m, adjustVolume(+5)
	case "j", "down":
		return m, adjustVolume(-5)
	}
	return m, nil
}

func adjustVolume(delta int) tea.Cmd {
	return func() tea.Msg {
		sign := "+"
		if delta < 0 {
			sign = "-"
			delta = -delta
		}
		arg := fmt.Sprintf("%s%d%%", sign, delta)
		if err := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", arg).Run(); err != nil {
			return statusMsg{Text: "volume error: " + err.Error()}
		}
		out, err := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output()
		if err != nil {
			return statusMsg{Text: "vol " + arg}
		}
		vol := parseVolume(string(out))
		return statusMsg{Text: "vol " + vol}
	}
}

func parseVolume(out string) string {
	// "Volume: front-left: 65536 / 100% / ..." → "100%"
	for _, field := range strings.Fields(out) {
		if strings.HasSuffix(field, "%") {
			return field
		}
	}
	return ""
}

func (m *model) openSearch() tea.Cmd {
	m.screen = screenSearch
	m.searchFilter = ""
	m.searchCursor = 0
	return m.applySearchMode(m.searchMode)
}

func (m *model) applySearchMode(mode searchMode) tea.Cmd {
	m.searchMode = mode
	m.searchFilter = ""
	m.searchCursor = 0
	m.searchStart = 0

	switch mode {
	case modeGenre:
		m.filteredItems = Genres
		m.searchLoading = false
		return nil
	case modeCountry:
		if len(m.countries) > 0 {
			m.filteredItems = m.countries
			m.searchLoading = false
			return nil
		}
		m.searchLoading = true
		m.filteredItems = nil
		return fetchList(modeCountry)
	case modeLanguage:
		if len(m.languages) > 0 {
			m.filteredItems = m.languages
			m.searchLoading = false
			return nil
		}
		m.searchLoading = true
		m.filteredItems = nil
		return fetchList(modeLanguage)
	case modeName:
		m.filteredItems = nil
		m.searchLoading = false
		return nil
	}
	return nil
}

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	modeOrder := []searchMode{modeGenre, modeCountry, modeLanguage, modeName}

	// mode switching
	switch msg.String() {
	case "1":
		return m, m.applySearchMode(modeGenre)
	case "2":
		return m, m.applySearchMode(modeCountry)
	case "3":
		return m, m.applySearchMode(modeLanguage)
	case "/":
		return m, m.applySearchMode(modeName)
	case "right":
		for i, mo := range modeOrder {
			if mo == m.searchMode {
				return m, m.applySearchMode(modeOrder[(i+1)%len(modeOrder)])
			}
		}
	case "left":
		for i, mo := range modeOrder {
			if mo == m.searchMode {
				return m, m.applySearchMode(modeOrder[(i+len(modeOrder)-1)%len(modeOrder)])
			}
		}
	case "esc", "q":
		if m.selectMode {
			return m, tea.Quit
		}
		m.screen = screenPlayer
		return m, nil
	}

	if m.searchLoading {
		return m, nil
	}

	// name search mode: Enter triggers API call
	if m.searchMode == modeName {
		switch msg.String() {
		case "enter":
			if m.searchFilter == "" {
				break
			}
			m.stationLoading = true
			m.screen = screenStation
			m.stations = nil
			return m, fetchStationsCmd(modeName, m.searchFilter)
		case "backspace":
			if len(m.searchFilter) > 0 {
				_, size := utf8.DecodeLastRuneInString(m.searchFilter)
				m.searchFilter = m.searchFilter[:len(m.searchFilter)-size]
			}
		default:
			if len(msg.Runes) > 0 {
				m.searchFilter += string(msg.Runes)
			}
		}
		return m, nil
	}

	// list modes: arrow keys navigate, all printable chars go to filter
	switch msg.String() {
	case "enter":
		if len(m.filteredItems) == 0 {
			break
		}
		selected := m.filteredItems[m.searchCursor]
		m.stationLoading = true
		m.screen = screenStation
		m.stations = nil
		return m, fetchStationsCmd(m.searchMode, selected)
	case "down":
		if m.searchCursor < len(m.filteredItems)-1 {
			m.searchCursor++
			maxRows := listMaxRows(m.height)
			if m.searchCursor >= m.searchStart+maxRows {
				m.searchStart = m.searchCursor - maxRows + 1
			}
		}
	case "up":
		if m.searchCursor > 0 {
			m.searchCursor--
			if m.searchCursor < m.searchStart {
				m.searchStart = m.searchCursor
			}
		}
	case "backspace":
		if len(m.searchFilter) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.searchFilter)
			m.searchFilter = m.searchFilter[:len(m.searchFilter)-size]
			m.filteredItems = filterItems(m.currentList(), m.searchFilter)
			m.searchCursor = 0
			m.searchStart = 0
		}
	default:
		if len(msg.Runes) > 0 {
			m.searchFilter += string(msg.Runes)
			m.filteredItems = filterItems(m.currentList(), m.searchFilter)
			m.searchCursor = 0
			m.searchStart = 0
		}
	}
	return m, nil
}

func (m model) currentList() []string {
	switch m.searchMode {
	case modeGenre:
		return Genres
	case modeCountry:
		return m.countries
	case modeLanguage:
		return m.languages
	}
	return nil
}

func (m model) handleStationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.stationLoading {
		if msg.String() == "esc" || msg.String() == "q" {
			if m.selectMode {
				return m, tea.Quit
			}
			m.screen = screenSearch
			m.stationLoading = false
		}
		return m, nil
	}
	switch msg.String() {
	case "esc", "backspace":
		m.screen = screenSearch
	case "q", "ctrl+c":
		if m.selectMode {
			return m, tea.Quit
		}
		stopStation()
		return m, tea.Quit
	case "enter":
		if len(m.stations) == 0 {
			break
		}
		s := m.stations[m.stationCursor]
		if m.selectMode {
			return m, tea.Batch(playCmd(s), tea.Quit)
		}
		m.screen = screenPlayer
		return m, playCmd(s)
	case "j", "l", "down":
		if m.stationCursor < len(m.stations)-1 {
			m.stationCursor++
			maxRows := listMaxRows(m.height)
			if m.stationCursor >= m.stationStart+maxRows {
				m.stationStart = m.stationCursor - maxRows + 1
			}
		}
	case "k", "h", "up":
		if m.stationCursor > 0 {
			m.stationCursor--
			if m.stationCursor < m.stationStart {
				m.stationStart = m.stationCursor
			}
		}
	}
	return m, nil
}

func listMaxRows(height int) int {
	rows := height - 10
	if rows < 3 {
		rows = 3
	}
	if rows > 15 {
		rows = 15
	}
	return rows
}

func filterItems(items []string, filter string) []string {
	if filter == "" {
		return items
	}
	lower := strings.ToLower(filter)
	var result []string
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), lower) {
			result = append(result, item)
		}
	}
	return result
}
