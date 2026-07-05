package infra

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestApplicationWithUserRowSchemaMapsPrimaryKeyID(t *testing.T) {
	t.Parallel()
	s, err := schema.Parse(&struct {
		Row                applicationRow `gorm:"embedded"`
		identityProjection `gorm:"embedded"`
	}{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	idField, ok := s.FieldsByDBName["id"]
	if !ok {
		t.Fatal("schema missing id column on list scan wrapper")
	}
	if idField.Name != "ID" {
		t.Fatalf("expected ID field on embedded applicationRow, got %q", idField.Name)
	}
}

func TestApplicationWithUserRowSchemaMapsIdentityColumns(t *testing.T) {
	t.Parallel()
	s, err := schema.Parse(&struct {
		Row          applicationRow `gorm:"embedded"`
		FullName     string         `gorm:"column:full_name"`
		Email        string         `gorm:"column:email"`
		Phone        string         `gorm:"column:phone"`
		AvatarFileID string         `gorm:"column:avatar_file_id"`
	}{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	for _, col := range []string{"full_name", "email", "phone", "avatar_file_id"} {
		if _, ok := s.FieldsByDBName[col]; !ok {
			t.Fatalf("schema missing identity column %q on list scan wrapper", col)
		}
	}
}
