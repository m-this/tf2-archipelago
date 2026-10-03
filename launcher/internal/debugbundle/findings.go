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

The scan also carries away a few numbers that mean nothing on their own: the
TF2 build srcds printed at start, and how many bot purchases the game took and
refused. pins.go is what puts them beside something.
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

// sigmodUnresolvedRule is SigMod on a TF2 build it was not made for, loaded
// rather than refused. See the rule below for why both shapes exist.
const sigmodUnresolvedRule = "SigMod cannot find the game's addresses"

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
	{"SigMod did not load", func(l string) bool { return strings.Contains(l, "this SigMod build is for TF2 ServerVersion") }, "TF2 updated after this SigMod build was made, and the Windows build only\n" +
		"      runs on the TF2 build it was made for. The server plays without it:\n" +
		"      missions that need SigMod are refused. A launcher release with a new\n" +
		"      SigMod build fixes it."},
	/* The other half of the same trouble, and the half that was read by hand.

	   A SigMod build that checks the TF2 ServerVersion refuses to load and says
	   so, which the rule above catches. A build without that check loads and
	   then resolves nothing: every address it wants comes back FAIL and the
	   first one it uses faults. Shysoul's bundle was 96 of these for
	   CBaseEntity::m_DataMap alone, reported as a line that repeats a lot. */
	{sigmodUnresolvedRule, func(l string) bool { return strings.Contains(l, "AddrManager::GetAddr FAIL") }, "SigMod loaded and then could not resolve the addresses it needs. That is\n" +
		"      what a SigMod build made for another TF2 build does when it has no\n" +
		"      ServerVersion check to refuse on: it faults on the first one it uses.\n" +
		"      Compare the TF2 build above with the build the pins were checked against."},
	/* SigMod installs its own fault handler. It prints the stack it faulted on
	   and then calls ExitProcess, which is a clean exit as far as Windows is
	   concerned, so Breakpad never runs and there is no minidump to look for.
	   crashDumpNote reads this rather than sending the player hunting. */
	{sigmodFaultRule, func(l string) bool {
		return strings.Contains(l, "SigMod: fault ") || strings.Contains(l, "SigMod: ExitProcess(")
	}, "SigMod caught this fault itself and quit the process, so Breakpad wrote\n" +
		"      no dump. The two stacks it printed are in this bundle instead."},
	{"a plugin threw", func(l string) bool { return strings.Contains(l, "[SM] Exception reported:") }, ""},
	{"the plugin reported an error", func(l string) bool { return strings.Contains(l, "[AP] error:") }, ""},
	{"a bot got stuck", func(l string) bool { return strings.Contains(l, "[defenderbots] stuck:") }, ""},
	{"the bridge lost the room", func(l string) bool {
		return strings.Contains(l, "archipelago session ended")
	}, ""},
	{"RED went over its team size", func(l string) bool { return strings.Contains(l, "leaves.") }, ""},
}

// sigmodFaultRule is the name of the rule whose hits mean SigMod exited the
// process itself, which is why no minidump exists.
const sigmodFaultRule = "SigMod faulted and quit the server"

/* tf2BuildLine is srcds naming the TF2 build it is running, which it prints at
 * every start.
 *
 * The whole of 2026-10-02 came down to this number against the one SigMod was
 * keyed to, and the bundle listed every pin without it. It needs no RCON and no
 * `status`: srcds has already written it into the log the bundle collects.
 */
var tf2BuildLine = regexp.MustCompile(`Using Breakpad minidump system\. Version: (\d+)`)

/* The two ends of a defender bot's trip to the upgrade station.
 *
 * One refusal is ordinary: the bot asks for an upgrade the game will not apply
 * and takes the next one. Every purchase refused and none taken is a different
 * thing, and it is what SourceMod below git7255 does on TF2 11076587. Neither
 * count says anything alone, so neither is a rule; pins.go compares them.
 */
const purchaseRefused = "the game refused one, trying the next"

// purchaseTaken is the plugin's own record of a purchase the game applied:
// "Howser bought damage bonus [0] (primary) x1, and holds 400 credits."
func purchaseTaken(line string) bool {
	return strings.Contains(line, " bought ") && strings.Contains(line, ", and holds ")
}

