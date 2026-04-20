# project_gekko

Zenetic Gekkos — gecko breeding business in Phnom Penh. Backend + admin panel + public storefront.

## Status (2026-04-20)

**Clean monorepo skeleton.** Currently contains:

- `.claude/skills/gekko-design/` — shared design skill (shadcn-vue + low-poly aesthetic)
- `docs/` — handoff and admin v1 design spec

The three predecessor repos (`gekko_backend/`, `gekko-admin/`, `zenetic-gekko/`) have been moved **out of this tree** to `../gekko_legacy/` on the developer's Mac. They still exist in their individual GitHub repos as archives. The Go backend rewrite and Vue frontends will be authored fresh into `backend/` and `apps/*` as the project resumes.

See **[docs/HANDOFF-2026-04-20.md](docs/HANDOFF-2026-04-20.md)** for full context.

## Where to start (for Claude Code or a new contributor)

1. Read [docs/HANDOFF-2026-04-20.md](docs/HANDOFF-2026-04-20.md) top-to-bottom.
2. Review the design skill: [.claude/skills/gekko-design/SKILL.md](.claude/skills/gekko-design/SKILL.md) — it governs every UI surface.
3. If on a fresh machine, follow the handoff's **Step 0: Verify environment** before anything else.

## Planned layout (post-migration)

```
project_gekko/
├── .claude/skills/       ← shared Claude Code skills
├── docs/                 ← specs, handoff docs, architecture notes
├── apps/
│   ├── admin/            ← Vue 3 + shadcn-vue admin SPA
│   └── storefront/       ← Vue 3 + Vue I18n public site
├── backend/              ← Go monolith (chi + sqlc + goose + Air)
└── infra/                ← docker-compose, Caddy, deployment configs
```

## Stack

| Layer | Choice |
|---|---|
| Backend | Go (single service, chi + sqlc + goose + zerolog) |
| Database | Postgres 15 |
| Admin | Vue 3 + Vite + shadcn-vue + Tailwind + bun |
| Storefront | Vue 3 + Vite + Vue I18n (currently on Vercel) |
| Package manager | bun |
| Media storage | Filesystem v1 → Cloudflare R2 v2 |
| Reverse proxy (prod) | Caddy |

Rationale for each decision — including rejected alternatives — lives in the handoff doc §3.
