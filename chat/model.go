package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/cursor"
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/viewport"
	bbt "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/mxyng/ollama-termui/client"
)

type model struct {
	client client.Client
	name   string

	inProgress bool
	chat       Chat
	metrics    Metrics

	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	help Help
}

var _ bbt.Model = (*model)(nil)

func New(host, name string) *model {
	m := model{
		client:  client.New(host),
		name:    name,
		chat:    Chat{history: LoadFromFile(1000, true)},
		metrics: Metrics{round: 100 * time.Millisecond},

		viewport: viewport.New(),
		textarea: textarea.New(),
		spinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot)),

		help: Help{
			Model: help.New(),
			Help: key.NewBinding(
				key.WithKeys("/?"),
				key.WithHelp("/?", "help"),
			),
			ClearScreen: key.NewBinding(
				key.WithKeys("ctrl+l"),
				key.WithHelp("ctrl+l", "clear screen"),
			),
			PageUp: key.NewBinding(
				key.WithKeys("pgup"),
				key.WithHelp("pgup", "page up"),
			),
			PageDown: key.NewBinding(
				key.WithKeys("pgdown"),
				key.WithHelp("pgdown", "page down"),
			),
			Cancel: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("ctrl+c", "cancel"),
			),
			Suspend: key.NewBinding(
				key.WithKeys("ctrl+z"),
				key.WithHelp("ctrl+z", "suspend"),
			),
			Quit: key.NewBinding(
				key.WithKeys("ctrl+d", "/bye"),
				key.WithHelp("ctrl+d", "quit"),
			),
		},
	}

	m.viewport.Style = lipgloss.NewStyle().MarginBottom(1)
	m.viewport.KeyMap = viewport.KeyMap{
		PageUp:   m.help.PageUp,
		PageDown: m.help.PageDown,
	}

	m.textarea.Placeholder = "Type here..."
	m.textarea.ShowLineNumbers = false
	m.textarea.CharLimit = 0
	m.textarea.MaxWidth = 0
	m.textarea.MaxHeight = 0
	m.textarea.Styles.Focused.CursorLine = lipgloss.NewStyle()
	m.textarea.Cursor.SetMode(cursor.CursorStatic)
	m.textarea.SetPromptFunc(6, func(line int) string {
		return defaultUserRoleStyle.Render()
	})

	m.textarea.SetHeight(1)
	return &m
}

type chat struct{}

// Init implements tea.Model.
func (m *model) Init() (bbt.Model, bbt.Cmd) {
	return m, m.Send()
}

