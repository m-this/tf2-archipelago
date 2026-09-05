package debugbundle

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

/* findings is what a reader would otherwise grep for.

Every rule here was earned by a bundle that took a long time to read. A crash
reported as "bridge stopping", a Spy that only a screenshot proved, and one
debug line repeated 282 times in a session all cost more to find than they
should have. The summary is the first file anybody opens, so what looks wrong
belongs in it.

Nothing here diagnoses. It counts and it quotes, and the reader decides. A rule
that guesses at a cause would be wrong in exactly the cases worth reading.
*/

// The scan is bounded: a log is megabytes and a summary is a page.
const (
	scanLinesMax  = 400_000
	scanBytesMax  = 32 << 20
	quotedLineMax = 160
	repeatsShown  = 5
	repeatsFloor  = 20
	samplesShown  = 3
)

/*
	noise strips what makes two copies of one message look like two messages.

The same line reaches the bundle up to four ways: the launcher log prefixes a
clock and a source column, the SourceMod log prefixes its own date, the plugin
tags itself, and the console carries the plain text. Counting those separately
turned one 282-line loop into four entries of about 70 and buried the thing
worth seeing.

Order matters. Each pattern runs against what the one before it left.
*/
var noise = []*regexp.Regexp{
	regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\s+\S+\s+`),                  // launcher: clock and source
	regexp.MustCompile(`^L \d{2}/\d{2}/\d{4} - \d{2}:\d{2}:\d{2}:\s*`), // SourceMod: its own date
	regexp.MustCompile(`^\[[A-Za-z0-9_]+\.smx\]\s*`),                   // the plugin naming itself
	regexp.MustCompile(`^\[AP\] (debug|error):\s*`),                    // the same event on two channels
	regexp.MustCompile(`\d+`),                                          // counters, ids, coordinates
}

// shapeOf reduces a line to what repeats about it.
func shapeOf(line string) string {
	for _, pattern := range noise {
		line = pattern.ReplaceAllString(line, "")
	}
	return strings.TrimSpace(line)
}

type rule struct {
	name  string
	match func(string) bool
	// hint says what a hit means, for the rules where the line alone does
	// not: it is printed once under the samples.
	hint string
}

// pluginMissingRule is the name of the rule whose hits mean the server is
// playing without the plugin.
const pluginMissingRule = "the plugin is not loaded"

// rules are ordered by how much a hit matters, because that is the order the
// summary prints them in.
// crashRule is the name of the rule whose hits mean a minidump should exist.
const crashRule = "the game server crashed"

var rules = []rule{
	{crashRule, func(l string) bool {
		return strings.Contains(l, "CRASHED") ||
			strings.Contains(l, "exit status 0xc0") ||
			strings.Contains(l, "signal: segmentation fault") ||
			strings.Contains(l, "signal: abort")
	}, ""},
	{pluginMissingRule, func(l string) bool {
		return strings.Contains(l, `Unknown command "tf2ap_`) ||
			strings.Contains(l, `Unknown command "sm_redbots_manager`)
	}, "Metamod or SourceMod did not load, so the server is playing stock Mann vs\n" +
		"      Machine: nothing locked, no checks sent, the settings above ignored.\n" +
		"      Look under tf-dedicated/tf/addons for metamod.vdf, metamod/bin and\n" +
		"      sourcemod/bin. The launcher reinstalls whichever is missing on the next start."},
	{"a plugin threw", func(l string) bool { return strings.Contains(l, "[SM] Exception reported:") }, ""},
	{"the plugin reported an error", func(l string) bool { return strings.Contains(l, "[AP] error:") }, ""},
	{"a bot got stuck", func(l string) bool { return strings.Contains(l, "[defenderbots] stuck:") }, ""},
	{"the bridge lost the room", func(l string) bool {
		return strings.Contains(l, "archipelago session ended")
	}, ""},
	{"RED went over its team size", func(l string) bool { return strings.Contains(l, "leaves.") }, ""},
}

type hit struct {
	count   int
	samples []string
	seen    map[string]bool
}

// scanLogs reads the named files and reports what matched, plus the lines that
// repeat far more than a log should need to. The second return says whether a
// crash was among them, which decides whether a missing minidump is worth
// pointing out.
func scanLogs(paths ...string) (string, bool) {
	found := map[string]*hit{}
	repeats := map[string]int{}
	shape := map[string]string{}

	lines := 0
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		reader := bufio.NewScanner(file)
		reader.Buffer(make([]byte, 0, 64<<10), 1<<20)
		var read int64
		for reader.Scan() && lines < scanLinesMax && read < scanBytesMax {
			lines++
			line := strings.TrimSpace(reader.Text())
			read += int64(len(line)) + 1
			if line == "" {
				continue
			}
			for _, r := range rules {
				if !r.match(line) {
					continue
				}
				got := found[r.name]
				if got == nil {
					got = &hit{}
					found[r.name] = got
				}
				got.count++
				// One sample per distinct message: the launcher writes some
				// lines twice, and three copies of one line is not three
				// examples.
				if len(got.samples) < samplesShown && !got.seen[shapeOf(line)] {
					if got.seen == nil {
						got.seen = map[string]bool{}
					}
					got.seen[shapeOf(line)] = true
					got.samples = append(got.samples, clip(line))
				}
			}
			key := shapeOf(line)
			if key == "" {
				continue
			}
			repeats[key]++
			if _, seen := shape[key]; !seen {
				shape[key] = clip(line)
			}
		}
		_ = file.Close()
	}
	if lines == 0 {
		return "", false
	}
	return render(found, repeats, shape, lines), found[crashRule] != nil
}

func render(found map[string]*hit, repeats map[string]int, shape map[string]string, lines int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nWhat looks wrong (%d log lines read)\n", lines)
	b.WriteString("This is a word search, not a diagnosis. A quiet section is not a clean run.\n\n")

	quiet := true
	for _, r := range rules {
		got := found[r.name]
		if got == nil {
			continue
		}
		quiet = false
		fmt.Fprintf(&b, "  %s (%d)\n", r.name, got.count)
		for _, sample := range got.samples {
			fmt.Fprintf(&b, "      %s\n", sample)
		}
		if r.hint != "" {
			fmt.Fprintf(&b, "      %s\n", r.hint)
		}
	}
	if quiet {
		b.WriteString("  nothing matched.\n")
	}

	type pair struct {
		key   string
		count int
	}
	var loud []pair
	for key, count := range repeats {
		if count >= repeatsFloor {
			loud = append(loud, pair{key, count})
		}
	}
	sort.Slice(loud, func(i, j int) bool {
		if loud[i].count != loud[j].count {
			return loud[i].count > loud[j].count
		}
		return loud[i].key < loud[j].key
	})
	if len(loud) > repeatsShown {
		loud = loud[:repeatsShown]
	}
	if len(loud) > 0 {
		b.WriteString("\n  Lines repeated the most. A loop that cannot settle looks like this.\n")
		for _, p := range loud {
			fmt.Fprintf(&b, "      %5d x  %s\n", p.count, shape[p.key])
		}
	}
	return b.String()
}

func clip(line string) string {
	if len(line) <= quotedLineMax {
		return line
	}
	return line[:quotedLineMax] + " ..."
}
