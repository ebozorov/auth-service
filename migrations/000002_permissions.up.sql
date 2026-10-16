-- resources: things a permission protects, e.g. 'user', 'payment'. Added by migrations only.
-- key:         e.g. 'user'
-- name:        e.g. 'Users'
-- description: e.g. 'Company employees who can sign in'
CREATE TABLE resources (
    key         TEXT PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*$'),
    name        TEXT NOT NULL CHECK (name <> ''),
    description TEXT NOT NULL DEFAULT ''
);

-- actions: what can be done to a resource. One character, assigned in migrations.
-- Rows are append-only: key and sort_order never change and rows are never deleted,
-- because permissions spell their actions with these characters in this order.
-- key:         e.g. 'r'
-- name:        e.g. 'Read'
-- description: e.g. 'View records'
-- sort_order:  order of the character inside a permission key, e.g. 2 ('c' = 1, 'r' = 2)
CREATE TABLE actions (
    key         CHAR(1)  PRIMARY KEY CHECK (key ~ '^[a-z0-9]$'),
    name        TEXT     NOT NULL UNIQUE CHECK (name <> ''),
    description TEXT     NOT NULL DEFAULT '',
    sort_order  SMALLINT NOT NULL UNIQUE CHECK (sort_order > 0)
);

CREATE FUNCTION actions_keep() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'action %: key and sort_order cannot change and rows cannot be deleted', OLD.key;
END $$;

CREATE TRIGGER actions_keep BEFORE UPDATE OF key, sort_order OR DELETE ON actions
    FOR EACH ROW EXECUTE FUNCTION actions_keep();

-- permissions: a set of actions on one resource; key goes into the access token.
-- key:         resource || ':' || actions, e.g. 'user:crud'
-- resource:    e.g. 'user'
-- actions:     action keys in sort_order, each once, e.g. 'crud'
-- name:        e.g. 'Manage users'
-- description: e.g. 'Create, view, change and disable users'
CREATE TABLE permissions (
    key         TEXT PRIMARY KEY CHECK (key = resource || ':' || actions),
    resource    TEXT NOT NULL REFERENCES resources (key),
    actions     TEXT NOT NULL CHECK (actions ~ '^[a-z0-9]+$'),
    name        TEXT NOT NULL CHECK (name <> ''),
    description TEXT NOT NULL DEFAULT ''
);

-- Every character must be a known action, used once, in sort_order.
CREATE FUNCTION permissions_check_actions() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    canonical TEXT;
BEGIN
    SELECT string_agg(a.key::text, '' ORDER BY a.sort_order) INTO canonical
    FROM actions a
    WHERE strpos(NEW.actions, a.key::text) > 0;
    IF canonical IS DISTINCT FROM NEW.actions THEN
        RAISE EXCEPTION 'permission %: actions must be known, unique and in order (expected %)',
            NEW.key, coalesce(canonical, '');
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER permissions_check_actions BEFORE INSERT OR UPDATE OF actions ON permissions
    FOR EACH ROW EXECUTE FUNCTION permissions_check_actions();

INSERT INTO resources (key, name) VALUES
    ('user', 'Users'),
    ('role', 'Roles');

INSERT INTO actions (key, name, sort_order) VALUES
    ('c', 'Create', 1),
    ('r', 'Read',   2),
    ('u', 'Update', 3),
    ('d', 'Delete', 4);

INSERT INTO permissions (key, resource, actions, name) VALUES
    ('user:r',    'user', 'r',    'View users'),
    ('user:crud', 'user', 'crud', 'Manage users'),
    ('role:r',    'role', 'r',    'View roles'),
    ('role:crud', 'role', 'crud', 'Manage roles');
