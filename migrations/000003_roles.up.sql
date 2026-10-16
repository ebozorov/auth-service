-- roles: named sets of permissions, managed by admins. system roles cannot be deleted.
-- key:         e.g. 'accountant'
-- name:        e.g. 'Accountant'
-- description: e.g. 'Sees and confirms payments'
-- system:      e.g. true for 'admin' and 'user'
-- created_at:  e.g. '2026-10-16 09:00:00+05'
-- updated_at:  e.g. NULL until the first change
CREATE TABLE roles (
    key         TEXT        PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*$'),
    name        TEXT        NOT NULL CHECK (name <> ''),
    description TEXT        NOT NULL DEFAULT '',
    system      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ
);

CREATE FUNCTION roles_keep_system() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'system role % cannot be deleted', OLD.key;
END $$;

CREATE TRIGGER roles_keep_system BEFORE DELETE ON roles
    FOR EACH ROW WHEN (OLD.system) EXECUTE FUNCTION roles_keep_system();

-- role_permissions: which permissions a role grants.
-- role:       e.g. 'admin'
-- permission: e.g. 'user:crud'
CREATE TABLE role_permissions (
    role       TEXT NOT NULL REFERENCES roles (key) ON DELETE CASCADE,
    permission TEXT NOT NULL REFERENCES permissions (key) ON DELETE CASCADE,
    PRIMARY KEY (role, permission)
);

CREATE INDEX role_permissions_permission_idx ON role_permissions (permission);

-- user_roles: roles of a user; the token gets the permissions of all of them.
-- A role in use cannot be deleted (RESTRICT).
-- user_id: e.g. 587364927401230337
-- role:    e.g. 'admin'
CREATE TABLE user_roles (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role    TEXT   NOT NULL REFERENCES roles (key) ON DELETE RESTRICT,
    PRIMARY KEY (user_id, role)
);

CREATE INDEX user_roles_role_idx ON user_roles (role);

INSERT INTO roles (key, name, system) VALUES
    ('admin', 'Administrator', TRUE),
    ('user',  'User',          TRUE);

INSERT INTO role_permissions (role, permission) VALUES
    ('admin', 'user:crud'),
    ('admin', 'role:crud');

-- users.role stays until the code reads user_roles; a later migration drops it.
INSERT INTO user_roles (user_id, role) SELECT id, role FROM users;
