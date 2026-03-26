package ui

import (
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
)

func NewSpinner(s spinner.Spinner) spinner.Model {
	return spinner.New(
		spinner.WithSpinner(s),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(Blue)),
	)
}

// PhasedMessages cycles through messages at a given interval.
type PhasedMessages struct {
	Messages []string
	Interval time.Duration
	current  int
	elapsed  time.Duration
}

func NewPhasedMessages(msgs []string, interval time.Duration) *PhasedMessages {
	return &PhasedMessages{
		Messages: msgs,
		Interval: interval,
	}
}

func (p *PhasedMessages) Current() string {
	if len(p.Messages) == 0 {
		return ""
	}
	return p.Messages[p.current]
}

func (p *PhasedMessages) Tick(d time.Duration) {
	p.elapsed += d
	if p.elapsed >= p.Interval && p.current < len(p.Messages)-1 {
		p.current++
		p.elapsed = 0
	}
}
