package models

import (
	"time"

	"mycourse-io-be/dbschema"
)

// Permission is the smallest authorization unit (e.g. course.read, rbac.manage).
type Permission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:128;not null" json:"code"`
	Description string    `gorm:"size:512" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Permission) TableName() string { return dbschema.RBAC.Permissions() }

// Role groups permissions; users receive roles (user_roles).
// Parent/child: permissions on an ancestor apply to descendants via role_closure (no duplicate inserts on children).
type Role struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Name        string        `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Description string        `gorm:"size:512" json:"description"`
	ParentID    *uint         `gorm:"index" json:"parent_id,omitempty"`
	Parent      *Role         `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children    []Role        `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	// many2many: literal must match dbschema.RBAC.RolePermissions() (Go tags cannot call funcs).
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (Role) TableName() string { return dbschema.RBAC.Roles() }

// RoleClosure stores (ancestor, descendant) for the role tree; maintained by services layer.
type RoleClosure struct {
	AncestorID   uint `gorm:"primaryKey;column:ancestor_id" json:"ancestor_id"`
	DescendantID uint `gorm:"primaryKey;column:descendant_id" json:"descendant_id"`
}

func (RoleClosure) TableName() string { return dbschema.RBAC.RoleClosure() }

// UserRole binds an external user id (e.g. Supabase/JWT sub) to a role.
type UserRole struct {
	UserID string `gorm:"size:128;primaryKey;index:idx_user_roles_user" json:"user_id"`
	RoleID uint   `gorm:"primaryKey" json:"role_id"`
}

func (UserRole) TableName() string { return dbschema.RBAC.UserRoles() }
