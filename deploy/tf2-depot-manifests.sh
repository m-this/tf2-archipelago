#!/bin/sh
# The manifest Steam serves now for each depot of the TF2 dedicated server
# (app 232250), one "<depot> <manifest>" line per depot on stdout.
#
# 232250 is the content shared by both platforms, 232255 the Windows binaries
# and 232256 the Linux ones. A TF2 update that leaves 232255 alone leaves
# server.dll alone, which is what decides whether SigMod for Windows only needs
# a new build key; the same goes for 232256 and the Linux binaries.
#
# DepotDownloader is the tool m-this/sigsegv-mvm-win's Address table job uses to
# read them. It logs in anonymously and, with -manifest-only, downloads no game
# file. Pinned by tag and by the checksum of the archive: a download that does
# not match stops here.
set -eu

depot_downloader_version=3.4.0
depot_downloader_sha256=a999dec66b4850fc961bd50366696d23c2d0fad7b18790e6a5647b2f19097a53

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

curl --silent --show-error --fail --location \
	--connect-timeout 10 --max-time 120 --retry 3 --retry-delay 5 --retry-all-errors \
	--output "$work/dd.zip" \
	"https://github.com/SteamRE/DepotDownloader/releases/download/DepotDownloader_${depot_downloader_version}/DepotDownloader-linux-x64.zip"
printf '%s  %s\n' "$depot_downloader_sha256" "$work/dd.zip" | sha256sum --check --status
unzip -q "$work/dd.zip" -d "$work/bin"
chmod +x "$work/bin/DepotDownloader"

for depot in 232250 232255 232256; do
	manifest=
	for _ in 1 2 3; do
		if timeout 120 "$work/bin/DepotDownloader" -app 232250 -depot "$depot" -manifest-only \
			-dir "$work/out-$depot" >"$work/log-$depot" 2>&1; then
			manifest=$(sed -n "s/^Got manifest request code for depot $depot from app 232250, manifest \([0-9][0-9]*\),.*/\1/p" \
				"$work/log-$depot" | tail -n 1)
			[ -n "$manifest" ] && break
		fi
		sleep 10
	done
	if [ -z "$manifest" ]; then
		cat "$work/log-$depot" >&2
		printf 'tf2-depot-manifests: no manifest for depot %s after 3 attempts\n' "$depot" >&2
		exit 1
	fi
	printf '%s %s\n' "$depot" "$manifest"
done
