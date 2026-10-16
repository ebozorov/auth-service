-- roles: named sets of permissions, managed by admins. A role can be deleted
-- while no user has it; its role_permissions rows go with it.
-- key:         e.g. 'accountant'
-- name:        e.g. 'Accountant'
-- description: e.g. 'Sees and confirms payments'
-- permissions: the role's permissions joined per resource, as they go into the
--              access token; computed by triggers, never written directly,
--              e.g. {role:r,user:cru}
-- created_at:  e.g. '2026-10-16 09:00:00+05'
-- updated_at:  e.g. NULL until the first change
CREATE TABLE roles (
    key         TEXT        PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*$'),
    name        TEXT        NOT NULL CHECK (name <> ''),
    description TEXT        NOT NULL DEFAULT '',
    permissions TEXT[]      NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ
);

-- role_permissions: which permissions a role grants.
-- role:       e.g. 'admin'
-- permission: e.g. 'user:c'
CREATE TABLE role_permissions (
    role       TEXT NOT NULL REFERENCES roles (key) ON DELETE CASCADE,
    permission TEXT NOT NULL REFERENCES permissions (key) ON DELETE CASCADE,
    PRIMARY KEY (role, permission)
);

CREATE INDEX role_permissions_permission_idx ON role_permissions (permission);

-- The role's permissions joined per resource, actions in sort_order, e.g. {role:r,user:cru}.
CREATE FUNCTION role_permission_keys(role_key TEXT) RETURNS TEXT[] LANGUAGE sql STABLE AS $$
    SELECT coalesce(array_agg(x.permission ORDER BY x.permission), '{}')
    FROM (
        SELECT p.resource || ':' || string_agg(p.action::text, '' ORDER BY a.sort_order) AS permission
        FROM role_permissions rp
        JOIN permissions p ON p.key = rp.permission
        JOIN actions a ON a.key = p.action
        WHERE rp.role = role_key
        GROUP BY p.resource
    ) x
$$;

-- roles.permissions is always computed, never written directly.
CREATE FUNCTION roles_set_permissions() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.permissions := role_permission_keys(NEW.key);
    RETURN NEW;
END $$;

CREATE TRIGGER roles_set_permissions BEFORE INSERT OR UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION roles_set_permissions();

-- A change in role_permissions touches the role, which recomputes roles.permissions.
CREATE FUNCTION role_permissions_touch_role() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'DELETE' THEN
        UPDATE roles SET updated_at = now() WHERE key = NEW.role;
    END IF;
    IF TG_OP <> 'INSERT' THEN
        UPDATE roles SET updated_at = now() WHERE key = OLD.role;
    END IF;
    RETURN NULL;
END $$;

CREATE TRIGGER role_permissions_touch_role AFTER INSERT OR UPDATE OR DELETE ON role_permissions
    FOR EACH ROW EXECUTE FUNCTION role_permissions_touch_role();

INSERT INTO roles (key, name) VALUES
    ('admin', 'Administrator'),
    ('user',  'User');

INSERT INTO role_permissions (role, permission) SELECT 'admin', key FROM permissions;
