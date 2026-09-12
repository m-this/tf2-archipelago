# tf2-archipelago

An Archipelago randomiser for TF2 Mann vs Machine. It has an apworld, a bridge,
a SourceMod plugin, and a Windows launcher that does the server setup for you.

- `CONTEXT.md` — read it first. Archipelago and TF2 use the same words for
  different things, and this file defines both.
- `apworld/` — the Archipelago world. The logic is in `rules.py`.
- `bridge/` — Go. It connects the game server to the multiworld.
- `plugin/` — the SourceMod side. `launcher/` — the Windows .exe that players run.

## The settings are declared once

`launcher/internal/form` holds one `Spec` per row: its label, its help, what
values it takes, and how it is read off and written back. `Build` turns the
specs and the state into a `Model` of plain data, and every interface draws
that. No closures and no styling in a `Model`, so it encodes as JSON and the
planned web mode is a renderer rather than a fourth hand-written list.

    Build(state, env) -> Model -> the interface draws it
                                      |
                                      v
                    Apply(state, env, Change) -> State -> Build again

Adding a setting is a line in `form.Specs` and nothing else. Adding one to an
interface is not a thing that happens.

- `Specs` is a function, not a variable. The Missions page has a row per
  mission and which missions exist depends on the asset packs on disk; the
  Loadouts page has a row per slot and the Spy's slots are not everybody's.
- `State` is the settings plus a `Draft`. The team name being typed and the
  loadout being built are not settings: nothing in the config file holds them
  and leaving the page throws them away. Both interfaces used to keep that
  scratch themselves, which is why the loadout builder was the one part the two
  did not agree about even in shape.
- A row that is parsed rather than stored is `Deferred`. The room address is
  the only one: reaching `archipelago.gg:12345` means passing through `a`, so
  the row keeps what was typed and Save is where an address that never became
  one is refused.
- `TestEverySettingIsOnAPageOrSaysWhyNot` walks `settings.Settings` and refuses
  a field no row writes. `notOnAPage` carries the exceptions with the reason
  for each. An entry there is a decision, not a way to quieten the test.
- `settings.Persist` is what saving means, for every interface: fill in the RCON
  password, refuse settings nothing could act on, write the file. Call it while
  the screen is still up. A write that fails has to be reported with the
  player's answers still in front of them, and reporting it afterwards is what
  `apw-ep5` was reported as.
- A missing or unparseable room does not refuse a Save. It used to, and that was
  the mechanism of `apw-ep5`: the reporter was setting a login token two pages
  from a field they could not see. `internal/roomcheck` asks the room once the
  settings are written, and points at archipelago.gg or Test mode when there is
  nothing there. Advice, never a refusal.

There is one interface now and it is a browser. `internal/webapi` holds the
launcher's state and answers the contract in `proto/tf2ap/launcher/v1`; the
Angular app in `launcher/web` renders `form.Model` and nothing else. The window,
the terminal interface and Kelly's hand-written page are gone, and so are walk,
bubbletea, lipgloss and termenv. `launcher/README.md` has the shape of it.

What that bought: a setting is one `Spec` and no interface has a list to keep in
step, a person helping a player can be looked at the same screen, and Linux
stopped getting the interface that was unreadable next to the server's own
output.

`uiparity` is gone with them. It compared the window's source with the
terminal's using a regular expression, which could only ask whether the two
wrote the same struct fields and never whether they told the player the same
thing. They did not: nine rows had different help text and nobody had chosen one
of the differences. Where they differed the window's wording was kept, and that
wording is what the browser draws.

## The browser interface, without a game server

`launcher/cmd/fakelauncher` answers the whole contract with nothing behind it:
the real Connect handlers, the real WebSocket and a real `form.Model` built from
`settings.Defaults`, with no game server, no bridge and no multiworld. `make
web-e2e` drives the real app against it in a browser, so the run is the same
every time on a machine with no Team Fortress 2 on it.

The fake is not a second launcher. Every answer it gives is built by the same
`Snapshot.Proto` the real one uses, and its mission table is built from the rows
`form` actually declares, so a table of invented ids cannot pass. What it makes
up is the state a run would be in.

`make web-check` is the rest: eslint, prettier, the component tests, and `ng
build` under a 500 kB initial budget that is an error rather than a warning.

## Running the plugin without a server

`make toolchain` builds SourcePawn's standalone compiler and VM, using the
defender mod's own script out of the module cache so the pinned commit cannot
drift between the two repositories. `gamedata/spdriver_test.go` then compiles a
plugin function and runs it on inputs a test chooses.

