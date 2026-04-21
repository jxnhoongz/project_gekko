# Telegram channel (extended) — finish-install note

**Context:** On 2026-04-21 we adopted [Xyloforge/claude-telegram-channel-modified](https://github.com/Xyloforge/claude-telegram-channel-modified) — a supplement to the official `telegram@claude-plugins-official` plugin that adds runtime slash commands (`/model`, `/effort`, `/stats`, `/cost`, `/shell`, `/new`, `/compact`, `/dirs`, `/plugins`, `/logs`, `/console`, `/cmdlist`, `/cmdremove`, `/deny`, `/context`).

The overlay was applied remotely by Facai while Zen was away from the machine. Step 1 of 2 is done; step 2 needs Zen at the keyboard.

## What's already done (step 1)

- Repo cloned to `~/tools/claude-telegram-channel-modified/`
- The modified `server.ts` has been copied over the upstream file at
  `~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/server.ts`
- The original upstream file is preserved at `server.ts.upstream-backup` in the same dir for easy rollback
- Bot token + access allowlist were already set up and untouched

## What you need to do (step 2)

The plugin's MCP server boots **once** when `claude` starts. The new `server.ts` only activates after a restart.

In the VS Code terminal-keeper, close the **claude** terminal. Terminal-keeper will auto-respawn it with the command:
```
claude --continue --name gekko --channels plugin:telegram@claude-plugins-official
```

When that new session comes up, the extended commands are live. Verify by messaging the bot:
```
/stats
```
If you get a reply with model, thinking budget, uptime, message count — it's working.

## Rollback

If anything breaks and you want the upstream plugin back:
```bash
cd ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/
mv server.ts.upstream-backup server.ts
```
Then restart the claude terminal.

## Overlay drift — watch for this

Running `claude plugin update` (or a bun install that rebuilds the plugin cache) will overwrite `server.ts` with the upstream version. To re-apply the overlay later:
```bash
cp ~/tools/claude-telegram-channel-modified/server.ts \
   ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/server.ts
```
Version number (`0.0.6`) may change with plugin updates — check the directory first.

## Caveats worth remembering

- `/shell` exposes host command execution through Telegram. The `access.json` allowlist already restricts the bot to your chat_id (`1674071123`), so only you can trigger it — but treat your phone as a remote-shell terminal accordingly.
- `/console` requires `tmux` (not installed yet — `sudo apt install tmux` if you want it).
- The maintainer is third-party. `server.ts` was reviewed before overlay; re-review on future updates is good hygiene.
