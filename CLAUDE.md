# CLAUDE.md — Guía de colaboración para Claude

Este archivo da contexto persistente a Claude (Code, agentes) sobre cómo trabajamos en este repo.

## Project management — Notion

Toda la planeación, tickets y documentación de producto vive en **Notion**, en el workspace de Fidel.

### Bases de datos clave

| Base | Para qué se usa |
| --- | --- |
| **Backlog Fidel** (`Issues`) | Todos los tickets. Cada uno tiene un ID auto-generado con prefijo `FID-N` (ej. `FID-4`). |
| **Épicas** | Agrupador de tickets. Cada épica tiene Owner, Fecha objetivo, Criterios de aceptación y **Release** asociada. |
| **Sprints** | Ventanas de trabajo. Los Issues se asignan a un Sprint vía relación. |
| **Releases** | Versiones desplegadas. Tanto Issues como Épicas se asignan a un Release. El **MVP es una Release** (`MVP — 2026-05-07`). |
| **Docs Fidel** | Requisitos, diseños, decisiones, guías, actas, riesgos y arquitectura. El **doc maestro 📋 MVP** vive aquí (spec/alcance) y se referencia desde la Release MVP. |

Workspace root: [Fidel](https://www.notion.so/356eec22390a810c8fb3d4af6abbefd2) (dentro de la base **Proyectos**)

### Reglas de trabajo

1. **Una rama por ticket.** El nombre sigue `feat/fid-<n>-<slug>` o `chore/fid-<n>-<slug>`. Nunca se trabaja directo en `main`.
2. **Un PR por ticket.** El título del PR incluye el ID Notion (ej. `FID-1: pushcard migration …`). El cuerpo del PR linkea al ticket y a la épica.
3. **Cada ticket cerrado debe tener su sección "Trabajo realizado"** en la página de Notion: branch, PR, archivos, decisiones de diseño y verificaciones ejecutadas.
4. **CODEOWNERS** rige los reviewers automáticos por área (ver `/CODEOWNERS`).
5. **Convención del título del ticket:** sin prefijo `FID-N` en el título — el ID lo asigna Notion en su columna `ID`.

### Convención de íconos en Notion

Toda página creada en Notion debe llevar el ícono que le corresponde según su tipo. Son emojis básicos (sin color) y se aplican vía `mcp__notion__notion-update-page` con `icon: "<emoji>"` (o al crear la página).

**Tickets — `Issues.Tipo`** (collection `357eec22-390a-818b-a817-000b9964d5bc`)

| Tipo | Ícono |
| --- | --- |
| Historia | 📖 |
| Tarea | ✅ |
| Bug | 🐛 |
| Spike | 🔍 |

**Documentos — `Docs Fidel.Tipo`** (collection `357eec22-390a-814d-ba96-000b7047cdf7`)

| Tipo | Ícono |
| --- | --- |
| Requisito | 📋 |
| Diseño | 🎨 |
| Decisión | 🧭 |
| Guía | 📚 |
| Acta | 📝 |
| Riesgo | ⚠️ |
| Arquitectura | 🏛️ |

**Sistemas de fidelización (sisfi)** — usar el ícono al referenciar el sisfi en páginas, etiquetas o tarjetas:

| Sisfi | Ícono |
| --- | --- |
| earn_burn | 💰 |
| cashback | 💵 |
| pushcard | 🎟️ |

> **Regla:** al crear/actualizar un ticket, doc o página de sisfi en Notion, fija el ícono según esta tabla en la misma operación. Si el `Tipo` cambia, actualiza también el ícono.

### Cómo se conecta Claude a Notion

Claude usa el MCP server `notion` (configurado a nivel de usuario). Operaciones típicas:

- `mcp__notion__notion-search` — buscar páginas, databases, usuarios.
- `mcp__notion__notion-fetch` — leer página o data source completo.
- `mcp__notion__notion-create-pages` — crear tickets, épicas, docs.
- `mcp__notion__notion-update-page` — actualizar contenido o propiedades (ej. cerrar el "Trabajo realizado").
- `mcp__notion__notion-update-data-source` — modificar schema (DDL).

### IDs de referencia

```
Workspace root (Fidel)  : 356eec22-390a-810c-8fb3-d4af6abbefd2
Backlog Fidel           : 357eec22-390a-8039-af27-db3decd1de46
  └─ Issues             : collection://357eec22-390a-818b-a817-000b9964d5bc
  └─ Sprints            : collection://357eec22-390a-81c7-9362-000bd08681d2
  └─ Épicas             : collection://357eec22-390a-81b5-88c8-000b3c9e361f
  └─ Releases           : collection://357eec22-390a-8165-98f8-000bcc71fcde
Docs Fidel              : 357eec22-390a-80ec-b099-c08020c803c3
  └─ Docs               : collection://357eec22-390a-814d-ba96-000b7047cdf7
Release MVP — 2026-05-07: 358eec22-390a-8167-a269-ef00cb1da72b
Doc 📋 MVP              : 357eec22-390a-81b7-b202-cadd4f44d62e
Owner default           : Luis Bolivar — user://586ff6f5-53a5-4f30-8a42-46f594f00ff5
```

### Modelo MVP ↔ tickets

- **Spec/alcance** se escribe en el doc 📋 **MVP** de Docs Fidel.
- **Tracking de entrega** se hace contra la Release **MVP — 2026-05-07** en Releases.
- **Épicas** del MVP llevan la propiedad `Release` apuntando a la Release MVP.
- **Issues** pueden taggear `Release` directo o heredar contexto vía su `Épica`.
- Al cerrar el MVP, la Release pasa de `Planeada` → `En desarrollo` → `En QA` → `Liberada`.

## Convenciones de código

Ver `README.md` para arquitectura y naming. Reglas adicionales:

- Migraciones `golang-migrate` numeradas secuencialmente (`NNNNNN_descripcion.up.sql` / `.down.sql`).
- Nuevos sisfi viven en `internal/modules/<nombre>/` y se registran en `loyalty.Registry` desde `main.go`.
- ER diagram (`docs/erd.mmd`) debe actualizarse cada vez que se agregan/quitan tablas.

## MVP target

**Lanzamiento: jueves 2026-05-07 con despliegue a producción incluido.**
- Spec / alcance: doc [📋 MVP](https://www.notion.so/357eec22390a81b7b202cadd4f44d62e) (Docs Fidel · Requisito).
- Release de entrega: [MVP — 2026-05-07](https://www.notion.so/358eec22390a8167a269ef00cb1da72b) (Backlog Fidel · Releases).
- Épica ancla: [E1 — Módulo Pushcard](https://www.notion.so/357eec22390a81569390c935c0d0aa0d) (ya asociada a la Release MVP).
