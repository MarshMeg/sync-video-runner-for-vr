package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MarshMeg/sync-video-runner-for-vr/internal/server"
)

type tickMsg time.Time
type actionResultMsg string

const startTimeout = 3 * time.Second

type model struct {
	store            *server.Store
	starter          *server.Starter
	phones           []server.Phone
	status           string
	actionInProgress bool
	lastUpdateTime   time.Time
}

func (m model) Init() tea.Cmd {
	return m.tickCmd()
}

func (m model) tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			if m.actionInProgress {
				m.status = "Ctrl+R: action already running."
				return m, nil
			}

			phonesSnapshot := m.store.SnapshotPhones()
			m.actionInProgress = true
			m.status = fmt.Sprintf("Ctrl+R: sending start command to %d devices...", len(phonesSnapshot))

			return m, m.sendStartRequestsCmd(phonesSnapshot)
		case "ctrl+e":
			cleared := m.store.ClearPhones()
			m.status = fmt.Sprintf("Ctrl+E: cleared %d devices.", cleared)
			m.phones = m.store.SnapshotPhones()
			return m, nil
		}

	case tickMsg:
		m.phones = m.store.SnapshotPhones()
		m.lastUpdateTime = time.Now()
		return m, m.tickCmd()

	case actionResultMsg:
		m.actionInProgress = false
		m.status = string(msg)
		return m, nil
	}

	return m, nil
}

func (m model) sendStartRequestsCmd(phones []server.Phone) tea.Cmd {
	return func() tea.Msg {
		results := m.starter.Start(context.Background(), phones)
		return actionResultMsg(startSummary(results))
	}
}

func startSummary(results []server.StartResult) string {
	if len(results) == 0 {
		return "Ctrl+R: no connected devices."
	}

	failCount := 0
	firstErr := ""

	for _, result := range results {
		if result.Err == nil {
			continue
		}

		failCount++
		if firstErr == "" {
			firstErr = fmt.Sprintf("%02d (%s): %v", result.Phone.ID, result.Phone.IP, result.Err)
		}
	}

	okCount := len(results) - failCount

	if failCount > 0 {
		return fmt.Sprintf("Ctrl+R: OK %d/%d, failed %d. First error: %s", okCount, len(results), failCount, firstErr)
	}

	return fmt.Sprintf("Ctrl+R: request sent to %d devices.", okCount)
}

func (m model) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Width(50)

	statusBarStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255"))

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	deviceRowStyle := lipgloss.NewStyle().
		Padding(0, 1)

	now := time.Now().Format("2006-01-02 15:04:05")

	view := titleStyle.Render("Sync Video Runner Device Panel") + "\n"
	view += fmt.Sprintf("Updated: %s\n", now)
	view += fmt.Sprintf("Devices: %d\n\n", len(m.phones))

	if len(m.phones) == 0 {
		view += "No connected devices yet.\n"
		view += "Waiting for /update-ip requests...\n\n"
	} else {
		view += headerStyle.Render("ID  ADDRESS            PORT") + "\n"
		view += headerStyle.Render("--  -----------------  ----") + "\n"
		for _, phone := range m.phones {
			view += deviceRowStyle.Render(
				fmt.Sprintf("%02d  %-17s  %4d\n", phone.ID, phone.IP, phone.Port),
			)
		}
		view += "\n"
	}

	view += "\n" + statusBarStyle.Render(
		"Controls:  Ctrl+R: Send start command to all phones  |  Ctrl+E: Clear devices list  |  Ctrl+C: Exit",
	) + "\n"

	view += fmt.Sprintf("Status: %s\n", m.status)

	return view
}

func Run(store *server.Store) {
	m := model{
		store:   store,
		starter: server.NewStarter(startTimeout),
		phones:  store.SnapshotPhones(),
		status:  "Waiting for input.",
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
	}
}
