-- permissions: one action on one resource; roles get permissions through role_permissions.
-- key:         resource || ':' || action, e.g. 'user:c'
-- resource:    e.g. 'user'
-- action:      e.g. 'c'
-- name:        e.g. 'Create users'
-- description: e.g. 'Add new employees'
CREATE TABLE permissions (
    key         TEXT    PRIMARY KEY CHECK (key = resource || ':' || action),
    resource    TEXT    NOT NULL REFERENCES resources (key),
    action      CHAR(1) NOT NULL REFERENCES actions (key),
    name        TEXT    NOT NULL CHECK (name <> ''),
    description TEXT    NOT NULL DEFAULT '',
    UNIQUE (resource, action)
);

INSERT INTO permissions (key, resource, action, name) VALUES
    ('user:c', 'user', 'c', 'Create users'),
    ('user:r', 'user', 'r', 'Read users'),
    ('user:u', 'user', 'u', 'Update users'),
    ('user:d', 'user', 'd', 'Delete users'),
    ('role:c', 'role', 'c', 'Create roles'),
    ('role:r', 'role', 'r', 'Read roles'),
    ('role:u', 'role', 'u', 'Update roles'),
    ('role:d', 'role', 'd', 'Delete roles');
