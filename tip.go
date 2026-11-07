package chart

import (
	"strconv"
	"strings"
)

// Tip is the panel a reader gets when they point at a chart mark or
// move focus to it.
//
// It replaces the title attribute. The browser draws a title attribute
// half a second late, in an OS font, at a wrap width nobody chose,
// never on touch, and never on keyboard focus. A Tip is an element the
// view renders and CSS shows. It opens at once, reads as the rest of
// the page, and opens on focus as well as on hover.
//
// One shape serves every chart, and the shape says where each kind of
// fact goes:
//
//   - Title is the mark's own name in full. An axis abbreviates, so
//     the title is the thing the reader does not have on screen yet.
//   - Image is a small picture beside the title, such as a logo, for a
//     mark that stands for one thing.
//   - Lines are the facts behind the mark, one labeled fact on each
//     line.
//   - Names is a list of the things the mark counts, capped, in its own
//     block. A list of proper nouns wraps where a labeled fact must not.
//
// The Has fields exist because hml has no truthiness: a view asks the
// builder.
type Tip struct {
	Title    string    `json:"title"`
	HasTitle bool      `json:"has_title"`
	Image    string    `json:"image"`
	HasImage bool      `json:"has_image"`
	Lines    []TipLine `json:"lines"`
	Names    string    `json:"names"`
	HasNames bool      `json:"has_names"`
	// Class says which way the panel opens. See NewTipAt.
	Class string `json:"class"`
}

// TipLine is one labeled fact.
//
// Class is empty for a top-level fact. [Tip.SubLine] sets it to the
// nested style, so a panel can break a total into its parts without a
// second list type.
type TipLine struct {
	Label    string `json:"label"`
	HasLabel bool   `json:"has_label"`
	Value    string `json:"value"`
	Class    string `json:"class"`
	// Names lists the things behind this fact, below its value.
	Names    string `json:"names"`
	HasNames bool   `json:"has_names"`
}

// TipNameLimit is how many names a tip lists before it counts the
// rest.
//
// A tip is a glance. The full list belongs in a table under the chart.
// The names in a tip say what kind of thing is in the mark, and the
// count says how many there are. Forty names make a panel taller than
// the chart it covers, and nobody reads to the end.
const TipNameLimit = 8

// NewTip starts a tip for the mark at index i of n marks across the
// plot, such as a bar in a row of bars.
func NewTip(title string, i, n int) Tip {
	return newTip(title, tipClass(i, n))
}

// NewTipAt starts a tip for a mark at pos, its fraction of the way
// across the plot from 0 to 1. Use it when the marks are not evenly
// spaced: a line chart places a point by its date, so the point's index
// says nothing about which way its panel has room to open.
//
// A panel near the left edge opens to the right, one near the right
// edge opens to the left, and one in the middle centers on its mark.
// The split is by thirds and not by first and last, because a centered
// panel needs half its width of room on each side. The builder knows
// the mark's position and the panel has a fixed width, so no script
// measures anything.
func NewTipAt(title string, pos float64) Tip {
	return newTip(title, tipClassAt(pos))
}

// NewStackedTip starts a tip for a chart whose marks run down the page,
// such as a bar for each row. Each mark spans the plot's width, so the
// panel opens to the right from the left edge, which is the side a
// narrow plot has room on.
func NewStackedTip(title string) Tip {
	return newTip(title, "chart-tip--left")
}

func newTip(title, class string) Tip {
	title = strings.TrimSpace(title)
	return Tip{Title: title, HasTitle: title != "", Class: class}
}

// WithImage puts a picture beside the title. A blank URL adds nothing:
// a row of placeholder squares says less than no picture at all.
func (t Tip) WithImage(url string) Tip {
	t.Image = strings.TrimSpace(url)
	t.HasImage = t.Image != ""
	return t
}

// Line adds a labeled fact. A blank label adds a value alone.
func (t Tip) Line(label, value string) Tip {
	t.Lines = append(t.Lines, newTipLine(label, value, ""))
	return t
}

// SubLine adds a fact under the line above it, indented and quieter:
// one part of a total. The total is the number the reader came for,
// and the parts are what they check it against.
func (t Tip) SubLine(label, value string) Tip {
	t.Lines = append(t.Lines, newTipLine(label, value, "chart-tip-line--sub"))
	return t
}

func newTipLine(label, value, class string) TipLine {
	label = strings.TrimSpace(label)
	return TipLine{Label: label, HasLabel: label != "", Value: value, Class: class}
}

// Count adds a labeled whole number, delimited as an axis prints it.
func (t Tip) Count(label string, n int) Tip {
	return t.Line(label, delimited(n))
}

// CountWithNames adds a count and the names behind it, capped at
// [TipNameLimit].
func (t Tip) CountWithNames(label string, n int, names []string) Tip {
	t = t.Count(label, n)
	line := &t.Lines[len(t.Lines)-1]
	line.Names = capNames(names)
	line.HasNames = line.Names != ""
	return t
}

// WithNames adds the list of names, capped at [TipNameLimit], with a
// count of the rest.
//
// It takes one element for each name, not a joined string. Splitting
// "Acme, Inc., Beta" on commas makes three names of two and counts the
// rest wrong.
func (t Tip) WithNames(names []string) Tip {
	t.Names = capNames(names)
	t.HasNames = t.Names != ""
	return t
}

// capNames keeps the first TipNameLimit names that are not blank and
// says how many it left out.
func capNames(names []string) string {
	kept := make([]string, 0, min(len(names), TipNameLimit+1))
	rest := 0
	for _, name := range names {
		if name = strings.TrimSpace(name); name == "" {
			continue
		}
		if len(kept) < TipNameLimit {
			kept = append(kept, name)
			continue
		}
		rest++
	}
	if rest == 0 {
		return strings.Join(kept, ", ")
	}
	return strings.Join(kept, ", ") + ", +" + strconv.Itoa(rest) + " more"
}

// tipClass is tipClassAt for the mark at index i of n evenly spaced
// marks.
func tipClass(i, n int) string {
	if n <= 1 {
		return "chart-tip--center"
	}
	return tipClassAt(float64(i) / float64(n-1))
}

// tipClassAt is the class that opens a panel toward the side of the
// plot with room for it. Each class is written out, so a search for
// the selector finds the code that sets it.
func tipClassAt(pos float64) string {
	switch {
	case pos < 1.0/3.0:
		return "chart-tip--left"
	case pos > 2.0/3.0:
		return "chart-tip--right"
	default:
		return "chart-tip--center"
	}
}
