package tenda

import (
	"fmt"
	"strconv"
	"strings"
)

// ListFormat packs a list field's rows and columns with two separators
// (docs: Conventions -> POST). It does no per-cell escaping; a feature whose
// doc section says a cell is url-encoded does that itself.
type ListFormat struct{ Row, Col string }

var (
	TildeComma = ListFormat{"~", ","}   // SetVirtualServerCfg, SetStaticRouteCfg
	TildeSemi  = ListFormat{"~", ";"}   // setPptpUserList
	LineCR     = ListFormat{"\n", "\r"} // SetNetControlList, SetIpMacBind, setMacFilterCfg
)

// SeparatorError is a cell that contains one of the format's separators.
type SeparatorError struct {
	Row, Col int
	Value    string
}

func (e *SeparatorError) Error() string {
	return fmt.Sprintf("list row %d column %d: value %s contains a list separator", e.Row+1, e.Col+1, strconv.Quote(e.Value))
}

// Encode joins rows. No rows encode to "".
func (f ListFormat) Encode(rows [][]string) (string, error) {
	lines := make([]string, len(rows))
	for i, row := range rows {
		for j, cell := range row {
			if strings.Contains(cell, f.Row) || strings.Contains(cell, f.Col) {
				return "", &SeparatorError{Row: i, Col: j, Value: cell}
			}
		}
		lines[i] = strings.Join(row, f.Col)
	}
	return strings.Join(lines, f.Row), nil
}

// Decode splits s. "" decodes to nil; trailing empty cells are kept.
func (f ListFormat) Decode(s string) [][]string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, f.Row)
	rows := make([][]string, len(lines))
	for i, l := range lines {
		rows[i] = strings.Split(l, f.Col)
	}
	return rows
}