// Update implements tea.Model.
func (m *model) Update(msg bbt.Msg) (_ bbt.Model, cmd bbt.Cmd) {
	switch msg := msg.(type) {
	case bbt.WindowSizeMsg:
		m.chat.SetWidth(msg.Width)
		m.chat.SetHeight(msg.Height - m.textarea.Height() - m.help.Height())

		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(m.chat.Height())

		m.textarea.SetWidth(msg.Width)
	case bbt.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.inProgress {
				m.inProgress = false
				m.chat.Cancel()
			} else {
				m.textarea.Reset()
			}
		case "ctrl+d":
			return m, m.Bye()
		case "ctrl+l":
			if !m.inProgress {
				m.chat.Reset()
				m.metrics.Reset()
				m.viewport.SetContent(m.chat.String())
				m.viewport.SetHeight(m.chat.Height())
			}
			return m, nil
		case "ctrl+z":
			return m, bbt.Suspend
		case "up":
			if line := m.chat.history.PreviousLine(); line != "" {
				m.textarea.SetValue(line)
			}

			return m, nil
		case "down":
			line := m.chat.history.NextLine()
			m.textarea.SetValue(line)
			return m, nil
		case "enter":
			switch value := m.textarea.Value(); {
			case value == "":
				// noop
			case value == "/?":
				m.help.ShowAll = !m.help.ShowAll
			case value == "/bye":
				return m, m.Bye()
			case strings.HasPrefix(value, "/set "):
				switch parts := strings.SplitN(value, " ", 3); parts[1] {
				case "history":
					m.chat.history.saveOnPush = true
				case "nohistory":
					m.chat.history.saveOnPush = false
				}
			default:
				m.metrics.Reset()
				m.chat.Add(value)
				m.viewport.SetContent(m.chat.String())
				m.viewport.SetHeight(m.chat.Height())
				m.viewport.GotoBottom()
				cmd = m.Send()
			}

			m.textarea.Reset()
			return m, cmd
		}
	case spinner.TickMsg:
		if m.inProgress {
			m.spinner, cmd = m.spinner.Update(msg)
		}

		return m, cmd
	case client.ErrMsg:
		return m, bbt.Sequence(
			bbt.Printf("Error: %s", msg),
			m.Bye(),
		)
	case *client.Response[chat]:
		if m.inProgress && msg.Scan() {
			var r struct {
				Message   chatMsg   `json:"message"`
				CreatedAt time.Time `json:"created_at"`
			}

			if err := json.Unmarshal(msg.Bytes(), &r); err != nil {
				return m, func() bbt.Msg {
					return err
				}
			}

			if r.Message.Content != "" {
				m.chat.Update(r.Message.Content)
				m.viewport.SetContent(m.chat.String())
				m.viewport.SetHeight(m.chat.Height())
				m.viewport.GotoBottom()
				m.metrics.Add(r.CreatedAt)
			}

			return m, func() bbt.Msg {
				return msg
			}
		}

		m.inProgress = false

		_ = msg.Close()
		m.chat.Complete()

		return m, m.textarea.Focus()
	}

	var cmds []bbt.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	m.help, cmd = m.help.Update(msg)
	cmds = append(cmds, cmd)
	return m, bbt.Batch(cmds...)
}

// View implements tea.Model.
func (m *model) View() string {
	views := make([]string, 0, 3)
	if len(m.chat.msgs) > 0 {
		views = slices.Insert(views, 0, m.viewport.View())
	}

	if m.inProgress {
		// truncate the input area and add the spinner
		maxWidth := m.viewport.Width() - lipgloss.Width(m.spinner.View())
		views = append(views, lipgloss.NewStyle().MaxWidth(maxWidth).Render(m.textarea.View())+m.spinner.View())
	} else {
		views = append(views, m.textarea.View())
	}

	footerLeft := "Top"
	if scroll := m.viewport.ScrollPercent(); scroll >= 1.0 {
		footerLeft = "Bot"
	} else if scroll > 0.0 {
		footerLeft = fmt.Sprintf("%.0f%%", scroll*100)
	}

	if rate := m.metrics.Rate(); rate > 0 {
		footerLeft = footerLeft + fmt.Sprintf(" %.1f tokens/s", rate)
	}

	footer := m.help.View()
	if !m.help.ShowAll {
		footerLeft = m.help.Styles.ShortDesc.Render(footerLeft)

		footerRight := footer

		footerCenter := m.name
		footerCenter = lipgloss.PlaceHorizontal(
			m.viewport.Width()-lipgloss.Width(footerLeft)-lipgloss.Width(footerRight),
			lipgloss.Center,
			m.help.Styles.ShortKey.Render(footerCenter),
		)

		footer = lipgloss.JoinHorizontal(
			lipgloss.Top,
			footerLeft,
			footerCenter,
			footerRight,
		)
	}

	views = append(views, footer)
	return lipgloss.JoinVertical(lipgloss.Right, views...)
}

func (m *model) Send() bbt.Cmd {
	m.inProgress = true
	m.textarea.Blur()
	return bbt.Batch(
		m.spinner.Tick,
		client.Send[chat](m.client, http.MethodPost, "/api/chat", map[string]any{
			"model":    m.name,
			"messages": slices.Collect(m.chat.Messages()),
		}),
	)
}

func (m *model) Bye() bbt.Cmd {
	m.textarea.Blur()
	return bbt.Quit
}
