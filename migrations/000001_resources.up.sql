-- resources: things a permission protects, e.g. 'user', 'payment'. Added by migrations only.
-- key:         e.g. 'user'
-- name:        e.g. 'Users'
-- description: e.g. 'Company employees who can sign in'
CREATE TABLE resources (
    key         TEXT PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*$'),
    name        TEXT NOT NULL CHECK (name <> ''),
    description TEXT NOT NULL DEFAULT ''
);

INSERT INTO resources (key, name) VALUES
    ('user', 'Users'),
    ('role', 'Roles');
