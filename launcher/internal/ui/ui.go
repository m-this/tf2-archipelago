// Package ui is the launcher's interactive prompt layer. Each prompt takes the
// saved-config value as the default, so a returning operator hits Enter to keep
// what they had. Nothing here reads or writes the config file; the caller does.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Prompt is the interactive surface, backed by stdin/stdout. Methods return the
// value the operator entered or, on an empty line, the default.
type Prompt struct {
	reader *bufio.Reader
}

// New returns a Prompt reading from stdin.
func New() *Prompt {
	return &Prompt{reader: bufio.NewReader(os.Stdin)}
}

// Text asks for a string. The default is shown in brackets and used on Enter.
func (p *Prompt) Text(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := p.reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// Password asks for a string without echoing it. The default is never shown;
// an empty answer keeps it. On Windows the no-echo is handled by termReadLine.
func (p *Prompt) Password(label, def string) string {
	fmt.Printf("%s%s: ", label, maskedDefault(def))
	line := p.readMaskedLine()
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// Int asks for an integer. The default is shown and used on Enter. A non-numeric
// answer re-prompts.
func (p *Prompt) Int(label string, def int) int {
	for {
		fmt.Printf("%s [%d]: ", label, def)
		line, _ := p.reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println("  enter a whole number, or press Enter for the default")
			continue
		}
		return n
	}
}

// Bool asks for yes/no. The default is shown and used on Enter.
func (p *Prompt) Bool(label string, def bool) bool {
	d := "n"
	if def {
		d = "y"
	}
	for {
		fmt.Printf("%s [%s]: ", label, d)
		line, _ := p.reader.ReadString('\n')
		line = strings.ToLower(strings.TrimSpace(line))
		switch line {
		case "":
			return def
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
		fmt.Println("  enter y or n, or press Enter for the default")
	}
}

// Choice asks for one of a fixed set of options. The default is shown and used
// on Enter; an unknown answer re-prompts.
func (p *Prompt) Choice(label string, options []string, def string) string {
	for {
		fmt.Printf("%s %v [%s]: ", label, options, def)
		line, _ := p.reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		for _, opt := range options {
			if strings.EqualFold(line, opt) {
				return opt
			}
		}
		fmt.Printf("  pick one of %v, or press Enter for %s\n", options, def)
	}
}

func maskedDefault(def string) string {
	if def == "" {
		return ""
	}
	return " (set, press Enter to keep)"
}

func (p *Prompt) readMaskedLine() string {
	if line, err := termReadLine(p.reader); err == nil {
		return line
	}
	line, _ := p.reader.ReadString('\n')
	return line
}

// Option is one row of a Select: the value that gets saved, and the line the
// player reads.
type Option struct {
	Value string
	Label string
}

// Select asks for one of a numbered list. It takes the number, the value, or
// Enter for the default, which is marked in the list. A list beats free text
// wherever the set is known and short: nobody has to remember how ghost_town
// is spelled.
func (p *Prompt) Select(label string, options []Option, def string) string {
	fmt.Printf("%s:\n", label)
	for i, option := range options {
		marker := " "
		if option.Value == def {
			marker = "*"
		}
		fmt.Printf("  %s %2d. %s\n", marker, i+1, option.Label)
	}
	for {
		fmt.Printf("  Number or name [%s]: ", def)
		line, _ := p.reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		if n, err := strconv.Atoi(line); err == nil {
			if n >= 1 && n <= len(options) {
				return options[n-1].Value
			}
			fmt.Printf("  pick a number between 1 and %d\n", len(options))
			continue
		}
		for _, option := range options {
			if strings.EqualFold(line, option.Value) {
				return option.Value
			}
		}
		fmt.Println("  pick a number, or type one of the names")
	}
}

// IntRange asks for a number inside a range, and says what the range is. An
// answer outside it re-prompts rather than being clamped, because a silently
// corrected number is one the player never sees.
func (p *Prompt) IntRange(label string, def, low, high int) int {
	if def < low {
		def = low
	}
	if def > high {
		def = high
	}
	for {
		fmt.Printf("%s (%d to %d) [%d]: ", label, low, high, def)
		line, _ := p.reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println("  enter a whole number, or press Enter for the default")
			continue
		}
		if n < low || n > high {
			fmt.Printf("  pick a number between %d and %d\n", low, high)
			continue
		}
		return n
	}
}
