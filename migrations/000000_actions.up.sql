-- actions: what can be done to a resource. One character, assigned in migrations.
-- key:         e.g. 'r'
-- name:        e.g. 'Read'
-- description: e.g. 'View records'
-- sort_order:  order of the characters when a role's actions on one resource are joined,
--              e.g. 2 ('c' = 1, 'r' = 2), so the token gets 'user:cr'
CREATE TABLE actions (
    key         CHAR(1)  PRIMARY KEY CHECK (key ~ '^[a-z0-9]$'),
    name        TEXT     NOT NULL UNIQUE CHECK (name <> ''),
    description TEXT     NOT NULL DEFAULT '',
    sort_order  SMALLINT NOT NULL UNIQUE CHECK (sort_order > 0)
);

INSERT INTO actions (key, name, sort_order) VALUES
    ('c', 'Create', 1),
    ('r', 'Read',   2),
    ('u', 'Update', 3),
    ('d', 'Delete', 4);
