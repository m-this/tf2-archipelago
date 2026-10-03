package debugbundle

import (
	"fmt"
	"sort"
	"strings"
)

/* collected is the written record of what the bundle tried to take.
 *
 * Three bundles out of one crash wave held three different sets of files: a Linux
 * one with debug.log and no sourcemod/ logs, a Windows one with
 * sourcemod/errors_*.log and no debug.log, and one with both. Which of those is a
 * file that was missed and which is a file that never existed could not be told
 * apart from the outside, and the summary read the same either way.
 *
 * So the bundle writes down every file it went for and what happened. A file that
 * is not here is a fact with a reason beside it rather than a silence.
 */
type collected struct {
	rows []row
}

type row struct {
	// name is where the file sits in the zip, or the folder it would sit in.
	name string
	// from is where it was looked for, for the ones that arrived. For the ones
	// that did not, err already names the path.
	from string
	err  error
}

// note records one attempt. err nil means the file is in the bundle.
func (c *collected) note(name, from string, err error) {
	c.rows = append(c.rows, row{name: name, from: from, err: err})
}

/* render writes the record, the files that arrived first.
 *
 * A name reached from two places counts as taken when either worked: debug.log is
 * looked for in tf/ and in tf-dedicated/, and the one that is not there is not a
 * loss.
 */
func (c *collected) render() string {
	taken := map[string]bool{}
	for _, r := range c.rows {
		if r.err == nil {
			taken[r.name] = true
		}
	}

	var in, out []string
	for _, r := range c.rows {
		switch {
		case r.err == nil:
			in = append(in, fmt.Sprintf("  %-28s %s", r.name, r.from))
		case !taken[r.name]:
			out = append(out, fmt.Sprintf("  %-28s %v", r.name, r.err))
		}
	}
	sort.Strings(in)
	sort.Strings(out)

	var b strings.Builder
	b.WriteString("What this bundle went for, and what came of it.\n\n")
	b.WriteString("Every file below is one the bundle always tries to take. A bundle holding\n")
	b.WriteString("fewer files than another is not a bundle from a quieter run: it is this\n")
	b.WriteString("list with more of it under the second heading.\n")

	b.WriteString("\nIn the bundle\n")
	if len(in) == 0 {
		b.WriteString("  nothing, which should be impossible: summary.txt is written from memory.\n")
	}
	for _, line := range in {
		b.WriteString(line + "\n")
	}

	b.WriteString("\nNot in the bundle\n")
	if len(out) == 0 {
		b.WriteString("  nothing missing.\n")
	}
	for _, line := range out {
		b.WriteString(line + "\n")
	}
	return b.String()
}
