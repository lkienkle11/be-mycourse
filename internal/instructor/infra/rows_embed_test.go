package infra

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestApplicationRowSchemaFlattensEmbeddedProfileColumns(t *testing.T) {
	t.Parallel()
	s, err := schema.Parse(&applicationRow{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	for _, col := range []string{"current_job_title_id", "bio", "cv_file_id", "headline"} {
		if _, ok := s.FieldsByDBName[col]; !ok {
			var names []string
			for k := range s.FieldsByDBName {
				names = append(names, k)
			}
			t.Fatalf("schema missing flattened column %q; have %v", col, names)
		}
	}
}

func TestProfileRowSchemaFlattensEmbeddedProfileColumns(t *testing.T) {
	t.Parallel()
	s, err := schema.Parse(&profileRow{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if _, ok := s.FieldsByDBName["current_job_title_id"]; !ok {
		t.Fatal("schema missing flattened current_job_title_id on profileRow")
	}
}
