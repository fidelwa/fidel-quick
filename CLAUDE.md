# CLAUDE.md — Guía de colaboración para Claude

Este archivo da contexto persistente a Claude (Code, agentes) sobre cómo trabajamos en este repo.

## Project management — Notion

Toda la planeación, tickets y documentación de producto vive en **Notion**, en el workspace de Fidel.

### Bases de datos clave

| Base | Para qué se usa |
| --- | --- |
| **Backlog Fidel** (`Issues`) | Todos los tickets. Cada uno tiene un ID auto-generado con prefijo `FID-N` (ej. `FID-4`). |
| **Épicas** | Agrupador de tickets. Cada épica tiene Owner, Fecha objetivo y Criterios de aceptación. |
| **Sprints** | Ventanas de trabajo. Los Issues se asignan a un Sprint vía relación. |
| **Releases** | Versiones desplegadas. Los Issues se asignan a un Release vía relación. |
| **Docs Fidel** | Requisitos, diseños, decisiones, guías, actas, riesgos y arquitectura. La página **MVP** vive aquí. |

Workspace root: [Fidel](https://www.notion.so/34aeec22390a8010b025ee69de9061c6)

### Reglas de trabajo

1. **Una rama por ticket.** El nombre sigue `feat/fid-<n>-<slug>` o `chore/fid-<n>-<slug>`. Nunca se trabaja directo en `main`.
2. **Un PR por ticket.** El título del PR incluye el ID Notion (ej. `FID-1: pushcard migration …`). El cuerpo del PR linkea al ticket y a la épica.
3. **Cada ticket cerrado debe tener su sección "Trabajo realizado"** en la página de Notion: branch, PR, archivos, decisiones de diseño y verificaciones ejecutadas.
4. **CODEOWNERS** rige los reviewers automáticos por área (ver `/CODEOWNERS`).
5. **Convención del título del ticket:** sin prefijo `FID-N` en el título — el ID lo asigna Notion en su columna `ID`.

### Cómo se conecta Claude a Notion

Claude usa el MCP server `notion` (configurado a nivel de usuario). Operaciones típicas:

- `mcp__notion__notion-search` — buscar páginas, databases, usuarios.
- `mcp__notion__notion-fetch` — leer página o data source completo.
- `mcp__notion__notion-create-pages` — crear tickets, épicas, docs.
- `mcp__notion__notion-update-page` — actualizar contenido o propiedades (ej. cerrar el "Trabajo realizado").
- `mcp__notion__notion-update-data-source` — modificar schema (DDL).

### IDs de referencia

```
Workspace root  : 34aeec22-390a-8010-b025-ee69de9061c6
Backlog Fidel   : daa31272-b0da-4d34-80cd-510577513beb
  └─ Issues     : collection://d120b165-60b1-4167-ba98-93bc0dc1d74d
  └─ Sprints    : collection://dcbd1790-1a80-401e-bcfb-24848ac3275c
  └─ Épicas     : collection://9ede290d-1ad3-4d65-b728-0e6cbb111582
  └─ Releases   : collection://3f420bd5-2893-4d00-8eb0-cb4e9214777f
Docs Fidel      : 5907635c-ab7f-45c8-9716-376f4de0e65a
  └─ Docs       : collection://c37a1e4a-5754-4853-a14a-058e7e72b04d
Owner default   : Luis Bolivar — user://586ff6f5-53a5-4f30-8a42-46f594f00ff5
```

## Convenciones de código

Ver `README.md` para arquitectura y naming. Reglas adicionales:

- Migraciones `golang-migrate` numeradas secuencialmente (`NNNNNN_descripcion.up.sql` / `.down.sql`).
- Nuevos sisfi viven en `internal/modules/<nombre>/` y se registran en `loyalty.Registry` desde `main.go`.
- ER diagram (`docs/erd.mmd`) debe actualizarse cada vez que se agregan/quitan tablas.
- **Commits**: seguir [Conventional Commits](.github/COMMIT_CONVENTION.md) — formato `tipo(scope): subject` con footer `Refs: Notion FID-N`. Habilita el template con `git config commit.template .gitmessage`.

## MVP target

**Lanzamiento: jueves 2026-05-07 con despliegue a producción incluido.** Detalle en la página [MVP](https://www.notion.so/356eec22390a81338098f7c6bdaad09c) y la épica [E1 — Módulo Pushcard](https://www.notion.so/356eec22390a8160bd6ac7857aab2f4e).
