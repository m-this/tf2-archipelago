# Fast map downloads with Tailscale

FastDL lets Team Fortress 2 download community maps over HTTPS instead of
squeezing them through the game-server connection. Tailscale Funnel can publish
those files without a webserver to install or a router port to forward. Only
the server PC runs Tailscale; players use an ordinary public web address.

This setting changes only map downloads. Keep choosing **local network**,
**Steam relay**, or **forwarded port** for the game server exactly as before.

## Set up the host

1. [Install Tailscale for Windows](https://tailscale.com/download/windows).
2. Open its system-tray icon, choose **Log in**, and finish signing in through
   the browser. The first account creates a private Tailscale network called a
   tailnet.
3. Open the launcher's **Settings**, then **Networking**.
4. Click **Set up / check Tailscale Funnel**. If the browser opens Tailscale's
   approval page, approve Funnel and click the check button again. It should
   say that Funnel is ready. This is a one-time check, not something to repeat
   before each server start.
5. Turn on **Publish downloads with Tailscale Funnel** and save.
6. Press **Start**. Look for `public Tailscale Funnel FastDL ready` in the
   launcher log.

Open the HTTPS address printed by the launcher. A working endpoint says
`TF2 Archipelago FastDL is ready.` It does not list the downloadable files.

The launcher serves its small allowlisted FastDL endpoint on this machine's
loopback address, then asks Tailscale Funnel to publish that endpoint. It keeps
the resulting address, such as
`https://host-name.example-tailnet.ts.net/tf`, in `sv_downloadurl`. Tailscale
keeps Funnel running in the background.

The launcher remembers the checkbox across restarts. Every **Start** verifies
Tailscale and reapplies the same background Funnel route, so restarting the
launcher, Windows, or the game server needs no setup click.

The first Funnel setup needs web approval. Directly giving Windows Tailscale a
folder would require administrator access, so the launcher does not do that:
Funnel proxies the loopback web endpoint instead. No launcher elevation should
be necessary after the tailnet owner approves Funnel.

## What friends do

Nothing Tailscale-specific. They join the TF2 server using the normal address
or Steam relay printed by the launcher. When TF2 needs a map, it follows the
public HTTPS FastDL address supplied by the game server.

Funnel makes the allowed asset files public to anyone who knows or discovers
the address. It does not publish `cfg`, plugins, or passwords. Funnel is a beta
Tailscale feature and applies bandwidth limits, so test it with the largest map
and expected player count before relying on it for an event.

## If Tailscale needs attention

When this setting is enabled, the launcher will not start the game server
without the selected FastDL. If Funnel approval expired, the Windows launcher
shows a message and opens the approval page. If Tailscale is signed out or not
running, it tells you to restore Tailscale first. Fix the problem and press
**Start** again. This prevents a server intended to use FastDL from silently
starting with slow or unreliable direct map downloads.

The launcher exposes only `maps`, `materials`, `models`, `sound`, `particles`,
and `resource`. It never exposes `cfg`, SourceMod plugins, passwords, or the
rest of the server installation.
