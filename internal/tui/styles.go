package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	palette      palette
	app          lipgloss.Style
	header       lipgloss.Style
	panel        lipgloss.Style
	panelFocused lipgloss.Style
	title        lipgloss.Style
	label        lipgloss.Style
	value        lipgloss.Style
	muted        lipgloss.Style
	info         lipgloss.Style
	warn         lipgloss.Style
	error        lipgloss.Style
	ok           lipgloss.Style
	tab          lipgloss.Style
	tabActive    lipgloss.Style
}

type palette struct {
	title string
	label string
	value string
	muted string
	ok    string
	info  string
	warn  string
	error string
}

func newStyles() styles {
	border := lipgloss.AdaptiveColor{Light: "#B7C2D0", Dark: "#2E3A46"}
	panelBg := lipgloss.AdaptiveColor{Light: "#F8FAFC", Dark: "#10161D"}
	headerBg := lipgloss.AdaptiveColor{Light: "#E8EEF5", Dark: "#14202B"}
	active := lipgloss.AdaptiveColor{Light: "#0F766E", Dark: "#63C7B2"}
	info := lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#72A7FF"}
	warn := lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#F4A261"}
	err := lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FF7A7A"}
	fg := lipgloss.AdaptiveColor{Light: "#0F172A", Dark: "#E6EDF3"}
	muted := lipgloss.AdaptiveColor{Light: "#475569", Dark: "#8A97A5"}

	basePanel := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(border).
		Background(panelBg).
		Padding(0, 1)

	p := palette{
		title: fg.Light,
		label: muted.Light,
		value: fg.Light,
		muted: muted.Light,
		ok:    active.Light,
		info:  info.Light,
		warn:  warn.Light,
		error: err.Light,
	}
	return styles{
		palette: p,
		app:     lipgloss.NewStyle().Padding(0, 1),
		header: basePanel.
			Background(headerBg).
			Padding(0, 1),
		panel: basePanel,
		panelFocused: basePanel.
			BorderForeground(active),
		title: lipgloss.NewStyle().Foreground(fg).Bold(true),
		label: lipgloss.NewStyle().Foreground(muted),
		value: lipgloss.NewStyle().Foreground(fg).Bold(true),
		muted: lipgloss.NewStyle().Foreground(muted),
		info:  lipgloss.NewStyle().Foreground(info).Bold(true),
		warn:  lipgloss.NewStyle().Foreground(warn).Bold(true),
		error: lipgloss.NewStyle().Foreground(err).Bold(true),
		ok:    lipgloss.NewStyle().Foreground(active).Bold(true),
		tab: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),
		tabActive: lipgloss.NewStyle().
			Foreground(fg).
			Background(headerBg).
			Bold(true).
			Padding(0, 1),
	}
}
