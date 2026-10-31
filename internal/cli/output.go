package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wendersoon/docker-ephemeral/internal/app"
)

const shortIDLength = 12

func printTeardown(w io.Writer, t app.Teardown) {
	fmt.Fprintf(w, "Session %s removed: %s\n", t.Session.DisplayName(), t.Report)
	for _, tag := range t.Report.Restored {
		fmt.Fprintf(w, "  restored tag %s to its previous image\n", tag)
	}
	for _, s := range t.Report.Skipped {
		fmt.Fprintf(w, "  kept %s %s: %s\n", s.Kind, shortID(s.ID), s.Reason)
	}
}

func printTeardowns(w io.Writer, teardowns []app.Teardown) {
	for _, t := range teardowns {
		printTeardown(w, t)
	}
}

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > shortIDLength {
		return id[:shortIDLength]
	}
	return id
}

func humanizeSince(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "less than a minute ago"
	case d < time.Hour:
		return plural(int(d.Minutes()), "minute") + " ago"
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hour") + " ago"
	default:
		return plural(int(d.Hours()/24), "day") + " ago"
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
