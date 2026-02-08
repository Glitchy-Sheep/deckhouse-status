package display

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if days == 0 && minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	return strings.Join(parts, " ")
}

// parseTZ parses a timezone string: IANA name ("Europe/Moscow") or numeric offset ("+3", "-5").
func parseTZ(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	tz = strings.TrimSpace(tz)
	if offset, err := strconv.Atoi(tz); err == nil && offset >= -12 && offset <= 14 {
		name := fmt.Sprintf("UTC%+d", offset)
		return time.FixedZone(name, offset*3600)
	}
	return time.FixedZone("MSK", 3*3600)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ParsePRTag extracts PR number and edition from an image tag.
// "pr15160" → (15160, "FE")
// "pr15160-ce" → (15160, "CE")
// "main" → (0, "")
var prTagRe = regexp.MustCompile(`^pr(\d+)(?:-(.+))?$`)

func ParsePRTag(tag string) (int, string) {
	matches := prTagRe.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return 0, ""
	}
	n, _ := strconv.Atoi(matches[1])
	edition := "FE"
	if len(matches) > 2 && matches[2] != "" {
		edition = strings.ToUpper(matches[2])
	}
	return n, edition
}
