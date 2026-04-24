# Telegram HTML-formatting patch — instructions

> **For the Linux agent:** this doc is a ready-to-execute patch on top of the already-overlaid
> `claude-telegram-channel-modified` plugin. Read all of §1–§4 before touching anything.

## 1. Context

Zen's Claude agent sends replies to Telegram as raw markdown. Telegram's `Markdown` and even
`MarkdownV2` parse modes don't render:

- `#` / `##` headings (appear as literal `#` chars)
- Tables (show as pipe-ASCII art)
- `-` / `*` bullet lists (dashes become literal dashes)
- Nested lists, `---` horizontal rules, em-dashes around raw text, etc.

The fix: **convert the agent's markdown to a Telegram-safe subset of HTML** before calling
`bot.sendMessage`, and switch `parse_mode: 'Markdown'` → `'HTML'`.

Telegram's HTML mode accepts a small whitelist: `<b>`, `<strong>`, `<i>`, `<em>`, `<u>`,
`<s>`, `<code>`, `<pre>`, `<a href>`, `<blockquote>`, `<blockquote expandable>`,
`<tg-spoiler>`. **Any other tag → 400 Bad Request.**

The converter below lives entirely inside `server.ts` with **zero new dependencies** —
deliberately, because adding deps gets wiped on every `claude plugin update` (see
overlay-drift section of `docs/telegram-extended-install-note.md`).

## 2. Goal

1. Zen's Telegram bot renders agent replies with proper bold / italic / code blocks
   / links / collapsible blockquotes, and readable fallbacks for features Telegram
   doesn't support (headings → bold + newline, tables → `<pre>` block, bullet lists →
   `• ` prefixed lines).
2. No new runtime dependencies.
3. The patched source lives in `~/tools/claude-telegram-channel-modified/server.ts`
   (Zen's canonical overlay copy) so the standard reinstall recipe still works.

## 3. Prerequisites

Zen is **not** at the laptop right now. The Claude session on the Linux box must already
be running for the agent to execute this. If the Claude session is down, surface that to
Zen first — this patch needs an active CLI session to apply + verify.

Expected initial state (verify before changing anything):

```bash
ls ~/tools/claude-telegram-channel-modified/server.ts
ls ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/server.ts
ls ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/server.ts.upstream-backup
```

All three must exist. If any is missing, stop and report — the overlay isn't in the
expected state.

## 4. Steps

### 4.1 Inspect the current `sendMessage` call site

Find where the agent's response is sent to Telegram:

```bash
grep -n 'sendMessage\|parse_mode\|bot\.' ~/tools/claude-telegram-channel-modified/server.ts | head -40
```

Expected: at least one `bot.sendMessage(chatId, text, { parse_mode: 'Markdown' })` call
(or `MarkdownV2`). Note the exact function / line so you can patch all instances.

If there's **no** `sendMessage` call (e.g. the overlay uses a different abstraction like
`sendLong` or `telegramify`), surface that back to Zen — the patch below assumes the
standard `bot.sendMessage(chatId, text, options)` signature.

### 4.2 Add the converter + the whitelist sanitizer

Open `~/tools/claude-telegram-channel-modified/server.ts`. Add the following block
**once**, near the top of the file but after all `import`s. If the file uses CommonJS
(`require`) instead of ESM, skip nothing — this block has no imports.

