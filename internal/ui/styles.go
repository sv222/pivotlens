package ui

import "charm.land/lipgloss/v2"

type Styles struct {
	Title    lipgloss.Style
	Header   lipgloss.Style
	Cursor   lipgloss.Style
	Null     lipgloss.Style
	Status   lipgloss.Style
	ErrorBar lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Title:    lipgloss.NewStyle().Bold(true),
		Header:   lipgloss.NewStyle().Bold(true).Underline(true),
		Cursor:   lipgloss.NewStyle().Reverse(true),
		Null:     lipgloss.NewStyle().Faint(true),
		Status:   lipgloss.NewStyle().Faint(true),
		ErrorBar: lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	}
}
