package pretty

import (
	"fmt"

	"github.com/kaatinga/robot/internal/color"
)

type ScopePrinter string

func NewScopePrinter(prefix string) ScopePrinter {
	return ScopePrinter(prefix)
}

func (s *ScopePrinter) AddPrefix(prefix string) {
	*s = *s + ScopePrinter(prefix)
}

func (s ScopePrinter) printPrefix() {
	fmt.Printf("%s%s%s", color.FaintItalic, s, color.Reset)
	if len(s) > 0 {
		fmt.Print(" ")
	}
}

func (s *ScopePrinter) printMessage(message, status, statusColor string, arguments ...any) {
	s.printPrefix()
	fmt.Printf("[%s%s%s] %s\n", statusColor, status, color.Reset, fmt.Sprintf(message, arguments...))
}

func (s *ScopePrinter) OK(message string, arguments ...any) {
	s.printMessage(message, " OK      ", color.Green, arguments...)
}

func (s *ScopePrinter) Skipped(message string, arguments ...any) {
	s.printMessage(message, " Skipped ", color.Yellow, arguments...)
}

func (s *ScopePrinter) Info(message string, arguments ...any) {
	s.printMessage(message, " Info    ", color.Blue, arguments...)
}

func (s *ScopePrinter) Error(message string, arguments ...any) {
	s.printMessage(message, " Error   ", color.Red, arguments...)
}
