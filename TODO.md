# TODO

What is still open. The code, the docs and the git history say what works.

## Not implemented

- [ ] DeathLink. The option is in the player file and the bridge reads the
      bounces, and the plugin does nothing with them yet. Individual death is
      noise in MvM; the sensible trigger is a wave failure, or a death outside
      the respawn grace window. Decide, then wire it.
- [ ] Shop checks. They need a plugin that adds a purchasable entry to the
      upgrade station UI. Nothing here knows whether that is possible.

## Worth watching

- [ ] A TF2 update that moves a signature CBaseNPC detours takes the defender
      bots down until a new release ships. `make bots-from-source` and
      `deploy/patches/` are the way to fix it without waiting.
- [ ] Every wave count comes from the wiki, and the game is the authority. The
      bridge reports a disagreement as `wave_drift`; a report is a table row to
      correct in `gamedata/missions.go`.

## Community

- [ ] Ask the Discord thread for Snolid Ice's MvM Manual. Mentioned twice,
      never linked.
- [ ] Tell Damonj17 and Roseburst this exists. The design is theirs.
