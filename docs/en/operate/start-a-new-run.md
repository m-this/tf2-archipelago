# Start a new run

A session never changes once it exists. A different run needs a new session, a
new room, and one edit in `.env`.

## The four steps

1. Edit `.env` if the shape of the run changes. See
   [The shape of the run](../setup/shape-of-the-run.md).
2. Run `make seed`. It writes another file into `seed/`.
3. Upload that file and create a room. See
   [Create the session](../setup/create-the-session.md).
4. Set `AP_PORT` to the port of the new room, then run `make restart`.

The game files are not touched, so the restart takes seconds.

Keep the old files in `seed/`. Each one is a whole run, and the room of a run
comes back from its file.

## What the bridge does with the old run

The bridge notices that the session is not the one it holds state for. It then:

1. Moves its state file aside to `bridge.<seed>.json`, in the same directory.
2. Starts over with no checks and no unlocks.
3. Tells the plugin that the run restarted, so the plugin drops its own copy
   and asks for the new unlock set.

The old file is never overwritten. If you point the bridge at the wrong room by
accident, the previous run is still on disk in the `tf2-archipelago_bridgestate`
volume.

Nothing else drops a run. Restarting a service, restarting the machine and
stopping for a week all keep it.

## What you do not need to delete

| Volume | Leave it alone |
| --- | --- |
| `tf2-archipelago_tf2game` | 14 GB of game files. Deleting it downloads them again. |
| `tf2-archipelago_bridgestate` | The bridge archives the old run into it by itself. |

## If you host the session yourself

`COMPOSE_PROFILES=selfhost` puts the session in the `tf2-archipelago_apoutput`
volume, and you upload nothing. A new run is three commands:

```sh
make down
docker volume rm tf2-archipelago_apoutput
make up
```

Edit `.env` between the first and the third command. The `archipelago`
container finds an empty output directory, makes a session from the current
`.env`, and hosts it. That takes under a minute.

## Starting completely over

```sh
make clean
```

This stops the stack and deletes every volume, including the game files. It
leaves `seed/` alone. Use it when you finish with the project, not between
runs.
