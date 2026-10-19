-- roles: named sets of permissions, managed by admins. A role can be deleted
-- while no user has it; its role_permissions rows go with it.
-- key:         e.g. 'accountant'
-- name:        e.g. 'Accountant'
-- description: e.g. 'Sees and confirms payments'
-- permissions: the role's permissions joined per resource, as they go into the
--              access token; written only by the role_permissions trigger,
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

-- A change in role_permissions recomputes the role's permissions.
-- role_permissions rows are only inserted and deleted, never updated.
CREATE FUNCTION role_permissions_refresh_role() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    role_key TEXT := CASE TG_OP WHEN 'INSERT' THEN NEW.role ELSE OLD.role END;
BEGIN
    UPDATE roles
    SET permissions = role_permission_keys(role_key), updated_at = now()
    WHERE key = role_key;
    RETURN NULL;
END $$;

CREATE TRIGGER role_permissions_refresh_role AFTER INSERT OR DELETE ON role_permissions
    FOR EACH ROW EXECUTE FUNCTION role_permissions_refresh_role();

INSERT INTO roles (key, name) VALUES
    ('admin', 'Administrator'),
    ('user',  'User');

INSERT INTO role_permissions (role, permission) SELECT 'admin', key FROM permissions;
