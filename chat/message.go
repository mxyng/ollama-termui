package chat

import (
	"iter"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss/v2"
)

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`

	roleStyle lipgloss.Style
}

var (
	defaultUserRoleStyle      = lipgloss.NewStyle().SetString("User: ").Bold(true)
	defaultAssistantRoleStyle = lipgloss.NewStyle().SetString("Assistant: ").Bold(true)
)

func (m chatMsg) Render(contentRenderer *glamour.TermRenderer) string {
	content, err := contentRenderer.Render(m.Content)
	if err != nil {
		content = m.Content
	}

	return m.roleStyle.Render() + content
}

type exchange struct {
	User, Assistant     chatMsg
	Complete, Cancelled bool
}

type Chat struct {
	sync.Mutex
	msgs []exchange

	width, height int
	history       History
	sb            strings.Builder
}

func (m *Chat) SetWidth(w int) {
	m.width = w
	m.build()
}

func (m *Chat) SetHeight(h int) {
	m.height = h
	m.build()
}

func (m *Chat) Height() int {
	return min(m.height, lipgloss.Height(m.sb.String()))
}

func (m *Chat) Add(s string) {
	m.Lock()
	defer m.Unlock()

	m.msgs = append(m.msgs, exchange{
		User: chatMsg{
			Role:      "user",
			Content:   s,
			roleStyle: defaultUserRoleStyle,
		},
		Assistant: chatMsg{
			Role:      "assistant",
			roleStyle: defaultAssistantRoleStyle,
		},
	})

	m.build()

	m.history.Push(s)
}

func (m *Chat) Update(s string) {
	m.Lock()
	defer m.Unlock()

	m.msgs[len(m.msgs)-1].Assistant.Content += s
	m.build()
}

func (m *Chat) Complete() {
	m.Lock()
	defer m.Unlock()

	if len(m.msgs) > 0 {
		m.msgs[len(m.msgs)-1].Complete = true
	}
}

func (m *Chat) Cancel() {
	m.Lock()
	defer m.Unlock()

	if len(m.msgs) > 0 && !m.msgs[len(m.msgs)-1].Complete {
		m.msgs[len(m.msgs)-1].Cancelled = true
	}
}

func (m *Chat) Reset() {
	m.Lock()
	defer m.Unlock()

	m.msgs = nil
	m.build()
}

func (m *Chat) build() {
	contentRenderer, _ := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.width-4),
	)

	m.sb.Reset()
	for _, msg := range m.msgs {
		m.sb.WriteString(msg.User.Render(contentRenderer))
		if msg.Assistant.Content != "" {
			m.sb.WriteString(msg.Assistant.Render(contentRenderer))
		}
	}
}

func (m *Chat) String() string {
	return m.sb.String()
}

func (m *Chat) Messages() iter.Seq[chatMsg] {
	m.Lock()
	defer m.Unlock()

	return func(yield func(chatMsg) bool) {
		for _, msg := range m.msgs {
			if msg.Cancelled {
				continue
			}

			if !yield(msg.User) {
				break
			} else if msg.Assistant.Content != "" {
				if !yield(msg.Assistant) {
					break
				}
			}
		}
	}
}