```ts
// ─────────────────────────────────────────────────────────────
// Telegram HTML formatting helpers.
// Converts a subset of CommonMark to Telegram-safe HTML.
// Source of truth: docs/telegram-html-formatting-patch.md
// ─────────────────────────────────────────────────────────────

/**
 * Escape the four characters Telegram forbids in HTML parse mode when they
 * appear as raw text (not inside our emitted tags).
 */
function escapeHtmlText(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/**
 * Convert CommonMark-ish markdown to Telegram-safe HTML.
 *
 * Supported:
 *   # H1..###### H6       → <b>text</b> with trailing blank line
 *   **bold**, __bold__    → <b>
 *   *italic*, _italic_    → <i>
 *   ~~strike~~            → <s>
 *   `code`                → <code>
 *   ```lang\ncode\n```    → <pre><code class="language-lang">…</code></pre>
 *   [text](url)           → <a href="url">text</a>
 *   > quote               → <blockquote>
 *   >>> long block        → <blockquote expandable>  (optional — see §4.3)
 *   ---                   → single em-dash line
 *   -, *, + lists         → • item
 *   1. 2. 3. lists        → kept as-is
 *   | pipe | tables |     → <pre>…ascii table…</pre>
 *
 * Not supported (agent should avoid in telegram replies):
 *   nested bullet indent, task lists [x], footnotes, images, HTML raw
 */
export function markdownToTelegramHtml(md: string): string {
  // 1. Extract fenced code blocks first; replace with placeholders
  //    so subsequent regexes don't mutilate their content.
  const codeBlocks: string[] = [];
  let out = md.replace(
    /```([a-zA-Z0-9_-]+)?\n([\s\S]*?)```/g,
    (_full, lang: string | undefined, code: string) => {
      const cls = lang ? ` class="language-${lang}"` : '';
      codeBlocks.push(`<pre><code${cls}>${escapeHtmlText(code.replace(/\n$/, ''))}</code></pre>`);
      return `\x00CODEBLOCK${codeBlocks.length - 1}\x00`;
    },
  );

  // 2. Extract inline code with similar placeholder trick.
  const inlineCodes: string[] = [];
  out = out.replace(/`([^`\n]+?)`/g, (_m, code: string) => {
    inlineCodes.push(`<code>${escapeHtmlText(code)}</code>`);
    return `\x01INLINECODE${inlineCodes.length - 1}\x01`;
  });

  // 3. Escape all remaining HTML specials.
  out = escapeHtmlText(out);

  // 4. Headings → bold + blank line
  out = out.replace(/^(#{1,6})\s+(.+?)\s*$/gm, (_m, _h, text: string) => `<b>${text}</b>`);

  // 5. Links BEFORE inline emphasis (text inside is regular — we already
  //    HTML-escaped it, so `<` / `>` are already `&lt;` / `&gt;`).
  //    Match [text](url) where url has no spaces.
  out = out.replace(
    /\[([^\]\n]+)\]\((https?:\/\/[^\s)]+|tg:\/\/[^\s)]+|mailto:[^\s)]+)\)/g,
    (_m, text: string, href: string) => `<a href="${href}">${text}</a>`,
  );

  // 6. Bold BEFORE italic (bold uses two markers, italic uses one —
  //    order matters to avoid **foo** becoming <i><i>foo</i></i>).
  out = out.replace(/\*\*([^*\n]+?)\*\*/g, '<b>$1</b>');
  out = out.replace(/__([^_\n]+?)__/g, '<b>$1</b>');

  // 7. Italic. Lookbehind/ahead on the same marker to avoid stealing
  //    the remaining `*` in a `***triple***`.
  out = out.replace(/(?<![*\w])\*([^*\n]+?)\*(?![*\w])/g, '<i>$1</i>');
  out = out.replace(/(?<![_\w])_([^_\n]+?)_(?![_\w])/g, '<i>$1</i>');

  // 8. Strikethrough ~~text~~
  out = out.replace(/~~([^~\n]+?)~~/g, '<s>$1</s>');

  // 9. Blockquotes: group consecutive `> ` lines into one <blockquote>.
  //    Extra-long ones use `<blockquote expandable>` for collapse — see §4.3.
  out = out.replace(
    /(?:^&gt;\s?.*\n?)+/gm,
    (block) => {
      const body = block
        .split('\n')
        .filter((l) => l.trim() !== '')
        .map((l) => l.replace(/^&gt;\s?/, ''))
        .join('\n');
      const expandable = body.length > 600 ? ' expandable' : '';
      return `<blockquote${expandable}>${body}</blockquote>\n`;
    },
  );

  // 10. Horizontal rule → em-dash line (can't emit <hr> in Telegram HTML mode)
  out = out.replace(/^-{3,}\s*$/gm, '———');

  // 11. Pipe tables → <pre>…as text…</pre>. We detect a run of ≥2 pipe-lines
  //     where the second is the separator row.
  out = out.replace(
    /^\|.+\|\n\|[\s:|-]+\|\n(?:\|.+\|\n?)+/gm,
    (table) => `<pre>${table.trimEnd()}</pre>\n`,
  );

  // 12. Bullet list markers.
  out = out.replace(/^[-*+]\s+/gm, '• ');

  // 13. Restore inline code placeholders.
  out = out.replace(/\x01INLINECODE(\d+)\x01/g, (_m, i: string) => inlineCodes[Number(i)] ?? '');

  // 14. Restore fenced code blocks.
  out = out.replace(/\x00CODEBLOCK(\d+)\x00/g, (_m, i: string) => codeBlocks[Number(i)] ?? '');

  return out;
}

