package domain

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)
var availableDomainStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
var unavailableDomainStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
var erroredDomainStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))

type item struct {
	title  string
	desc   string
	status string
}

var ListItems []list.Item

var STATUS_AVAILABLE = "available"
var STATUS_UNAVAILABLE = "unavailable"
var STATUS_ERRORED = "errored"

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
)

func (i item) Status() string { return i.status }
func (i item) Domain() string { return i.title }

func (i item) Title() string {
	if i.status == STATUS_ERRORED {
		return "errored: " + i.title
	}
	if i.status == STATUS_AVAILABLE {
		return "✔️ " + i.title
	}
	return "❌ " + i.title
}
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + i.status }

type listKeyMap struct {
	toggleHelpMenu key.Binding
	exportList     key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
		exportList: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "export list"),
		),
	}
}

type model struct {
	list         list.Model
	quitting     bool
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil

		case key.Matches(msg, m.keys.exportList):

			// we are going to export these domains to stdout for now after exiting the program
			m.quitting = true

			return m, tea.Quit
			// case key.Matches(msg, m.keys.insertItem):
			// m.delegateKeys.remove.SetEnabled(true)
			// newItem := m.itemGenerator.next()
			// insCmd := m.list.InsertItem(0, newItem)
			// statusCmd := m.list.NewStatusMessage(statusMessageStyle("Added " + newItem.Title()))
			// return m, tea.Batch(insCmd, statusCmd)
		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func RunList(domains map[string]bool, errs map[string]error) {

	var listKeys = newListKeyMap()
	var delegateKeys = newDelegateKeyMap()

	ListItems = make([]list.Item, 0)
	for domain, available := range domains {
		var status string
		if available {
			status = STATUS_AVAILABLE
		} else {
			status = STATUS_UNAVAILABLE
		}
		ListItems = append(ListItems, item{title: domain, desc: fmt.Sprintf("Available: %t", available), status: status})
	}
	for domain, err := range errs {
		ListItems = append(ListItems, item{title: domain, desc: fmt.Sprintf("Errored: %s", err), status: "errored"})
	}

	d := newItemDelegate(delegateKeys)
	d.ShowDescription = false

	ls := list.New(ListItems, d, 0, 0)
	ls.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleHelpMenu,
		}
	}
	ls.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.exportList,
		}
	}

	m := model{
		list:         ls,
		keys:         listKeys,
		delegateKeys: delegateKeys,
	}
	m.list.Title = "Domain Results"
	m.list.InfiniteScrolling = true
	p := tea.NewProgram(m, tea.WithAltScreen())
	runtimeModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	var remainingDomains []list.Item
	if runtimeModel.(model).quitting {
		remainingDomains = m.list.Items()
	}
	for _, do := range remainingDomains {
		if i, ok := do.(item); ok {
			if i.status == STATUS_AVAILABLE {
				fmt.Println(Available(i.title))
			}
			if i.status == STATUS_UNAVAILABLE {
				fmt.Println(NotAvailable(i.title))
			}
			if i.status == STATUS_ERRORED {
				fmt.Println(Errored(i.title, fmt.Errorf("Error: Failed to check domain")))
			}
		}

	}
}
