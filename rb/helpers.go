// Package rb — utility helpers used across the report-builder pipeline.
package rb

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Safe map accessors — work on the map[string]any rows returned by queryGenericRows
// ---------------------------------------------------------------------------

func getStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(val))
		return n
	default:
		return 0
	}
}

func getFloat(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(val), 64)
		return f
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

func filterEmptyStrings(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Currency formatting — mirrors formatCurrency from currencyFormatter.js
// Returns "1,234.56" (no leading $); callers add $ as needed.
// ---------------------------------------------------------------------------

func formatCurrency(amount float64) string {
	rounded := math.Round(amount*100) / 100
	intPart := int64(rounded)
	decPart := int64(math.Abs(math.Round((rounded-float64(intPart))*100)))

	intStr := strconv.FormatInt(intPart, 10)
	start := 0
	prefix := ""
	if len(intStr) > 0 && intStr[0] == '-' {
		prefix = "-"
		start = 1
	}
	digits := intStr[start:]

	var buf strings.Builder
	for i, ch := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			buf.WriteByte(',')
		}
		buf.WriteRune(ch)
	}
	return fmt.Sprintf("%s%s.%02d", prefix, buf.String(), decPart)
}

// dollarFormat returns "$0.00" for zero/negative values, "$X,XXX.XX" otherwise.
func dollarFormat(amount float64) string {
	if amount <= 0 {
		return "$0.00"
	}
	return "$" + formatCurrency(amount)
}

// ---------------------------------------------------------------------------
// Date / time formatting — mirrors moment(date).format(...)
// ---------------------------------------------------------------------------

var dateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseAnyDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// formatDate mirrors moment(date).format('MM-DD-YYYY')
func formatDate(s string) string {
	if t, ok := parseAnyDate(s); ok {
		return t.Format("01-02-2006")
	}
	return "N/A"
}

// formatTime12h mirrors moment(date).format('h:mm A')
func formatTime12h(s string) string {
	if t, ok := parseAnyDate(s); ok {
		return t.Format("3:04 PM")
	}
	return "N/A"
}

// ---------------------------------------------------------------------------
// Eastern-time helpers — mirror getSearchDate / getSearchTime / getEstOrEdt
// ---------------------------------------------------------------------------

func getSearchDate(t time.Time) string {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return t.Format("01-02-2006")
	}
	return t.In(loc).Format("01-02-2006")
}

func getSearchTime(t time.Time) string {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return t.Format("3:04 PM")
	}
	return t.In(loc).Format("3:04 PM")
}

// getEstOrEdt mirrors getEstOrEdt() — returns "EDT" during daylight saving, "EST" otherwise.
func getEstOrEdt(t time.Time) string {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return "EST"
	}
	name, _ := t.In(loc).Zone()
	if name == "EDT" {
		return "EDT"
	}
	return "EST"
}

// ---------------------------------------------------------------------------
// HTML stripping — mirrors the rawHtml.replace(/<[^>]*>/g,'') + /&nbsp;|\s/g pattern
// Used to determine whether Report_Cover_Info has meaningful content.
// ---------------------------------------------------------------------------

var (
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
	spaceRe   = regexp.MustCompile(`&nbsp;|\s+`)
	urlRe     = regexp.MustCompile(`(https?://[^\s]+)`)
)

// cleanHTMLForCheck strips tags and whitespace to check if content is non-empty.
func cleanHTMLForCheck(raw string) string {
	noTags := htmlTagRe.ReplaceAllString(raw, "")
	return spaceRe.ReplaceAllString(noTags, "")
}

// linkifyURLs wraps bare URLs in anchor tags — mirrors the disclaimer regex replace.
func linkifyURLs(text string) string {
	return urlRe.ReplaceAllString(text, `<a href="$1" target="_blank">$1</a>`)
}

// delinqLabel converts the Prior_Delinquencies integer to a display string.
func delinqLabel(v int) string {
	switch v {
	case 1:
		return "YES"
	case 0:
		return "NO"
	default:
		return "N/A"
	}
}
