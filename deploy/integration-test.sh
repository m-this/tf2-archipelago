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
compose="docker compose -f $here/compose.test.yml"
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
unlocks=$(curl -fsS "$bridge/unlocks")
seq=$(printf '%s' "$unlocks" | json "json.load(sys.stdin)['seq']")
classes=$(printf '%s' "$unlocks" | json "len(json.load(sys.stdin)['classes'])")
slots=$(printf '%s' "$unlocks" | json "len(json.load(sys.stdin)['slots'])")
missions=$(printf '%s' "$unlocks" | json "len(json.load(sys.stdin)['missions'])")

# The apworld precollects a ticket, at least one class and at least one slot,
# which is the sphere 0 guarantee arriving intact at the other end of the
# stack.
[ "${seq:-0}" -ge 3 ] && pass "unlock sequence is $seq" || fail "unlock sequence is ${seq:-unset}"
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

log "the state survives a restart"
$compose restart bridge >/dev/null
waitfor "the bridge came back and reconnected" "$check_timeout" bridge_connected
after=$(curl -fsS "$bridge/unlocks" | json "json.load(sys.stdin)['seq']")
[ "${after:-0}" -ge "${seq:-0}" ] && pass "the unlock set survived at sequence $after" \
	|| fail "the unlock set came back as ${after:-unset}, was $seq"

if [ "$failures" -eq 0 ]; then
	log "all checks passed"
	exit 0
fi
log "$failures check(s) failed"
exit 1