/**
 * Strip every tag NOT in the Telegram HTML whitelist, preserving the inner text.
 * Defensive: catches anything the agent emits that telegramify missed, so we
 * never 400 from sendMessage.
 */
export function sanitizeTelegramHtml(html: string): string {
  const allowed = new Set([
    'b', 'strong', 'i', 'em', 'u', 's', 'strike', 'del',
    'code', 'pre', 'a', 'blockquote', 'tg-spoiler', 'br',
  ]);
  return html.replace(
    /<\/?([a-zA-Z][a-zA-Z0-9-]*)([^>]*)>/g,
    (match, rawName: string, rest: string) => {
      const name = rawName.toLowerCase();
      if (!allowed.has(name)) return '';
      // Allow `expandable` on <blockquote>, `href` on <a>, `class="language-..."` on <code>.
      if (name === 'blockquote') {
        return /expandable/i.test(rest)
          ? `<blockquote expandable>`
          : match.startsWith('</')
            ? `</blockquote>`
            : `<blockquote>`;
      }
      if (name === 'a') {
        const href = /href\s*=\s*"([^"]+)"/i.exec(rest)?.[1];
        return match.startsWith('</')
          ? `</a>`
          : href
            ? `<a href="${href}">`
            : '';
      }
      if (name === 'code') {
        const cls = /class\s*=\s*"([^"]+)"/i.exec(rest)?.[1];
        return match.startsWith('</')
          ? `</code>`
          : cls
            ? `<code class="${cls}">`
            : `<code>`;
      }
      return match.startsWith('</') ? `</${name}>` : `<${name}>`;
    },
  );
}

/** Full pipeline: raw markdown → Telegram-safe HTML. Use this at every send site. */
export function toTelegramHtml(md: string): string {
  return sanitizeTelegramHtml(markdownToTelegramHtml(md));
}
```

### 4.3 Wire it into every `sendMessage` call

For **every** `bot.sendMessage(chatId, text, options)` call in `server.ts`:

**Before:**

```ts
await bot.sendMessage(chatId, text, { parse_mode: 'Markdown' });
```

**After:**

```ts
await bot.sendMessage(chatId, toTelegramHtml(text), {
  parse_mode: 'HTML',
  disable_web_page_preview: true,
});
```

The `disable_web_page_preview: true` option stops Telegram from showing a fat preview
card for every URL in the reply (agent output usually has multiple links; the previews
are visual clutter).

Do the same for `bot.editMessageText` and `bot.answerCallbackQuery` if they take a
`text` + `parse_mode` too. If the current overlay has a helper that wraps
`sendMessage` (search for `sendLong`, `reply`, `chunked`, `send`), patch the helper
instead of each call site — one change beats many.

### 4.4 Telegram 4096-char message limit

Telegram truncates anything over 4096 chars per message. If the overlay doesn't
already chunk long messages, add a helper:

```ts
const TG_MAX = 4000; // keep margin for closing tags we may need to rebalance

/** Split HTML into chunks ≤ TG_MAX, preferring split points between paragraphs,
 *  and never cutting inside a tag. Closing + re-opening `<pre>` / `<blockquote>`
 *  across chunks is intentional — simpler than stitching, still valid. */
export function chunkTelegramHtml(html: string): string[] {
  if (html.length <= TG_MAX) return [html];
  const chunks: string[] = [];
  let remaining = html;
  while (remaining.length > TG_MAX) {
    // Prefer a paragraph break within the last 500 chars of the window.
    const slice = remaining.slice(0, TG_MAX);
    const cut = Math.max(
      slice.lastIndexOf('\n\n'),
      slice.lastIndexOf('</pre>') + 6,
      slice.lastIndexOf('</blockquote>') + 13,
    );
    const split = cut > TG_MAX - 500 ? cut : TG_MAX;
    chunks.push(remaining.slice(0, split));
    remaining = remaining.slice(split).trimStart();
  }
  if (remaining) chunks.push(remaining);
  return chunks;
}
```

Use it like:

```ts
const html = toTelegramHtml(text);
for (const part of chunkTelegramHtml(html)) {
  await bot.sendMessage(chatId, part, {
    parse_mode: 'HTML',
    disable_web_page_preview: true,
  });
}
```

### 4.5 Copy the overlay into place

```bash
cp ~/tools/claude-telegram-channel-modified/server.ts \
   ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/server.ts