type hit struct {
	count   int
	samples []string
	seen    map[string]bool
}

/* source is one log the scan reads.
 *
 * current says whether it belongs to the run this bundle was made from. A count
 * that only means something about one run cannot be taken across two: Cowser's
 * bundle held a working previous run with 329 purchases taken beside a current
 * run with none, and summed they look like an ordinary evening.
 */
type source struct {
	path    string
	current bool
}

// currentRun and previousRun name a source for the reader of a call.
func currentRun(path string) source  { return source{path: path, current: true} }
func previousRun(path string) source { return source{path: path} }

/* scan is what the logs said, as numbers. report() is the part that goes in the
 * summary; the fields beside it are the ones pins.go compares.
 */
type scan struct {
	lines   int
	found   map[string]*hit
	repeats map[string]int
	shape   map[string]string

	// tf2Build is the build srcds printed, preferring this run's logs.
	tf2Build string
	// refused and taken count this run's bot purchases, the two outcomes.
	refused int
	taken   int
}

// sawCrash says whether a crash was among the hits, which decides whether a
// missing minidump is worth pointing out.
func (s scan) sawCrash() bool { return s.found[crashRule] != nil }

// sigmodQuit says whether SigMod exited the process itself, which is why
// Breakpad left nothing behind.
func (s scan) sigmodQuit() bool { return s.found[sigmodFaultRule] != nil }

// scanLogs reads the named files and reports what matched, plus the lines that
// repeat far more than a log should need to.
func scanLogs(sources ...source) scan {
	got := scan{
		found:   map[string]*hit{},
		repeats: map[string]int{},
		shape:   map[string]string{},
	}
	var anyBuild string

	for _, src := range sources {
		file, err := os.Open(src.path)
		if err != nil {
			continue
		}
		reader := bufio.NewScanner(file)
		reader.Buffer(make([]byte, 0, 64<<10), 1<<20)
		var read int64
		for reader.Scan() && got.lines < scanLinesMax && read < scanBytesMax {
			got.lines++
			line := strings.TrimSpace(reader.Text())
			read += int64(len(line)) + 1
			if line == "" {
				continue
			}
			for _, r := range rules {
				if !r.match(line) {
					continue
				}
				h := got.found[r.name]
				if h == nil {
					h = &hit{}
					got.found[r.name] = h
				}
				h.count++
				// One sample per distinct message: the launcher writes some
				// lines twice, and three copies of one line is not three
				// examples.
				if len(h.samples) < samplesShown && !h.seen[shapeOf(line)] {
					if h.seen == nil {
						h.seen = map[string]bool{}
					}
					h.seen[shapeOf(line)] = true
					h.samples = append(h.samples, clip(line))
				}
			}
			if m := tf2BuildLine.FindStringSubmatch(line); m != nil {
				anyBuild = m[1]
				if src.current {
					got.tf2Build = m[1]
				}
			}
			if src.current {
				switch {
				case strings.Contains(line, purchaseRefused):
					got.refused++
				case purchaseTaken(line):
					got.taken++
				}
			}
			key := shapeOf(line)
			if key == "" {
				continue
			}
			got.repeats[key]++
			if _, seen := got.shape[key]; !seen {
				got.shape[key] = clip(line)
			}
		}
		_ = file.Close()
	}
	// Better the build from the run before this one than none: it is still the
	// build this install was running, and a bundle made before the server came
	// up this time has only that one.
	if got.tf2Build == "" {
		got.tf2Build = anyBuild
	}
	return got
}

func (s scan) report() string {
	if s.lines == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\nWhat looks wrong (%d log lines read)\n", s.lines)
	b.WriteString("This is a word search, not a diagnosis. A quiet section is not a clean run.\n\n")

	quiet := true
	for _, r := range rules {
		got := s.found[r.name]
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
	for key, count := range s.repeats {
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
			fmt.Fprintf(&b, "      %5d x  %s\n", p.count, s.shape[p.key])
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
