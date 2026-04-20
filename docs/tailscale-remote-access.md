# Remote access via Tailscale

**Purpose**: reach the Gekko dev stack (admin on `:5173`, backend on `:8420`, storefront on `:5174`) from another device or network without exposing anything to the public internet.

## Why Tailscale (vs. alternatives)

| Option | Use case | Verdict |
|---|---|---|
| **Tailscale** | Private mesh VPN — devices see each other by hostname over WireGuard | **Default choice for personal dev.** No public exposure, works through CGNAT, free for <100 devices. |
| Cloudflare Tunnel | Publishes a stable public URL (`*.trycloudflare.com` or custom domain) | Use when sharing with a non-technical person who shouldn't install anything. Pair with Cloudflare Access for Google-login gating. |
| ngrok | One-off demo: `ngrok http 5173` → public URL in seconds | Free tier rotates URL, adds warning page, has bandwidth limits. Fine for a 10-min demo, annoying for anything ongoing. |
| Router port-forward + DDNS | Classic home-server setup | **Avoid.** ISPs in Cambodia typically use CGNAT so it won't work; if it did, you'd publish a dev server to the open internet. |

## Setup — one-time

### 1. Create a tailnet
Sign in at https://login.tailscale.com/start with Google/GitHub/Microsoft. This creates your private network ("tailnet").

### 2. Install on the Linux dev box (Zorin 17 / Ubuntu 22 base)

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up --hostname=gekko-dev
```

The second command prints an auth URL — open it in a browser, approve, done. The `--hostname=gekko-dev` flag overrides the default machine hostname (which is long and ugly) so other devices can reach this box at `http://gekko-dev:<port>`.

Verify:

```bash
tailscale status     # shows your device + any others in the tailnet
tailscale ip -4      # prints the 100.x.y.z address this box got
```

### 3. Install on the remote device

- **macOS**: `brew install tailscale` + `sudo tailscale up`, or install the App Store version (gives you a menu bar icon).
- **iOS / Android**: install the Tailscale app, sign in with the same account, toggle the VPN on.
- **Windows**: installer from https://tailscale.com/download.
- **Another Linux**: same `curl ... | sh` one-liner as above.

### 4. Test connectivity from the remote device

```bash
ping gekko-dev
# or browse to http://gekko-dev:5173 (once dev servers are configured to bind 0.0.0.0)
```

## Make dev servers reachable over Tailscale

By default Vite and most Go `http.ListenAndServe(":port", ...)` calls bind to `127.0.0.1` (or on Vite, just `localhost`). That rejects Tailscale clients. Fix per service:

### Vite (admin, storefront)

In `apps/admin/vite.config.ts` (and the storefront's equivalent):

```ts
export default defineConfig({
  // ...
  server: {
    port: 5173,
    host: '0.0.0.0',           // bind all interfaces (incl. the Tailscale one)
    // optional: restrict to the Tailscale interface specifically
    // host: '100.x.y.z',
  },
});
```

Or without editing the config, pass `--host` on the CLI:

```bash
bun --cwd apps/admin run dev -- --host
```

### Go backend

`backend/cmd/gekko/main.go` uses `http.ListenAndServe(":"+port, r)`. The `:` prefix (no IP) already binds all interfaces, so **no change needed** — Tailscale clients can reach it at `http://gekko-dev:8420` as soon as the server is up.

If you ever want to restrict to Tailscale only (block LAN), bind the Tailscale IP explicitly:

```go
tsIP := "100.x.y.z"   // from `tailscale ip -4`
http.ListenAndServe(tsIP+":"+port, r)
```

## Using it day-to-day

On the remote device, everything behaves as if you were on localhost — just swap the host:

| Local URL | Over Tailscale |
|---|---|
| `http://localhost:5173` (admin) | `http://gekko-dev:5173` |
| `http://localhost:5174` (storefront) | `http://gekko-dev:5174` |
| `http://localhost:8420/health` (backend) | `http://gekko-dev:8420/health` |
| `psql -h localhost -p 5433 ...` | `psql -h gekko-dev -p 5433 ...` |

The admin's `VITE_API_BASE_URL` in `apps/admin/.env.local` points at `http://localhost:8420`. When accessing the admin from a remote device, the browser on that device needs to reach the backend too — easiest fix is to set `VITE_API_BASE_URL=http://gekko-dev:8420` in a shared env, or add a second env file. (Not urgent — `localhost` in the env works fine as long as you're developing locally.)

## Gotchas

- **Magic DNS must be enabled** in the Tailscale admin console (Settings → DNS → "Enable MagicDNS"). It usually is by default, but if `ping gekko-dev` fails while `ping 100.x.y.z` works, that's the cause.
- **CORS**: the backend's `CORS_ORIGINS` in `backend/.env.local` lists `http://localhost:5173`. If you hit the admin at `http://gekko-dev:5173` and the browser calls the backend at a different origin, CORS will block it. Add `http://gekko-dev:5173` (and `:5174`) to `CORS_ORIGINS` when working remotely.
- **Sleep**: the Linux dev box must be awake for remote access to work. Consider disabling suspend-on-lid-close if you want to close the laptop but keep services running.
- **Leaving the tailnet**: `sudo tailscale down` disconnects without uninstalling. `sudo tailscale logout` removes the device from your account.
