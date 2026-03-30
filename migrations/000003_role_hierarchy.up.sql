-- Role tree (single parent) + transitive closure for O(depth)-free permission resolution

ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS parent_id BIGINT REFERENCES roles (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_roles_parent_id ON roles (parent_id);

CREATE TABLE IF NOT EXISTS role_closure (
    ancestor_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    descendant_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (ancestor_id, descendant_id)
);

CREATE INDEX IF NOT EXISTS idx_role_closure_desc ON role_closure (descendant_id);
CREATE INDEX IF NOT EXISTS idx_role_closure_anc ON role_closure (ancestor_id);

-- Full closure backfill (one-off migration; runtime reads use a single flat JOIN, no recursion in app code)
INSERT INTO role_closure (ancestor_id, descendant_id)
WITH RECURSIVE reach (ancestor_id, descendant_id) AS (
    SELECT id, id FROM roles
    UNION ALL
    SELECT reach.ancestor_id, r.id
    FROM roles r
    INNER JOIN reach ON reach.descendant_id = r.parent_id
    WHERE r.parent_id IS NOT NULL
)
SELECT DISTINCT ancestor_id, descendant_id FROM reach
ON CONFLICT DO NOTHING;
