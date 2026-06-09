package appcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
)

type legacyImportReport struct {
	Statements   int
	ImportedRows int
	SkippedRows  int
	IDMapPath    string
}

type legacyIDMap struct {
	GeneratedAt string                       `json:"generated_at"`
	SourceDump  string                       `json:"source_dump"`
	Tables      map[string]map[string]string `json:"tables"`
}

type legacyInsertStmt struct {
	Table   string
	Columns []string
	Values  []string
}

type legacyRewriteCtx struct {
	table         string
	row           map[string]any
	idmap         *legacyIDMap
	coursePatches *[]legacyCourseVersionPatch
}

// legacyCourseVersionPatch defers courses.current_*_version_id until course_versions exist.
type legacyCourseVersionPatch struct {
	courseNewID    string
	draftOldID     string
	publishedOldID string
}

type csvSplitState struct {
	out      []string
	builder  strings.Builder
	inSingle bool
	inDouble bool
	depth    int
}

type legacyRewriteHandler func(*legacyRewriteCtx) error

// importLegacyDump parses INSERT statements from a legacy SQL backup and imports rows
// into the UUID-v7 schema while producing old->new id maps.
func importLegacyDump(ctx context.Context, db *gorm.DB, dumpPath string) (*legacyImportReport, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	raw, err := os.ReadFile(dumpPath)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	stmts, err := parseLegacyInsertStatements(string(raw))
	if err != nil {
		return nil, err
	}

	idmap := &legacyIDMap{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SourceDump:  dumpPath,
		Tables:      map[string]map[string]string{},
	}
	report := &legacyImportReport{Statements: len(stmts)}

	ordered := orderStatementsForImport(stmts)
	var coursePatches []legacyCourseVersionPatch
	txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, st := range ordered {
			if !isSupportedImportTable(st.Table) {
				report.SkippedRows++
				continue
			}
			if err := importLegacyStatement(tx, st, idmap, report, &coursePatches); err != nil {
				return err
			}
		}
		return applyLegacyCourseVersionPatches(tx, idmap, coursePatches)
	})
	if txErr != nil {
		return nil, txErr
	}

	idMapPath, err := writeLegacyIDMapFile(dumpPath, idmap)
	if err != nil {
		return nil, err
	}
	report.IDMapPath = idMapPath
	return report, nil
}

func orderStatementsForImport(stmts []legacyInsertStmt) []legacyInsertStmt {
	priority := map[string]int{
		"permissions": 1, "roles": 1, "role_permissions": 2,
		"users": 3, "user_roles": 4, "user_permissions": 4, "system_privileged_users": 4,
		"media_files": 5, "media_pending_cloud_cleanup": 5,
		"course_levels": 6, "course_topics": 6, "tags": 6, "course_outcomes": 6, "course_skills": 6,
		"instructor_applications": 8, "instructor_profiles": 8, "instructor_expertise_topics": 8, "instructor_expertise_skills": 8,
		"instructor_tickets": 8, "instructor_ticket_messages": 8,
		"courses": 9, "course_versions": 10,
		"course_version_tags": 11, "course_version_skills": 11, "course_version_outcomes": 11,
		"course_collaborators": 12, "course_sections": 12, "course_lessons": 13, "course_sub_lessons": 14,
		"course_sub_lesson_videos": 15, "course_sub_lesson_texts": 15, "course_sub_lesson_quizzes": 15, "course_sub_lesson_quiz_options": 16,
		"course_edit_leases": 17, "course_enrollments": 17, "course_progress_items": 18, "system_app_config": 19,
	}
	out := append([]legacyInsertStmt(nil), stmts...)
	slices.SortFunc(out, func(a, b legacyInsertStmt) int {
		pa := priority[a.Table]
		pb := priority[b.Table]
		if pa == pb {
			if a.Table < b.Table {
				return -1
			}
			if a.Table > b.Table {
				return 1
			}
			return 0
		}
		if pa < pb {
			return -1
		}
		return 1
	})
	return out
}

func legacyInsertOnConflictSQL(table string) string {
	switch table {
	case "permissions":
		return " ON CONFLICT (permission_id) DO NOTHING"
	case "roles":
		return " ON CONFLICT (id) DO NOTHING"
	case "role_permissions":
		return " ON CONFLICT (role_id, permission_id) DO NOTHING"
	case "system_app_config":
		return " ON CONFLICT (id) DO NOTHING"
	default:
		return ""
	}
}