```

### 4.6 Restart the Claude session

Plugin MCP servers only reload on `claude` start. In Zen's VS Code terminal-keeper,
close the **claude** terminal — terminal-keeper auto-respawns it with:

```
claude --continue --name gekko --channels plugin:telegram@claude-plugins-official
```

## 5. Verification

Message the bot any of the following; compare before/after. All three should render cleanly.

### 5.1 Mixed-formatting probe

```
/send test

# Heading one

This is *italic* and **bold** and ~~strike~~ and `inline`.

- bullet one
- bullet two

Here's a link: [Anthropic](https://anthropic.com)

> A short quote.
>
> Second line of the quote.

```go
fmt.Println("hello, world")
```
```

Expected visible result:
- **Heading one** as bold on its own line (no `#`)
- *italic*, **bold**, ~~strike~~, `inline` all rendered
- Bullets as `• bullet one` / `• bullet two`
- Quote in a grey left-barred block
- Go code in a monospace block with syntax highlight (Telegram iOS+)

### 5.2 Table probe

```
| col a | col b |
|-------|-------|
| x     | y     |
| aa    | bb    |
```

Expected: rendered inside a `<pre>` block as fixed-width ASCII table — readable,
not mangled pipes.

### 5.3 Long-message probe

Ask the agent for a ~6k-char response. Expected: two cleanly-split messages arrive,
each ≤ 4000 chars, no broken tags at the seam.

### 5.4 If Telegram returns 400 Bad Request

The sanitizer missed something. Check the error message Telegram returned (usually
prints in the plugin logs). Common culprits:

- Stray `<p>` or `<div>` — add to the stripping set (shouldn't happen; our converter
  doesn't emit them).
- Malformed href on `<a>` (e.g. a link like `file:///…`) — Telegram only accepts
  `http://`, `https://`, `tg://`, `mailto:`. Tighten the regex in step 5 of
  `markdownToTelegramHtml`.
- Nested `<code>` inside `<pre><code>` — fine if the inner has no attributes.

## 6. Rollback

If anything goes wrong and the bot stops responding:

```bash
cd ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/
cp server.ts.upstream-backup server.ts
# restart the claude terminal
```

You lose the Xyloforge slash commands (`/model`, `/effort`, etc.) on rollback. If
only the HTML patch is bad but you want to keep the slash commands, rewind to the
pre-patch modified file instead:

```bash
# If you kept the pre-patch copy (recommended before applying §4):
cp ~/tools/claude-telegram-channel-modified/server.ts.prehtml \
   ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/server.ts
```

**Recommended before applying**: save the pre-patch file as
`~/tools/claude-telegram-channel-modified/server.ts.prehtml`.

## 7. Overlay-drift note

This patch lives in `~/tools/claude-telegram-channel-modified/server.ts`. Running
`claude plugin update` (or a bun install that rebuilds the plugin cache) will
overwrite the file in `~/.claude/plugins/cache/…`. To reapply after an update:

```bash
cp ~/tools/claude-telegram-channel-modified/server.ts \
   ~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.6/server.ts
```

Then restart the claude terminal. The `0.0.6` version string may change — check the
directory first.

## 8. Agent prompt companion (optional, cheap)

Even with the converter, the agent is better at writing Telegram-friendly output
when reminded. Append this block to `~/.claude/CLAUDE.md` (user-scope) on the Linux
box:

```md
## Telegram channel output

When responding through `plugin:telegram@claude-plugins-official`:

- Prefer short paragraphs over long ones.
- Use `**bold**` for section headers (will render as bold line).
- Use `-` bullets, not tables, when possible. Tables render as
  monospaced blocks — fine for <5-column data, not for comparisons.
- Keep code blocks to one per message when possible (they don't
  chunk across message splits as gracefully as paragraphs).
- Avoid nested lists / nested emphasis — they degrade.
```

Optional. The converter handles the rendering; this just makes the agent
prefer telegram-friendly output to start with.

## 9. Scope summary

- Touches: `server.ts` in the Xyloforge overlay (one file).
- New deps: **zero**.
- New files: **zero**.
- Behavior: Telegram bot replies render properly. Slash commands unchanged.
- Rollback: one `cp` + restart.

Ping Zen when done with a sample "before/after" screenshot of the probe in §5.1.
