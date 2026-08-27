# Done, kept for the record

- The bots only turn up when the first wave starts. `Bots_Fill` tops RED up
  every three seconds between waves so the bots shop and the engineer builds.
- The bot upgrade chat named the wrong upgrade. `Bots_LoadUpgradeNames` counted
  a commented-out `attribute` line, so 44 of the 63 names were off by one.
