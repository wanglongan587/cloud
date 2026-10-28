-- Deployment must stop every old Cloud writer before applying this migration.
-- Startup schema validation prevents an old binary from restarting afterwards.
ALTER TABLE space_agents ADD CONSTRAINT space_agent_tenant_identity UNIQUE(id,tenant_id);
CREATE TABLE squads (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 space_id uuid NOT NULL,
 name text NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
 description text NOT NULL DEFAULT '' CHECK (length(description)<=2000),
 instructions text NOT NULL DEFAULT '' CHECK (length(instructions)<=8000),
 leader_agent_id uuid REFERENCES space_agents(id),
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 archived_at timestamptz,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 FOREIGN KEY(space_id,tenant_id) REFERENCES collab_workspaces(id,tenant_id),
 FOREIGN KEY(leader_agent_id,tenant_id) REFERENCES space_agents(id,tenant_id),
 UNIQUE(id,tenant_id)
);
CREATE INDEX squads_tenant ON squads(tenant_id,id);
CREATE TABLE squad_members (
 id uuid PRIMARY KEY,
 squad_id uuid NOT NULL,
 tenant_id uuid NOT NULL,
 kind text NOT NULL CHECK(kind IN ('user','agent')),
 user_id uuid REFERENCES users(id),
 agent_id uuid REFERENCES space_agents(id),
 responsibility text NOT NULL DEFAULT '' CHECK(length(responsibility)<=2000),
 maintainer boolean NOT NULL DEFAULT false,
 joined_by uuid NOT NULL REFERENCES users(id),
 source_kind text NOT NULL CHECK(source_kind IN ('created','manual','invitation','application','directory','leader')),
 source_id uuid,
 joined_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 ended_at timestamptz,
 ended_reason text,
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 FOREIGN KEY(squad_id,tenant_id) REFERENCES squads(id,tenant_id),
 FOREIGN KEY(tenant_id,user_id) REFERENCES tenant_memberships(tenant_id,user_id),
 FOREIGN KEY(agent_id,tenant_id) REFERENCES space_agents(id,tenant_id),
 CHECK((kind='user' AND user_id IS NOT NULL AND agent_id IS NULL) OR
       (kind='agent' AND agent_id IS NOT NULL AND user_id IS NULL AND NOT maintainer)),
 CHECK((ended_at IS NULL)=(ended_reason IS NULL))
);
CREATE UNIQUE INDEX squad_active_user ON squad_members(squad_id,user_id) WHERE ended_at IS NULL AND kind='user';
CREATE UNIQUE INDEX squad_active_agent ON squad_members(squad_id,agent_id) WHERE ended_at IS NULL AND kind='agent';
CREATE INDEX squad_member_history ON squad_members(squad_id,id);
CREATE TABLE squad_changes (
 id uuid PRIMARY KEY,
 squad_id uuid NOT NULL REFERENCES squads(id),
 actor_user_id uuid REFERENCES users(id),
 reason text NOT NULL,
 details jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

ALTER TABLE tenant_invitations ADD COLUMN target_squad_ids jsonb NOT NULL DEFAULT '[]'
 CHECK(jsonb_typeof(target_squad_ids)='array' AND jsonb_array_length(target_squad_ids)<=20);
ALTER TABLE tenant_join_links ADD COLUMN target_squad_ids jsonb NOT NULL DEFAULT '[]'
 CHECK(jsonb_typeof(target_squad_ids)='array' AND jsonb_array_length(target_squad_ids)<=20);
ALTER TABLE tenant_join_requests ADD COLUMN target_squad_ids jsonb NOT NULL DEFAULT '[]'
 CHECK(jsonb_typeof(target_squad_ids)='array' AND jsonb_array_length(target_squad_ids)<=20);
ALTER TABLE tenant_invitations ADD COLUMN invalidated_at timestamptz;
ALTER TABLE tenant_join_links ADD COLUMN invalidated_at timestamptz;

-- Invalidation is irreversible and distinct from explicit revocation so anonymous
-- preview can collapse loss of issuer authority to unavailable, without details.
CREATE FUNCTION preserve_join_intent() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.target_squad_ids IS DISTINCT FROM OLD.target_squad_ids THEN
  RAISE EXCEPTION 'join targets are immutable' USING ERRCODE='23514';
 END IF;
 IF TG_TABLE_NAME <> 'tenant_join_requests' THEN
  IF OLD.invalidated_at IS NOT NULL THEN
   NEW.invalidated_at := OLD.invalidated_at;
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER invitation_frozen_intent BEFORE UPDATE ON tenant_invitations FOR EACH ROW EXECUTE FUNCTION preserve_join_intent();
CREATE TRIGGER link_frozen_intent BEFORE UPDATE ON tenant_join_links FOR EACH ROW EXECUTE FUNCTION preserve_join_intent();
CREATE TRIGGER request_frozen_intent BEFORE UPDATE ON tenant_join_requests FOR EACH ROW EXECUTE FUNCTION preserve_join_intent();

-- Run for all mutation paths, including global account administration. Ended
-- relationships retain their original maintainer/source facts and never reopen.
CREATE FUNCTION close_revoked_membership() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.role='admin' AND OLD.status='active' AND (NEW.role<>'admin' OR NEW.status<>'active') THEN
  UPDATE tenant_invitations SET invalidated_at=clock_timestamp(),version=version+1
   WHERE tenant_id=NEW.tenant_id AND created_by=NEW.user_id AND consumed_at IS NULL AND revoked_at IS NULL AND invalidated_at IS NULL;
  UPDATE tenant_join_links SET invalidated_at=clock_timestamp(),version=version+1
   WHERE tenant_id=NEW.tenant_id AND created_by=NEW.user_id AND revoked_at IS NULL AND invalidated_at IS NULL;
 END IF;
 IF OLD.status='active' AND NEW.status<>'active' THEN
  UPDATE squads SET version=version+1,updated_at=clock_timestamp() WHERE id IN
   (SELECT squad_id FROM squad_members WHERE tenant_id=NEW.tenant_id AND user_id=NEW.user_id AND ended_at IS NULL);
  UPDATE squad_members SET ended_at=clock_timestamp(),ended_reason='membership_revoked',version=version+1
   WHERE tenant_id=NEW.tenant_id AND user_id=NEW.user_id AND ended_at IS NULL;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER membership_closes_squads AFTER UPDATE ON tenant_memberships FOR EACH ROW EXECUTE FUNCTION close_revoked_membership();

CREATE FUNCTION close_disabled_user() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (NEW.status<>'active' OR NEW.deleted_at IS NOT NULL) AND OLD.status='active' AND OLD.deleted_at IS NULL THEN
  UPDATE squads SET version=version+1,updated_at=clock_timestamp() WHERE id IN
   (SELECT squad_id FROM squad_members WHERE user_id=NEW.id AND ended_at IS NULL);
  UPDATE tenant_invitations SET invalidated_at=clock_timestamp(),version=version+1
   WHERE created_by=NEW.id AND consumed_at IS NULL AND revoked_at IS NULL AND invalidated_at IS NULL;
  UPDATE tenant_join_links SET invalidated_at=clock_timestamp(),version=version+1
   WHERE created_by=NEW.id AND revoked_at IS NULL AND invalidated_at IS NULL;
  UPDATE squad_members SET ended_at=clock_timestamp(),ended_reason='user_disabled',version=version+1
   WHERE user_id=NEW.id AND ended_at IS NULL;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER user_closes_squads AFTER UPDATE ON users FOR EACH ROW EXECUTE FUNCTION close_disabled_user();

CREATE FUNCTION close_retired_squad_agent() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.status='active' AND NEW.status<>'active' THEN
  UPDATE squads SET version=version+1,updated_at=clock_timestamp() WHERE id IN
   (SELECT squad_id FROM squad_members WHERE agent_id=NEW.id AND ended_at IS NULL);
  INSERT INTO squad_changes(id,squad_id,reason,details)
   SELECT gen_random_uuid(),id,'leader_retired',jsonb_build_object('agentId',NEW.id)
   FROM squads WHERE leader_agent_id=NEW.id;
  UPDATE squads SET leader_agent_id=NULL,version=version+1,updated_at=clock_timestamp() WHERE leader_agent_id=NEW.id;
  UPDATE squad_members SET ended_at=clock_timestamp(),ended_reason='agent_retired',version=version+1
   WHERE agent_id=NEW.id AND ended_at IS NULL;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER agent_closes_squads AFTER UPDATE ON space_agents FOR EACH ROW EXECUTE FUNCTION close_retired_squad_agent();

CREATE FUNCTION close_archived_squad() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.archived_at IS NULL AND NEW.archived_at IS NOT NULL THEN
  NEW.leader_agent_id := NULL;
  UPDATE squad_members SET ended_at=clock_timestamp(),ended_reason='squad_archived',version=version+1 WHERE squad_id=NEW.id AND ended_at IS NULL;
  UPDATE tenant_invitations SET invalidated_at=clock_timestamp(),version=version+1
   WHERE target_squad_ids @> jsonb_build_array(NEW.id::text) AND consumed_at IS NULL AND invalidated_at IS NULL;
  UPDATE tenant_join_links SET invalidated_at=clock_timestamp(),version=version+1
   WHERE target_squad_ids @> jsonb_build_array(NEW.id::text) AND invalidated_at IS NULL;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER squad_archive_closes_grants BEFORE UPDATE ON squads FOR EACH ROW EXECUTE FUNCTION close_archived_squad();

-- Preserve legitimate old links and record the precise policy activation time.
CREATE TABLE join_policy_activation (id boolean PRIMARY KEY DEFAULT true CHECK(id), activated_at timestamptz NOT NULL DEFAULT clock_timestamp());
INSERT INTO join_policy_activation DEFAULT VALUES;
UPDATE tenant_invitations i SET invalidated_at=clock_timestamp(),version=version+1
 WHERE consumed_at IS NULL AND revoked_at IS NULL AND NOT EXISTS (
 SELECT 1 FROM tenant_memberships m JOIN users u ON u.id=m.user_id
 WHERE m.tenant_id=i.tenant_id AND m.user_id=i.created_by AND m.role='admin' AND m.status='active' AND u.status='active' AND u.deleted_at IS NULL);
UPDATE tenant_join_links i SET invalidated_at=clock_timestamp(),version=version+1
 WHERE revoked_at IS NULL AND NOT EXISTS (
 SELECT 1 FROM tenant_memberships m JOIN users u ON u.id=m.user_id
 WHERE m.tenant_id=i.tenant_id AND m.user_id=i.created_by AND m.role='admin' AND m.status='active' AND u.status='active' AND u.deleted_at IS NULL);
