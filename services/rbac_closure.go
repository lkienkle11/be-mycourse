package services

import (
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/dbschema"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/sqlnamed"
)

const (
	rbacSQLRebuildClosureDeleteTmpl = `DELETE FROM %s`
	// Same transitive-closure logic as migrations/000003_role_hierarchy.up.sql.
	rbacSQLRebuildClosureInsertRecursiveTmpl = `
INSERT INTO %s (ancestor_id, descendant_id)
WITH RECURSIVE reach (ancestor_id, descendant_id) AS (
	SELECT id, id FROM %s
	UNION ALL
	SELECT reach.ancestor_id, r.id
	FROM %s r
	INNER JOIN reach ON reach.descendant_id = r.parent_id
	WHERE r.parent_id IS NOT NULL
)
SELECT DISTINCT ancestor_id, descendant_id FROM reach
ON CONFLICT DO NOTHING
`
	rbacSQLInsertClosureRootTmpl = `
INSERT INTO %s (ancestor_id, descendant_id) VALUES (:role_id, :role_id)
`
	rbacSQLInsertClosureWithParentTmpl = `
INSERT INTO %s (ancestor_id, descendant_id)
SELECT ancestor_id, :role_id FROM %s WHERE descendant_id = :parent_id
UNION ALL
SELECT :role_id, :role_id
ON CONFLICT DO NOTHING
`
	rbacSQLCountClosureCycleTmpl = `
SELECT COUNT(*) FROM %s WHERE ancestor_id = :role_id AND descendant_id = :new_parent_id
`
)

var (
	rbacSQLRebuildClosureDelete          string
	rbacSQLRebuildClosureInsertRecursive string
	rbacSQLInsertClosureRoot             string
	rbacSQLInsertClosureWithParent       string
	rbacSQLCountClosureCycle             string
)

func init() {
	rc := dbschema.RBAC.RoleClosure()
	roles := dbschema.RBAC.Roles()
	rbacSQLRebuildClosureDelete = fmt.Sprintf(rbacSQLRebuildClosureDeleteTmpl, rc)
	rbacSQLRebuildClosureInsertRecursive = fmt.Sprintf(rbacSQLRebuildClosureInsertRecursiveTmpl, rc, roles, roles)
	rbacSQLInsertClosureRoot = fmt.Sprintf(rbacSQLInsertClosureRootTmpl, rc)
	rbacSQLInsertClosureWithParent = fmt.Sprintf(rbacSQLInsertClosureWithParentTmpl, rc, rc)
	rbacSQLCountClosureCycle = fmt.Sprintf(rbacSQLCountClosureCycleTmpl, rc)
}

// rebuildRoleClosure recomputes role_closure from roles.parent_id in one statement (PostgreSQL WITH RECURSIVE),
// matching migrations/000003_role_hierarchy.up.sql — no iterative INSERT loop in Go.
func rebuildRoleClosure(tx *gorm.DB) error {
	if err := tx.Exec(rbacSQLRebuildClosureDelete).Error; err != nil {
		return err
	}
	return tx.Exec(rbacSQLRebuildClosureInsertRecursive).Error
}

func insertClosureForNewRole(tx *gorm.DB, roleID uint, parentID *uint) error {
	if parentID == nil {
		q, args, err := sqlnamed.Postgres(rbacSQLInsertClosureRoot, map[string]interface{}{"role_id": roleID})
		if err != nil {
			return err
		}
		return tx.Exec(q, args...).Error
	}
	q, args, err := sqlnamed.Postgres(rbacSQLInsertClosureWithParent, map[string]interface{}{
		"role_id":   roleID,
		"parent_id": *parentID,
	})
	if err != nil {
		return err
	}
	return tx.Exec(q, args...).Error
}

func wouldCreateRoleCycle(tx *gorm.DB, roleID, newParentID uint) (bool, error) {
	if newParentID == roleID {
		return true, nil
	}
	q, args, err := sqlnamed.Postgres(rbacSQLCountClosureCycle, map[string]interface{}{
		"role_id":       roleID,
		"new_parent_id": newParentID,
	})
	if err != nil {
		return false, err
	}
	var n int64
	if err := tx.Raw(q, args...).Scan(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func assertParentExists(tx *gorm.DB, parentID uint) error {
	var n int64
	if err := tx.Model(&models.Role{}).Where("id = ?", parentID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