func isSupportedImportTable(t string) bool {
	switch t {
	case "permissions", "roles", "role_permissions",
		"users", "user_roles", "user_permissions", "system_privileged_users", "system_app_config",
		"course_levels", "course_topics", "tags", "course_outcomes", "course_skills",
		"media_files", "media_pending_cloud_cleanup",
		"instructor_applications", "instructor_profiles", "instructor_expertise_topics", "instructor_expertise_skills",
		"instructor_tickets", "instructor_ticket_messages",
		"courses", "course_versions", "course_version_tags", "course_version_skills", "course_version_outcomes",
		"course_collaborators", "course_sections", "course_lessons", "course_sub_lessons",
		"course_sub_lesson_videos", "course_sub_lesson_texts", "course_sub_lesson_quizzes", "course_sub_lesson_quiz_options",
		"course_edit_leases", "course_enrollments", "course_progress_items":
		return true
	default:
		return false
	}
}

func importLegacyStatement(tx *gorm.DB, st legacyInsertStmt, idmap *legacyIDMap, report *legacyImportReport, coursePatches *[]legacyCourseVersionPatch) error {
	row, err := buildInsertRow(st.Columns, st.Values)
	if err != nil {
		return fmt.Errorf("parse row %s: %w", st.Table, err)
	}
	newRow, oldID, newID, skip, err := rewriteLegacyRowForUUID(st.Table, row, idmap, coursePatches)
	if err != nil {
		return fmt.Errorf("rewrite %s: %w", st.Table, err)
	}
	if skip {
		report.SkippedRows++
		return nil
	}
	cols := make([]string, 0, len(newRow))
	args := make([]any, 0, len(newRow))
	for _, c := range st.Columns {
		if v, ok := newRow[c]; ok {
			cols = append(cols, c)
			args = append(args, v)
		}
	}
	if len(cols) == 0 {
		report.SkippedRows++
		return nil
	}
	placeholders := make([]string, 0, len(cols))
	for i := range cols {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}
	onConflict := legacyInsertOnConflictSQL(st.Table)
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)%s",
		st.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		onConflict,
	)
	result := tx.Exec(sql, args...)
	if result.Error != nil {
		return fmt.Errorf("insert %s: %w", st.Table, result.Error)
	}
	if result.RowsAffected == 0 && onConflict != "" {
		report.SkippedRows++
		return nil
	}
	if oldID != "" && newID != "" {
		if idmap.Tables[st.Table] == nil {
			idmap.Tables[st.Table] = map[string]string{}
		}
		idmap.Tables[st.Table][oldID] = newID
	}
	report.ImportedRows++
	return nil
}

func applyLegacyCourseVersionPatches(tx *gorm.DB, idmap *legacyIDMap, patches []legacyCourseVersionPatch) error {
	for _, p := range patches {
		sets := make([]string, 0, 2)
		args := make([]any, 0, 3)
		argN := 1
		if p.draftOldID != "" {
			mapped := lookupMappedID(idmap, "course_versions", p.draftOldID)
			if mapped == "" {
				return fmt.Errorf("missing id map for courses.current_draft_version_id=%s", p.draftOldID)
			}
			sets = append(sets, fmt.Sprintf("current_draft_version_id = $%d", argN))
			args = append(args, mapped)
			argN++
		}
		if p.publishedOldID != "" {
			mapped := lookupMappedID(idmap, "course_versions", p.publishedOldID)
			if mapped == "" {
				return fmt.Errorf("missing id map for courses.current_published_version_id=%s", p.publishedOldID)
			}
			sets = append(sets, fmt.Sprintf("current_published_version_id = $%d", argN))
			args = append(args, mapped)
			argN++
		}
		if len(sets) == 0 {
			continue
		}
		args = append(args, p.courseNewID)
		sql := fmt.Sprintf("UPDATE courses SET %s WHERE id = $%d", strings.Join(sets, ", "), argN)
		if err := tx.Exec(sql, args...).Error; err != nil {
			return fmt.Errorf("patch course version refs: %w", err)
		}
	}
	return nil
}

func writeLegacyIDMapFile(dumpPath string, idmap *legacyIDMap) (string, error) {
	if idmap == nil {
		return "", errors.New("idmap is nil")
	}
	stamp := time.Now().UTC().Format("02012006-150405")
	base := strings.TrimSuffix(filepath.Base(dumpPath), filepath.Ext(dumpPath))
	outName := fmt.Sprintf("%s.%s.idmap.json", base, stamp)
	outPath := filepath.Join(filepath.Dir(dumpPath), outName)
	b, err := json.MarshalIndent(idmap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal idmap: %w", err)
	}
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		return "", fmt.Errorf("write idmap: %w", err)
	}
	return outPath, nil
}
