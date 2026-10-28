# Squads and targeted joining

English | [中文](squads.md)

Squads reference workspace members and Space Agents without duplicating membership or granting project, model, Git or runtime access. P1 manages grouping only: it registers no executable Team target and configuring a leader starts no run. The governing specs are the [Squad root decision](https://github.com/ora-space/specs/blob/main/decisions/cloud/squad/0-workspace-scoped-human-agent-squads.md) and [targeted joining and issuer revocation](https://github.com/ora-space/specs/blob/main/decisions/cloud/tenancy/20261027-squad-targeted-joining-and-inviter-revocation.md); ADR lifecycle and direct evidence remain maintained in specs.

## Maintenance and history

Active workspace members may create and read squads; creation atomically grants the creator its first maintainer relationship. Only current administrators or active human maintainers may edit, add existing members/agents, maintain responsibilities, archive or configure leaders. Creation gives no permanent privilege; administrators can take over after all maintainers leave. Writes carry the displayed squad version and conflicts require a fresh read.

Relationships distinguish user/agent types and stable identities. An object may join several squads but has at most one active relationship in each. Ending a relationship preserves its source, time, responsibility and maintainer facts. Rejoining creates a new grant rather than reviving old authority. Archival is irreversible, closes all relationships and the leader, and retains history behind current tenant membership checks.

A leader must be an active Space Agent in the workspace; setting it atomically adds any missing relationship. Removing the current leader requires explicitly clearing that configuration; replacing it leaves the former agent as an ordinary member. Agent retirement closes all active relationships and clears leaders through transactional database triggers; reinstalling does not restore them. Global user disable/deletion and tenant membership revocation close the affected human relationships in the same transaction without altering admitted runs.

## Invitations, approvals and provenance

Only administrators issue external invitations or approve requests. Issuance canonicalizes, deduplicates and freezes up to 20 active same-tenant squad targets. Redemption reads only that server-owned intent. Ordinary invitations have no targets.

Anonymous preview reveals only kind and state; issuer or target failure becomes unavailable. Only an authenticated active identity receives workspace, inviter, target names and its own state, and submission rechecks within the original transaction. Existing workspace members who lack squad relationships must still explicitly accept. Membership creation/restoration, every target grant, consumption or approval commit together; an invalid target cannot produce partial admission.

If neither workspace qualification nor grouping changes, no token is consumed and no join notice or provenance is added. Grouping-only acceptance consumes the invitation but records only squad provenance. Invitations, approvals and corporate directory admission share this rule; workspace provenance is appended only for actual qualification transitions. Requests freeze targets; one pending request is allowed per user/tenant, identical targets reuse it and different targets conflict. Issuer revocation stops new applications while a current administrator may still independently decide a previously submitted request.

Administrator demotion, membership loss and global user disable/deletion permanently set invalidated_at on usable links; restoring authority cannot clear it. Squad archival likewise terminates links targeting that squad. A failed last-administrator check rolls back the transaction and invalidates no links.

## Upgrade and rollback boundary

Stop old writers for this upgrade; mixed-version rolling Cloud deployment is unsupported:

1. Back up and rehearse on an isolated copy. Pause joining and membership operations at ingress, stop every old Cloud instance and its automatic restart/scaling sources, and verify no old instance can still access the database.
2. Run `cloudctl migrate` with the candidate binary, retaining old migration checksums and real data. Keep ingress closed on migration or validation failure.
3. Upgrade all Cloud, Gateway and frontend instances, start new Cloud and pass `CheckSchema`. An old binary rejects the unknown migration on startup; that check does not fence an already-running process, so step 1 is mandatory.
4. Verify tenant isolation, minimal preview and controlled admission before reopening ingress together. On failure keep writes stopped; never restart an old binary to bypass the new policy. Evaluate and preserve post-upgrade writes before any backup restoration.

Old links and pending requests receive empty targets. Currently authorized issuers retain original digests, deadlines and state; usable links from unauthorized issuers become permanently unavailable. No pre-upgrade revoke/restore history is guessed. `join_policy_activation` records activation once; migration creates no demo squads or historical workspace provenance. Number migrations after main at merge time.

REST owns list/detail state. Post-commit membership/plugin notices invalidate squad views and selectors; reconnect, focus and workspace switching revalidate. SSE never grants authority. Unredeemed invitations do not invent recipient names and pending requests do not count as members.
