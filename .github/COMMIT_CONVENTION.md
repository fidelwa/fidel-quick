# Convención de commits — Fidel Quick

Adoptamos **[Conventional Commits 1.0](https://www.conventionalcommits.org/)** como formato estándar para mensajes. Permite parsing automático para changelogs, alineación con Notion (cada commit referencia su `FID-N`), y revisión más rápida en GitHub.

## Estructura

```
<tipo>[(scope opcional)][!]: <subject corto>

[body opcional explicando el por qué]

[footer opcional con refs y co-authors]
```

## Tipos permitidos

| Tipo | Cuándo usarlo | Ejemplo |
|---|---|---|
| `feat` | Nueva funcionalidad visible al usuario | `feat(auth): vincular Google a cuenta existente` |
| `fix` | Corrección de bug visible al usuario | `fix(wizard): step 2 vacío en flujo pushcard-only` |
| `chore` | Tareas internas que no tocan la app (deps, config) | `chore: bump go.mod a 1.25` |
| `refactor` | Cambio de código sin alterar comportamiento | `refactor(repo): extraer scanAdmin helper` |
| `docs` | Solo documentación | `docs: agregar runbook deploy GCP` |
| `style` | Formato/whitespace, sin cambio funcional | `style: gofmt` |
| `test` | Agregar o ajustar tests | `test(verifier): cubrir rotación de JWKS` |
| `perf` | Mejora de performance | `perf(redis): cachear flags 60s` |
| `build` | Sistema de build, Dockerfile, Makefile | `build: multistage con admin embed` |
| `ci` | CI/CD pipelines | `ci: deploy a Cloud Run en push a main` |
| `revert` | Revertir un commit anterior | `revert: "feat(...)"` |

## Scope (opcional pero recomendado)

Identifica el área del repo afectada. Convenciones del proyecto:

- **Backend Go**: `auth`, `whatsapp`, `wizard`, `pushcard`, `cashback`, `earnburn`, `flow`, `db`, `storage`, `webhook`, `config`.
- **Frontend**: `admin`, `dashboard`, `login`, `onboarding`, `profile`, `programs`.
- **Infra**: `terraform`, `docker`, `gcp`, `gitignore`.
- **Cross-cutting**: `design`, `ci`, `release`.

## Breaking changes

Agrega `!` antes de `:` y un footer `BREAKING CHANGE:`:

```
feat(auth)!: cambiar formato de JWT claim "role"

BREAKING CHANGE: el claim "role" pasa de string a array. Tokens
emitidos antes del 2026-05-08 quedarán inválidos al rotar el secret.
```

## Vincular con Notion

Cada commit debe referenciar el ticket Notion en el footer:

```
Refs: Notion FID-N <url-completa>
```

Ejemplo completo:

```
feat(webhook): validar X-Hub-Signature-256

Antes el POST /webhook aceptaba cualquier JSON. Si la URL pública se
filtraba, era trivial inyectar mensajes falsos. Ahora validamos HMAC
SHA256 del body con WHATSAPP_APP_SECRET (Meta App Settings → Basic).

- Nueva env var WHATSAPP_APP_SECRET (vacío = skip en dev).
- Receive() lee body raw, calcula HMAC, compara con header.
- Firma inválida → 401 sin procesar.

Refs: Notion FID-25 https://www.notion.so/359eec22390a81f78ca7e53f9f6c8b31
Plan: ~/.claude/plans/nifty-drifting-toast.md (Fase 0 · FID-N3).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

## Subject (primera línea)

- Imperativo en español o inglés (uno consistente, el repo usa **español**): "agregar", "corregir", "renombrar" — no "agrega" ni "agregando".
- ≤ 72 caracteres incluyendo `tipo(scope):`.
- Sin punto final.
- Lo primero que un reviewer lee — debe explicar qué + dónde sin abrir el diff.

## Body (opcional pero útil)

- Separado del subject por **una línea en blanco**.
- Explica el **por qué**, no el qué (el diff ya muestra el qué).
- Wrap a ~72 columnas.
- Si listas cambios, usa bullets con `-`.

## Footer

- `Refs: Notion FID-N <url>` — siempre que aplique.
- `Plan: <path>` — si el commit ejecuta un paso de un plan en `.claude/plans/`.
- `Closes #<num>` — para issues o PRs.
- `BREAKING CHANGE: <descripción>` — para cambios que rompen compatibilidad.
- `Co-Authored-By: <nombre> <email>` — pair programming, AI assistance.

## Habilitar el template localmente

```bash
git config commit.template .gitmessage
```

Después, `git commit` (sin `-m`) abrirá tu editor con la estructura pre-poblada.

## Anti-patterns a evitar

❌ `Update files` — sin tipo, sin contexto.
❌ `Fix bug` — sin scope, sin descripción.
❌ `feat:` solo — falta el subject.
❌ `feat: (FID-1) ...` — el FID va en el footer, no el subject.
❌ Body de 600 caracteres en una línea — wrap a 72.
❌ Mezclar 2 features en un commit — separa en 2 commits.

## Excepciones

- **Merge commits** generados por GitHub: dejar el formato default `Merge pull request #N from ...`.
- **Revert commits**: `git revert` autogenera el formato; está bien.
- **Tooling automático** (Dependabot, Renovate): su formato propio se respeta.
