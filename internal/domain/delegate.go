package domain

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

var previousItem *item
var previousIndex int

func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string
		var status string
		if i, ok := m.SelectedItem().(item); ok {
			title = i.Title()
			status = i.Status()
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.inspect):
				return m.NewStatusMessage(statusMessageStyle(title + " is " + status))

			case key.Matches(msg, keys.undo):
				if previousItem == nil {
					return m.NewStatusMessage(statusMessageStyle("Nothing to undo"))
				}
        cmd := m.InsertItem(previousIndex, previousItem)
        previousItemTitle := previousItem.title
				previousIndex = 0 
				previousItem = nil
        return tea.Batch(cmd, m.NewStatusMessage(statusMessageStyle("Restored " + previousItemTitle)))

			case key.Matches(msg, keys.remove):
        if len(m.Items()) == 0 {
          return m.NewStatusMessage(statusMessageStyle("Nothing to delete"))
        }
				previousIndex = m.Index() 
				i := m.SelectedItem()
        if it, ok := i.(item); ok {
          previousItem = &item{
            title: it.title,
            status: it.status,
            desc: it.desc,
          }
          m.RemoveItem(previousIndex)
        }
				return m.NewStatusMessage(statusMessageStyle("Deleted " + title))
			}
		}
		return nil
	}

	help := []key.Binding{keys.inspect, keys.remove, keys.undo}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

type delegateKeyMap struct {
	inspect key.Binding
	remove  key.Binding
	undo    key.Binding
}

// Additional short help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.inspect,
		d.remove,
		d.undo,
	}
}

// Additional full help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.inspect,
			d.remove,
			d.undo,
		},
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		inspect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "inspect"),
		),
		remove: key.NewBinding(
			key.WithKeys("x", "backspace"),
			key.WithHelp("x", "delete"),
		),
		undo: key.NewBinding(
			key.WithKeys("u", "ctrl+z"),
			key.WithHelp("u", "undo"),
		),
	}
}
