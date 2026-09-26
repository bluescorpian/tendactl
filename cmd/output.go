package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// render writes v as indented JSON under -o json, otherwise calls text.
// Stdout carries only results.
func (a *app) render(cmd *cobra.Command, v any, text func(w io.Writer) error) error {
	out := cmd.OutOrStdout()
	if a.output == "json" {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		_, err = out.Write(append(b, '\n'))
		return err
	}
	return text(out)
}

type col struct {
	title string
	right bool
}

// table writes a header, a rule of "─" per column, then rows. Each column is
// as wide as its widest cell; columns are separated by one space and
// trailing spaces are trimmed.
func table(w io.Writer, cols []col, rows [][]string) error {
	width := make([]int, len(cols))
	for i, c := range cols {
		width[i] = utf8.RuneCountInString(c.title)
	}
	for _, r := range rows {
		for i := range cols {
			if i < len(r) {
				width[i] = max(width[i], utf8.RuneCountInString(r[i]))
			}
		}
	}
	line := func(cells []string) string {
		var sb strings.Builder
		for i, c := range cols {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			pad := strings.Repeat(" ", width[i]-utf8.RuneCountInString(cell))
			if i > 0 {
				sb.WriteByte(' ')
			}
			if c.right {
				sb.WriteString(pad + cell)
			} else {
				sb.WriteString(cell + pad)
			}
		}
		return strings.TrimRight(sb.String(), " ") + "\n"
	}
	titles := make([]string, len(cols))
	rules := make([]string, len(cols))
	for i, c := range cols {
		titles[i] = c.title
		rules[i] = strings.Repeat("─", width[i])
	}
	var sb strings.Builder
	sb.WriteString(line(titles))
	sb.WriteString(line(rules))
	for _, r := range rows {
		sb.WriteString(line(r))
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

// fields writes alternating label, value pairs as "Label:  value" lines,
// with labels padded to the longest.
func fields(w io.Writer, kv ...string) error {
	if len(kv)%2 != 0 {
		return fmt.Errorf("fields: odd number of arguments")
	}
	longest := 0
	for i := 0; i < len(kv); i += 2 {
		longest = max(longest, utf8.RuneCountInString(kv[i])+1)
	}
	var sb strings.Builder
	for i := 0; i < len(kv); i += 2 {
		label := kv[i] + ":"
		l := label + strings.Repeat(" ", longest-utf8.RuneCountInString(label)) + "  " + kv[i+1]
		sb.WriteString(strings.TrimRight(l, " ") + "\n")
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

// done reports a mutation's success in table mode; JSON mode prints nothing.
func (a *app) done(cmd *cobra.Command, format string, args ...any) error {
	if a.output == "json" {
		return nil
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
	return err
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// securityLabel renders a WifiBasicSet/WifiExtraSet security enum ("none",
// "wpapsk", "wpa2psk", "wpawpa2psk") as the UI's "Encryption Mode" text.
// Shared by wifi show and system status.
func securityLabel(security string) string {
	switch security {
	case "none":
		return "None"
	case "wpapsk":
		return "WPA-PSK"
	case "wpa2psk":
		return "WPA2-PSK"
	case "wpawpa2psk":
		return "WPA/WPA2-PSK"
	default:
		return security
	}
}

// ledModeLabel renders SetLEDCfg's ledType ("close"/"open"/"time") as the
// UI's "LED Control" text.
func ledModeLabel(mode string) string {
	switch mode {
	case "close":
		return "Always off"
	case "open":
		return "Always on"
	case "time":
		return "Schedule"
	default:
		return mode
	}
}

// ledCloseTypeLabel renders ledCloseType ("allClose"/"unpowerClose") as the
// UI's "Indicator" text. Shared by led and sleep, which both expose this
// field.
func ledCloseTypeLabel(closeType string) string {
	switch closeType {
	case "allClose":
		return "All off"
	case "unpowerClose":
		return "All off except power"
	default:
		return closeType
	}
}

// maskSecret hides s unless show is set.
func maskSecret(s string, show bool) string {
	if show || s == "" {
		return s
	}
	return "********"
}
