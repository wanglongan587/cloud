# squads: Squad management entry

[中文](README.md)

Real squad management reads lists, responsibilities, maintainers, leaders and archival history from current-workspace REST endpoints. Existing people, outside newcomers and Space Agents have separate entry points; tenant membership remains authoritative. Pages never use demo fallback or fabricate counts, presence or execution capability. A configured leader does not enable automatic execution.

| File | Responsibility |
| --- | --- |
| `api.ts` | Generated clients, pagination, versioned commands and query invalidation |
| `squads-page.tsx`, `squad-detail-page.tsx` | Real lists, empty/error/archive states and current authority |
| `squad-editor.tsx`, `squad-members.tsx` | Descriptive forms, maintenance, retained history and explicit leader clearing |
| `squad-add.tsx`, `squad-invites.tsx` | Existing people/Agent selectors, administrator targeted invitations and directory entry |
| `translations.ts`, `*.test.tsx` | Bilingual copy and network, creation, archive and authorization regression |

Depends on spaces for current tenancy, members for admission forms, auth for deployment mode, and UI primitives; consumed by routes. The backend owns authority and transactions. Commands express explicit intent and conflicts refetch. Membership/plugin events, reconnect and focus refresh real state. Only administrators mount external invitation controls. Archived pages retain history and expose no mutations. MSW tests use real endpoint shapes without demo fallback.
