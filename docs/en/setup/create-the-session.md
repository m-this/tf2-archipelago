# Create the session

The randomized session runs on `archipelago.gg`. Your machine makes the session
and writes it to a file. The website takes that file, hosts it, and gives you an
address.

Your machine makes it because Mann vs Machine is not one of the games that come
with Archipelago. The website generates only those games. It hosts any session
that somebody sends it.

## 1. Set the shape of the run

[The shape of the run](shape-of-the-run.md) holds the length, the difficulty and
the goal of an evening. Set those values in `.env` now. The session keeps them,
and a later change needs a new session.

## 2. Make the session

```sh
make seed
```

The first run builds the randomizer image and takes a few minutes. A later run
takes under a minute.

The command writes one file and prints the name of it:

```
generated /ap/output/AP_53174869021847362095.zip
upload it at https://archipelago.gg/uploads, then create a room
```

On your machine that file is in `seed/`. Git ignores the directory. Keep the
files in it. The same file gives the same session again, so a room that you lose
comes back from it.

## 3. Send it to archipelago.gg

1. Open [archipelago.gg/uploads](https://archipelago.gg/uploads).
2. Upload the file from `seed/`.
3. Select **Create New Room**.

The website asks for no account.

The room page holds the address of the room and a link to the tracker. Send that
page to your players. They watch the run from it and they install nothing.

## 4. Point the stack at the room

The room page gives an address in the form `archipelago.gg:12345`. Write the two
halves of it into `.env`:

```sh
AP_HOST=archipelago.gg
AP_PORT=12345
AP_TLS=true
```

Every new room takes a new port, so set `AP_PORT` again after every new room.

Anybody who has the address of a room reaches that room. Set a password on the
room, then put the same password in `AP_PASSWORD`.

Now start the stack. See [Install](install.md).

## The version of Archipelago

`deploy/env/versions.env` pins the version that makes the file.
`archipelago.gg` runs a version of its own, and it refuses a file that this
version cannot read.

If the upload fails, read the version in the footer of the website. Then set
`ARCHIPELAGO_VERSION` to that version and run `make seed` again.

## Host the session yourself

The stack hosts the session itself as well. This needs four lines in `.env`:

```sh
COMPOSE_PROFILES=selfhost
AP_HOST=archipelago
AP_PORT=38281
AP_TLS=false
```

`make up` then starts a third container, makes the session at the first start,
and hosts it beside the game server. You upload nothing, and steps 2 to 4 above
do not apply.

What it costs:

- Your players get no room page, so they get no tracker.
- One more container runs on your machine.
- A player in another game needs a second public port. `deploy/compose.yml` says
  where.
