package pull

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/v2/progress"
	bbt "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mxyng/ollama-termui/client"
)

type model struct {
	client client.Client
	name   string

	pulls []*pull
}

var _ bbt.Model = (*model)(nil)

func New(host, name string) *model {
	return &model{
		client: client.New(host),
		name:   name,
	}
}

type pull struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`

	sync.Once
	progress.Model
}

func (p *pull) Render() string {
	if p.Digest == "" {
		return p.Status
	}

	p.Do(func() {
		p.Model = progress.New()
	})

	return p.Status + " " + p.ViewAs(float64(p.Completed)/float64(p.Total))
}

// Init implements tea.Model.
func (m *model) Init() (bbt.Model, bbt.Cmd) {
	return m, m.Send()
}

// Update implements tea.Model.
func (m *model) Update(msg bbt.Msg) (_ bbt.Model, cmd bbt.Cmd) {
	switch msg := msg.(type) {
	case bbt.WindowSizeMsg:
		for _, p := range m.pulls {
			p.SetWidth(msg.Width)
		}
	case bbt.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, bbt.Quit
		}
	case client.ErrMsg:
		return m, bbt.Sequence(
			bbt.Printf("Error: %s", msg),
			bbt.Quit,
		)
	case *client.Response[pull]:
		if msg.Scan() {
			var p pull
			if err := json.Unmarshal(msg.Bytes(), &p); err != nil {
				return m, func() bbt.Msg {
					return err
				}
			}

			if i := slices.IndexFunc(m.pulls, func(pp *pull) bool {
				return pp.Digest == p.Digest
			}); i > 0 {
				m.pulls[i].Completed = p.Completed
			} else {
				m.pulls = append(m.pulls, &p)
			}

			return m, func() bbt.Msg {
				return msg
			}
		}

		_ = msg.Close()
		return m, bbt.Tick(200*time.Millisecond, func(time.Time) bbt.Msg {
			fmt.Println()
			return bbt.QuitMsg{}
		})
	}

	return m, nil
}

// View implements tea.Model.
func (m *model) View() string {
	var views []string
	for _, p := range m.pulls {
		views = append(views, p.Render())
	}

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

func (m *model) Send() bbt.Cmd {
	return client.Send[pull](m.client, http.MethodPost, "/api/pull", map[string]any{
		"model": m.name,
	})
}
