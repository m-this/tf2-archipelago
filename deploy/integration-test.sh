#!/bin/sh
# End-to-end test of the half of the stack that can run unattended.
#
# It builds and starts a real Archipelago server on a freshly generated seed
# and a real bridge, then makes the calls the SourceMod plugin would make and
# checks what the multiworld did about them. No mocks: the only thing standing
# in for the game is curl.
#
# What it does not cover is everything downstream of those calls, which is the
# plugin itself and the game. That needs a Team Fortress 2 client and a human.
set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
root=$(CDPATH= cd -- "$here/.." && pwd)

# Same wiring as the Makefile: the pins are what the build args read, and the
# paths in the compose file are relative to the repository root.
compose="docker compose --project-directory $root --env-file $root/deploy/env/versions.env -f $here/compose.test.yml"
bridge="http://127.0.0.1:24680"

# ReadyTimeout covers a cold build plus generating a seed.
ready_timeout=300
check_timeout=60

failures=0

log() { printf '\n=== %s\n' "$*"; }
pass() { printf 'ok    %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; failures=$((failures + 1)); }

cleanup() {
	status=$?
	if [ "${KEEP_STACK:-0}" = "1" ]; then
		echo "KEEP_STACK=1, leaving the stack up"
		return
	fi
	log "tearing the stack down"
	$compose down --volumes --remove-orphans >/dev/null 2>&1 || true
	exit "$status"
}
trap cleanup EXIT INT TERM

# json extracts one value from a JSON document on stdin. Bare python rather
# than jq: one fewer thing to install on a machine that is already running
# Docker.
json() {
	python3 -c "import json,sys; print($1)" 2>/dev/null || true
}

waitfor() {
	what=$1
	timeout=$2
	shift 2
	elapsed=0
	while [ "$elapsed" -lt "$timeout" ]; do
		if "$@" >/dev/null 2>&1; then
			pass "$what"
			return 0
		fi
		sleep 2
		elapsed=$((elapsed + 2))
	done
	fail "$what (waited ${timeout}s)"
	return 1
}

bridge_connected() {
	curl -fsS --max-time 5 "$bridge/healthz" | grep -q '"connected":true'
}

archipelago_logged() {
	$compose logs archipelago 2>/dev/null | grep -qF "$1"
}

log "building and starting the stack"
$compose up --detach --build

log "waiting for the bridge to reach Archipelago"
waitfor "the bridge connected to the multiworld" "$ready_timeout" bridge_connected || {
	$compose logs --tail 40
	exit 1
}

health=$(curl -fsS "$bridge/healthz")
mission=$(printf '%s' "$health" | json "json.load(sys.stdin)['missions'][0]")
goal=$(printf '%s' "$health" | json "json.load(sys.stdin)['missions'][-1]")

if [ -n "$mission" ]; then
	pass "the seed has missions, starting with $mission"
else
	fail "the bridge reported no missions"
	exit 1
fi

log "the starting inventory reached the plugin's view"
# The unlock set is keyed by grant kind, so a kind added later needs no change
# on either side of the wire.
unlocks=$(curl -fsS "$bridge/unlocks")
classes=$(printf '%s' "$unlocks" | json "len(json.load(sys.stdin)['unlocks']['class'])")
slots=$(printf '%s' "$unlocks" | json "len(json.load(sys.stdin)['unlocks']['weapon_slot'])")
missions=$(printf '%s' "$unlocks" | json "len(json.load(sys.stdin)['unlocks']['mission_ticket'])")
items=$(curl -fsS "$bridge/healthz" | json "json.load(sys.stdin)['items']")

# The apworld precollects a ticket, at least one class and at least one slot,
# which is the sphere 0 guarantee arriving intact at the other end of the
# stack.
[ "${items:-0}" -ge 3 ] && pass "the run holds $items item(s)" || fail "the run holds ${items:-unset} item(s)"
[ "${classes:-0}" -ge 1 ] && pass "$classes class(es) unlocked" || fail "no classes unlocked"
[ "${slots:-0}" -ge 1 ] && pass "$slots weapon slot(s) unlocked" || fail "no weapon slots unlocked"
[ "${missions:-0}" -ge 1 ] && pass "$missions mission(s) unlocked" || fail "no missions unlocked"

log "reporting a wave clear, twice"
status=$(curl -fsS -o /dev/null -w '%{http_code}' -X POST "$bridge/objective" \
	-d "{\"kind\":\"wave_cleared\",\"popfile\":\"$mission\",\"wave\":1}")
[ "$status" = "204" ] && pass "the bridge took the objective" || fail "the bridge answered $status"

status=$(curl -fsS -o /dev/null -w '%{http_code}' -X POST "$bridge/objective" \
	-d "{\"kind\":\"wave_cleared\",\"popfile\":\"$mission\",\"wave\":1}")
[ "$status" = "204" ] && pass "the retry was taken too" || fail "the retry answered $status"

waitfor "the check reached Archipelago" "$check_timeout" archipelago_logged "Wave 1)"

# A server that comes back from a crash asks where the team was, and the answer
# is the mission of the last wave cleared and that wave. apw-onf.
log "the bridge remembers where the team is"
resume_pop=$(curl -fsS "$bridge/missions" | json "json.load(sys.stdin)['resume']['popfile']")
resume_wave=$(curl -fsS "$bridge/missions" | json "json.load(sys.stdin)['resume']['wave']")
[ "${resume_pop:-x}" = "$mission" ] && [ "${resume_wave:-0}" = "1" ] \
	&& pass "resume says $resume_pop after wave $resume_wave" \
	|| fail "resume says ${resume_pop:-unset} wave ${resume_wave:-unset}, wanted $mission wave 1"

sent=$($compose logs archipelago 2>/dev/null | grep -cF "Wave 1)" || true)
[ "$sent" = "1" ] && pass "Archipelago saw the check once, not twice" \
	|| fail "Archipelago logged the check $sent time(s)"

log "reporting an objective the tables do not have"
status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$bridge/objective" \
	-d '{"kind":"wave_cleared","popfile":"mvm_potato","wave":1}')
[ "$status" = "400" ] && pass "an unknown mission is refused" || fail "an unknown mission answered $status"

log "clearing the goal mission"
status=$(curl -fsS -o /dev/null -w '%{http_code}' -X POST "$bridge/objective" \
	-d "{\"kind\":\"mission_cleared\",\"popfile\":\"$goal\"}")
[ "$status" = "204" ] && pass "the bridge took the mission clear" || fail "the bridge answered $status"

waitfor "Archipelago recorded the goal" "$check_timeout" \
	archipelago_logged "has completed their goal"

log "a player talking to the multiworld"
status=$(curl -fsS -o /dev/null -w '%{http_code}' -X POST "$bridge/say" \
	-d '{"text":"!missing"}')
[ "$status" = "204" ] && pass "the multiworld took the line" || fail "say answered $status"

# The plugin starts from a negative sequence so a server joining late does not
# replay the evening into everyone's chat.
start=$(curl -fsS "$bridge/messages?since=-1" | json "json.load(sys.stdin)['seq']")
[ -n "$start" ] && pass "chat starts from sequence $start" || fail "no chat sequence"

curl -fsS -o /dev/null -X POST "$bridge/say" -d '{"text":"hello from the test"}'
messages=$(curl -fsS "$bridge/messages?since=$start" | json "len(json.load(sys.stdin)['messages'])")
[ "${messages:-0}" -ge 1 ] && pass "the multiworld answered with $messages line(s)" \
	|| fail "the multiworld said nothing back"

log "a plugin whose sequence is ahead is told at once"
ahead=$(curl -fsS "$bridge/grants?since=9999" | json "json.load(sys.stdin)['seq']")
[ -n "$ahead" ] && [ "$ahead" -lt 9999 ] \
	&& pass "the bridge answered with its own sequence, $ahead" \
	|| fail "the bridge left a plugin that is ahead waiting"

log "the multiworld refuses a command that would end the run"
status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$bridge/say" \
	-d '{"text":"!release"}')
[ "$status" = "403" ] && pass "!release cannot be sent from the game" \
	|| fail "!release answered $status"

log "the game disagreeing about a mission's length is reported"
curl -fsS -o /dev/null -X POST "$bridge/objective" \
	-d "{\"kind\":\"wave_cleared\",\"popfile\":\"$mission\",\"wave\":1,\"waves_total\":99}"
drift=$(curl -fsS "$bridge/healthz" | json "json.load(sys.stdin)['wave_drift'][0]['observed']")
[ "${drift:-0}" = "99" ] && pass "the wave count from the game is reported as drift" \
	|| fail "a mission length the tables disagree with went unreported"

log "the cursor the plugin resumes from is the acknowledged one"
resume=$(printf '%s' "$unlocks" | json "json.load(sys.stdin)['resume_from']")
[ "${resume:-x}" = "0" ] && pass "nothing acknowledged yet, so the cursor is 0" \
	|| fail "resume_from is ${resume:-unset} before any acknowledgement"

# Resuming from that cursor has to hand back everything the plugin has not
# applied, including what the unlock set does not carry. A cursor set to the
# length of the item list would step over an unapplied cash bundle and lose it.
resumed=$(curl -fsS "$bridge/grants?since=$resume" | json "len(json.load(sys.stdin)['grants'])")
[ "${resumed:-0}" -ge 1 ] && pass "resuming from $resume returned $resumed grant(s)" \
	|| fail "resuming from $resume returned nothing, so a grant would be lost"

log "an acknowledgement is durable and holds effects back"
acked=$(curl -fsS "$bridge/grants?since=0" | json "json.load(sys.stdin)['seq']")
status=$(curl -fsS -o /dev/null -w '%{http_code}' -X POST "$bridge/grants/ack" \
	-d "{\"seq\":$acked}")
[ "$status" = "204" ] && pass "the bridge took the acknowledgement at $acked" \
	|| fail "the acknowledgement answered $status"

resume=$(curl -fsS "$bridge/unlocks" | json "json.load(sys.stdin)['resume_from']")
[ "${resume:-0}" = "${acked:-x}" ] && pass "the cursor moved to $resume" \
	|| fail "the cursor is ${resume:-unset} after acknowledging $acked"

# A plugin that reloaded resumes from there. State comes back, effects must not.
effects=$(curl -fsS "$bridge/grants?since=$resume" \
	| json "len([g for g in json.load(sys.stdin)['grants'] if g['kind'] == 'credits'])")
[ "${effects:-0}" = "0" ] && pass "no effect was handed out a second time" \
	|| fail "$effects effect(s) came back after being acknowledged"

log "the state survives a restart"
$compose restart bridge >/dev/null
waitfor "the bridge came back and reconnected" "$check_timeout" bridge_connected
after=$(curl -fsS "$bridge/healthz" | json "json.load(sys.stdin)['items']")
kept=$(curl -fsS "$bridge/healthz" | json "json.load(sys.stdin)['acked_seq']")
[ "${kept:-0}" = "${acked:-0}" ] && pass "the acknowledgement survived at $kept" \
	|| fail "the acknowledgement came back as ${kept:-unset}, was $acked"
[ "${after:-0}" -ge "${items:-0}" ] && pass "the run survived with $after item(s)" \
	|| fail "the run came back with ${after:-unset} item(s), had $items"
resume_back=$(curl -fsS "$bridge/missions" | json "json.load(sys.stdin)['resume']['wave']")
[ "${resume_back:-0}" = "1" ] && pass "where the team was survived the restart" \
	|| fail "the resume came back as wave ${resume_back:-unset}"

if [ "$failures" -eq 0 ]; then
	log "all checks passed"
	exit 0
fi
log "$failures check(s) failed"
exit 1
