# Trademarks and artwork

This repository is MIT. See [LICENSE](../LICENSE).

## What ships under other terms

The defender bots are GPL-3.0, and so is
[tf2-mvm-bots-go](https://github.com/m-this/tf2-mvm-bots-go), which carries them
now. Both `tf2ap.exe` and `tf2-defender-bots.zip` carry their compiled plugins,
and that repository is where the source lives.

Up to tf2-mvm-bots-go v0.18.3 the defender plugin also included headers from
[SM_Stock_OfficerSpy](https://github.com/OfficerSpy/SM_Stock_OfficerSpy), a
library its author has published no licence for. They were compiled into
`tf2_defenderbots.smx`, which ships inside `tf2ap.exe`, `tf2ap-linux-amd64` and
`tf2-defender-bots.zip`. v0.18.4 removed the includes, and this build no longer
fetches the library. A release built on v0.18.4 or later carries none of it.
Every release with the defender bots up to and including v1.17.9 was built on
an earlier version and still contains those headers.

Every other project in the bot stack keeps its own terms.
[The bots on your team](../docs/en/play/defender-bots.md) names each one and what it is
for.

## The icon

The launcher's icon is the Team Fortress crosshair, traced from the
[Team Fortress 2 wordmark](https://commons.wikimedia.org/wiki/File:Team-Fortress-2-logo.png)
on Wikimedia Commons. That file is public domain as a work below the threshold
of originality, so tracing it carries no copyright.

The mark is still Valve's trademark. Public domain covers the copyright in the
drawing and says nothing about the trademark in the sign.

## Not affiliated with Valve

This is a fan project. It is not affiliated with, sponsored by, or endorsed by
Valve Corporation. Team Fortress 2 and Mann vs Machine are Valve's.

Players bring their own copy of the game, and the server downloads its files
from Steam with SteamCMD. Nothing here redistributes any part of Team Fortress
2.