`plugin/scripting/tf2_archipelago/weapon_buffs_math.inc` is what makes that
possible. It holds the decisions that need nothing but their arguments and the
generated tables, it includes nothing itself, and both the plugin and the driver
include it. A function belongs there when it needs no native: no entity
property, no SDKCall, no TF2 lookup, not even `RoundToFloor` or `StrEqual`,
which the standalone VM does not bind, and no constant out of a SourceMod
include either.

Prefer running a function to reading it. The tests that look for substrings in
`weapon_buffs.inc` cannot tell a rename from a rewrite: inverting the cooldown
floor so every cooldown collapses to the minimum leaves every watched string in
place and they pass. What is still read is what needs the engine, and moving one
of those over means first making it need nothing, which is a change to the
plugin rather than to its tests.

`make check` sets `TF2AP_REQUIRE_SPSHELL`, so a driver fails there rather than
skipping. Without the toolchain a developer gets a skip naming what to run.

## Build

Use the Makefile. `make help` lists every target.

```bash
make test lint                  # Go
make apworld-test apworld-lint
make seed                       # generate a seed
make up down logs rcon          # start the server on this machine
```

Build on Linux. The plugin compiler spcomp is a Linux binary, and every
launcher asset build compiles the plugin. So `make launcher` and `make check`
do not run on Windows. Use WSL, which is what CI uses.

`go build` still works anywhere for a compile check. The launcher it makes
carries a placeholder plugin, so never ship one built that way.

Move the defender mod with `go get github.com/m-this/tf2-mvm-bots-go@<tag>`.
The go.mod requirement is what the debug bundle prints, so a bundle now names a
release rather than a commit, and turning one back into a commit means resolving
the tag against the repository. `bots-pin-check` used to refuse a tag for that
reason and is gone: `go get` rewrites a tagged commit to its tag, so the check
and tagging a release could never both hold.

`.github/workflows/release.yml` copies the `CHANGELOG.md` section that matches
the tag into the release notes. Players read it. Keep developer notes out.

## Triage from Discord

All discussion happens in Discord and no bot reads it. The maintainer copies the
chat here. Turn it into beads.

- Read `bd list` first. Comment on the open bead. Do not create a second one.
- Quote the reporter and give their name. Put your reading under the quote.
- If the message does not show the cause, say so and leave the bead a report.
- Add the label `discord`. Drop the messages that report nothing, then say so.
- Choose the repository by the cause, not by the symptom. The wave-loss money
  bug appeared in the mod and belonged here.
- Read the close reason before you reopen a bead. It names the trigger measured.

## Beads in this repository

The block below has the commands. These facts are local to this repository.

- The prefix is `apw`. P0 crash, P1 costs a player a run, P2 bug, P3 polish.
- Git tracks `.beads/issues.jsonl`. Git ignores the Dolt database.
- There is no Dolt remote. A new clone needs `bd init --from-jsonl`.
- Two sessions share one database and can overwrite each other. Check with
  `bd show` before you set a status.


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:970c3bf2 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Agent Context Profiles

The managed Beads block is task-tracking guidance, not permission to override repository, user, or orchestrator instructions.

- **Conservative (default)**: Use `bd` for task tracking. Do not run git commits, git pushes, or Dolt remote sync unless explicitly asked. At handoff, report changed files, validation, and suggested next commands.
- **Minimal**: Keep tool instruction files as pointers to `bd prime`; use the same conservative git policy unless active instructions say otherwise.
- **Team-maintainer**: Only when the repository explicitly opts in, agents may close beads, run quality gates, commit, and push as part of session close. A current "do not commit" or "do not push" instruction still wins.

## Session Completion

This protocol applies when ending a Beads implementation workflow. It is subordinate to explicit user, repository, and orchestrator instructions.

1. **File issues for remaining work** - Create beads for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Handle git/sync by active profile**:
   ```bash
   # Conservative/minimal/default: report status and proposed commands; wait for approval.
   git status

   # Team-maintainer opt-in only, unless current instructions forbid it:
   git pull --rebase
   bd dolt push
   git push
   git status
   ```
5. **Hand off** - Summarize changes, validation, issue status, and any blocked sync/commit/push step

**Critical rules:**
- Explicit user or orchestrator instructions override this Beads block.
- Do not commit or push without clear authority from the active profile or the current user request.
- If a required sync or push is blocked, stop and report the exact command and error.
<!-- END BEADS INTEGRATION -->
