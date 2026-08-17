# Start a new run

The stack generates the randomized session at the first start and then keeps
it. Editing `.env` afterwards changes nothing. To play a different run, delete
the generated session and start again.

## The three commands

```sh
make down
docker volume rm tf2-archipelago_apoutput
make up
```

Edit `.env` between the first and the third command. See
[The shape of the run](../setup/shape-of-the-run.md).

The randomizer container finds an empty output directory, generates a new
session from the current `.env`, and hosts it. That takes under a minute. The
game files are not touched, so the start is quick.

## What the bridge does with the old run

The bridge notices that the session is not the one it was working on. It then:

1. Moves its state file aside to `bridge.<seed>.json`, in the same directory.
2. Starts over with no checks and no unlocks.
3. Tells the plugin that the run restarted, so the plugin drops its own copy
   and asks for the new unlock set.

The old file is never overwritten. If you point the bridge at the wrong
randomizer server by accident, the previous run is still on disk in the
`tf2-archipelago_bridgestate` volume.

Nothing else drops a run. Restarting a service, restarting the machine and
stopping for a week all keep it.

## What you do not need to delete

| Volume | Leave it alone |
| --- | --- |
| `tf2-archipelago_tf2game` | 14 GB of game files. Deleting it downloads them again. |
| `tf2-archipelago_bridgestate` | The bridge archives the old run into it by itself. |

Only `tf2-archipelago_apoutput` holds the session.

## Starting completely over

```sh
make clean
```

This stops the stack and deletes every volume, including the game files. Use it
when you are done with the project, not between runs.
