package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	renderstyle "github.com/renderinc/render-cli/pkg/style"
)

var stackHeaderStyle = lipgloss.NewStyle().MarginTop(1).MarginBottom(1)
var stackInfoStyle = lipgloss.NewStyle().Foreground(renderstyle.ColorInfo).Bold(true)

type StackModel struct {
	loadingSpinner *spinner.Model
	stack          []ModelWithCmd

	width  int
	height int
}

type ModelWithCmd struct {
	Model tea.Model
	Cmd   string
	Breadcrumb string
}

type StackSizeMsg struct {
	Width  int
	Height int
	Top    int
}

// ErrorMsg quits the program after displaying an error message
type ErrorMsg struct {
	Err error
}

// DoneMsg quits the program after displaying a message
type DoneMsg struct {
	Message string
}

// ClearScreenMsg is a message that clears the screen before rendering the next message
type ClearScreenMsg struct {
	NextMsg tea.Msg
}

func NewStack() *StackModel {
	return &StackModel{}
}

func newSpinner() *spinner.Model {
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &spin
}

func (m *StackModel) Push(model ModelWithCmd) tea.Cmd {
	m.stack = append(m.stack, model)
	return tea.Batch(model.Model.Init(), func() tea.Msg { return m.StackSizeMsg() })
}

func (m *StackModel) Pop() {
	if len(m.stack) > 0 {
		m.stack = m.stack[:len(m.stack)-1]
	}
}

func (m *StackModel) Init() tea.Cmd {
	var cmd tea.Cmd
	for _, model := range m.stack {
		cmd = tea.Batch(cmd, model.Model.Init())
	}
	return tea.Batch(cmd)
}

func (m *StackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.stack) == 0 {
		return m, tea.Quit
	}

	subMsg := msg

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlD:
			m.Pop()
			if len(m.stack) == 0 {
				return m, tea.Quit
			}
			return m, m.Init()
		case tea.KeyCtrlS:
			// copy command to clipboard
			if len(m.stack) > 0 {
				err := clipboard.WriteAll(m.stack[len(m.stack)-1].Cmd)
				if err != nil {
					m.Push(ModelWithCmd{Model: NewErrorModel("Failed to copy command to clipboard")})
				}
			}
		}
	case ClearScreenMsg:
		m.stack = m.stack[:0]
		return m, func() tea.Msg {
			return msg.NextMsg
		}
	case LoadingDataMsg:
		m.loadingSpinner = newSpinner()
		return m, tea.Batch(m.loadingSpinner.Tick, tea.Cmd(msg))
	case DoneLoadingDataMsg:
		m.loadingSpinner = nil
		return m, nil
	case ErrorMsg:
		m.Push(ModelWithCmd{Model: NewErrorModel(msg.Err.Error())})
		return m, nil
	case spinner.TickMsg:
		if m.loadingSpinner != nil {
			spin, cmd := m.loadingSpinner.Update(msg)
			m.loadingSpinner = &spin
			return m, cmd
		}
	case DoneMsg:
		m.Pop()
		if len(m.stack) == 0 {
			return m, tea.Sequence(
				tea.Println(msg.Message),
				tea.Quit,
			)
		}

		return m, m.Init()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update the message for subcomponents to exclude the header
		subMsg = m.StackSizeMsg()
	}

	var cmd tea.Cmd
	if len(m.stack) > 0 {
		m.stack[len(m.stack)-1].Model, cmd = m.stack[len(m.stack)-1].Model.Update(subMsg)
	}

	return m, cmd
}

func (m *StackModel) StackSizeMsg() StackSizeMsg {
	return StackSizeMsg{
		Width:  m.width,
		Height: m.height - lipgloss.Height(m.header()) - lipgloss.Height(m.footer()),
		Top:    lipgloss.Height(m.header()),
	}
}

func (m *StackModel) View() string {
	if m.loadingSpinner != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fmt.Sprintf("%s Loading...", m.loadingSpinner.View()))
	}

	if len(m.stack) == 0 {
		return ""
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.header(), m.stack[len(m.stack)-1].Model.View(), m.footer())
}

func (m *StackModel) header() string {
	var breadCrumbs []string
	for _, model := range m.stack {
		if model.Breadcrumb != "" {
			breadCrumbs = append(breadCrumbs, stackInfoStyle.Render(model.Breadcrumb))
		}
	}

	return stackHeaderStyle.Render(strings.Join(breadCrumbs, " > "))
}

func (m *StackModel) footer() string {
	quitCommand := fmt.Sprintf("%s Quit", renderstyle.CommandKey.Render("[Ctrl+C]"))
	prevCommand := fmt.Sprintf("%s Previous command", renderstyle.CommandKey.Render("[Ctrl+D]"))
	saveToClipboard := fmt.Sprintf("%s Copy command to clipboard", renderstyle.CommandKey.Render("[Ctrl+S]"))

	var commands []string
	commands = append(commands, quitCommand)

	if len(m.stack) > 1 {
		commands = append(commands, prevCommand)
	}

	if m.stack[len(m.stack)-1].Cmd != "" {
		commands = append(commands, saveToClipboard)
	}

	return renderstyle.CommandTitle.Render("Navigation: ") + strings.Join(commands, " ")
}
