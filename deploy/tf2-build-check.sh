#!/bin/sh
# Ask Valve which TF2 build is live, and say whether the pins still match it.
#
# The only argument is the TF2 ServerVersion the pins were made for, which is
# TF2_SERVER_VERSION in deploy/env/versions.env. Current build goes to stdout.
#
# Exit codes, because the caller has two different things to do:
#   0   the live build is the one passed in
#   10  the live build is something else, and stdout names it
#   1   nothing could be established
#
# Why ISteamApps/UpToDateCheck and not steamcmd: it is Valve's own answer, it
# needs no credentials, no Steam login and no 300 MB of steamcmd in the runner,
# and there is no app_info cache to go stale underneath it. It also answers in
# the number that matters here. SigMod is keyed to a TF2 ServerVersion and says
# so when it refuses to load ("this SigMod build is for TF2 ServerVersion
# 11068238, and the server is 11076587"); app_info_print and SteamDB report the
# Steam depot buildid for 232250 instead, which is a different number nothing in
# this repository pins against.
#
# The appid is 440, the game, not 232250, the dedicated server tool: 232250 has
# no listed version and UpToDateCheck answers "Couldn't get app info for the app
# specified" for it. version=1 is a version no server ever had, so the answer is
# always "out of date" and always carries required_version.
set -eu

known=${1:?the TF2 ServerVersion the pins were made for is the only argument}

case $known in
'' | *[!0-9]*)
	printf 'tf2-build-check: not a ServerVersion: %s\n' "$known" >&2
	exit 1
	;;
esac

# Bounded: 10s to connect, 20s in total per attempt, three attempts. A watcher
# that hangs is a watcher nobody hears from.
answer=$(curl --silent --show-error --fail \
	--connect-timeout 10 --max-time 20 \
	--retry 3 --retry-delay 5 --retry-all-errors \
	'https://api.steampowered.com/ISteamApps/UpToDateCheck/v1/?appid=440&version=1')

current=$(printf '%s' "$answer" |
	jq -r 'if .response.success == true then (.response.required_version | tostring) else empty end')

case $current in
'' | *[!0-9]* | 0)
	printf 'tf2-build-check: no ServerVersion in Steam'\''s answer: %s\n' "$answer" >&2
	exit 1
	;;
esac

printf '%s\n' "$current"

if [ "$current" = "$known" ]; then
	exit 0
fi

exit 10
