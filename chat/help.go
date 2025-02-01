package chat

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	bbt "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type Help struct {
	help.Model
	Help        key.Binding
	ClearScreen key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Cancel      key.Binding
	Suspend     key.Binding
	Quit        key.Binding
}

func (m Help) Update(msg bbt.Msg) (_ Help, cmd bbt.Cmd) {
	switch msg := msg.(type) {
	case bbt.WindowSizeMsg:
		m.Width = msg.Width
	}

	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

func (m Help) View() string {
	return m.Model.View(m)
}

func (m Help) ShortHelp() []key.Binding {
	return []key.Binding{m.Help}
}

func (m Help) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.ClearScreen},
		{m.PageUp, m.PageDown},
		{m.Help, m.Cancel, m.Quit},
	}
}

func (m Help) Height() int {
	if m.ShowAll {
		return lipgloss.Height(m.FullHelpView(m.FullHelp()))
	}

	return lipgloss.Height(m.ShortHelpView(m.ShortHelp()))
}
