package infra

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/slug"
	"mycourse-io-be/internal/shared/slug/gormslug"
	"mycourse-io-be/internal/shared/timex"
)

// DefaultSlugBackfillBatchSize bounds how many users one backfill round loads,
// resolves and writes with a single UPDATE.
const DefaultSlugBackfillBatchSize = 500

type backfillUserRow struct {
	ID          string `gorm:"column:id"`
	DisplayName string `gorm:"column:display_name"`
}

type backfillAssignment struct {
	UserID string
	Slug   string
}

// BackfillMissingSlugs implements domain.UserRepository. It walks users whose
// slug is NULL (soft-deleted included, so the final NOT NULL constraint can be
// applied table-wide) in id order. Each round runs one SELECT for the batch,
// one availability query for the whole batch and one set-based UPDATE guarded
// by slug IS NULL, so a re-run changes nothing. If the batch UPDATE loses a
// unique-index race it falls back to resolving the batch row by row.
func (r *GormUserRepository) BackfillMissingSlugs(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = DefaultSlugBackfillBatchSize
	}
	updated, cursor := 0, ""
	for {
		var rows []backfillUserRow
		q := r.db.WithContext(ctx).Table("users").Select("id", "display_name").Where("slug IS NULL")
		if cursor != "" {
			q = q.Where("id > ?", cursor)
		}
		if err := q.Order("id").Limit(batchSize).Find(&rows).Error; err != nil {
			return updated, err
		}
		if len(rows) == 0 {
			return updated, nil
		}
		n, err := r.backfillBatch(ctx, rows)
		if err != nil {
			return updated, err
		}
		updated += n
		cursor = rows[len(rows)-1].ID
	}
}

// backfillReplanAttempts bounds how often a batch plan is recomputed after the
// database reports that some planned (suffixed) candidates are already held.
const backfillReplanAttempts = 3

func (r *GormUserRepository) backfillBatch(ctx context.Context, rows []backfillUserRow) (int, error) {
	taken, err := r.takenSlugsForBases(ctx, rows)
	if err != nil {
		return 0, err
	}
	assignments, err := r.planAgainstDatabase(ctx, rows, taken)
	if err != nil {
		return 0, err
	}
	if assignments == nil {
		return r.backfillRowByRow(ctx, rows)
	}
	n, err := r.applyBackfillAssignments(ctx, assignments)
	if err == nil {
		return n, nil
	}
	if !isUserSlugDuplicateKey(err) {
		return 0, err
	}
	return r.backfillRowByRow(ctx, rows)
}

// planAgainstDatabase plans the batch in memory and then verifies the whole
// plan, suffixed candidates included, against active users and the former-slug
// history with one query, replanning around anything already held. The users
// unique index would catch a clash with another active user but never one with
// user_slug_history, so this check is what keeps former slugs reserved. It
// returns nil assignments when the plan does not settle, so the caller falls
// back to resolving row by row.
func (r *GormUserRepository) planAgainstDatabase(ctx context.Context, rows []backfillUserRow, taken map[string]struct{}) ([]backfillAssignment, error) {
	for range backfillReplanAttempts {
		assignments, err := planBackfillSlugs(rows, taken)
		if err != nil {
			return nil, err
		}
		planned := make([]string, len(assignments))
		for i, a := range assignments {
			planned[i] = a.Slug
		}
		conflicts, err := r.takenAmong(ctx, planned)
		if err != nil {
			return nil, err
		}
		if len(conflicts) == 0 {
			return assignments, nil
		}
		for s := range conflicts {
			taken[s] = struct{}{}
		}
	}
	return nil, nil
}

// takenSlugsForBases returns which of the batch's plain (non-suffixed) base
// slugs are already held by an active user or reserved in history.
func (r *GormUserRepository) takenSlugsForBases(ctx context.Context, rows []backfillUserRow) (map[string]struct{}, error) {
	bases := make([]string, 0, len(rows))
	for _, row := range rows {
		if base, mustSuffix := slug.GenerateAutoSlugBase(row.DisplayName, userSlugMaxLen, slug.KindUser); !mustSuffix {
			bases = append(bases, base)
		}
	}
	return r.takenAmong(ctx, bases)
}

// takenAmong returns the subset of slugs already held by an active user or
// recorded in user_slug_history, using two queries for the whole set instead of
// one per slug.
func (r *GormUserRepository) takenAmong(ctx context.Context, slugs []string) (map[string]struct{}, error) {
	taken := make(map[string]struct{})
	if len(slugs) == 0 {
		return taken, nil
	}
	var active, former []string
	if err := r.db.WithContext(ctx).Table("users").
		Where("deleted_at IS NULL AND slug IN ?", slugs).Pluck("slug", &active).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&userSlugHistoryRow{}).
		Where("slug IN ?", slugs).Pluck("slug", &former).Error; err != nil {
		return nil, err
	}
	for _, s := range append(active, former...) {
		taken[s] = struct{}{}
	}
	return taken, nil
}

// planBackfillSlugs assigns every row a slug through the shared resolver using
// only in-memory availability: the already-taken set plus slugs handed out
// earlier in the same batch, so two users with the same display name get
// different slugs. Suffixed candidates are not checked against the database
// here; the unique index plus the row-by-row fallback cover the rare clash.
func planBackfillSlugs(rows []backfillUserRow, taken map[string]struct{}) ([]backfillAssignment, error) {
	used := make(map[string]struct{}, len(taken)+len(rows))
	for s := range taken {
		used[s] = struct{}{}
	}
	avail := func(candidate string) (bool, error) {
		if slug.IsReserved(candidate) {
			return false, nil
		}
		_, isUsed := used[candidate]
		return !isUsed, nil
	}
	out := make([]backfillAssignment, 0, len(rows))
	for _, row := range rows {
		final, _, err := slug.ResolveCreate("", row.DisplayName, slug.KindUser, userSlugMaxLen, avail)
		if err != nil {
			return nil, err
		}
		used[final] = struct{}{}
		out = append(out, backfillAssignment{UserID: row.ID, Slug: final})
	}
	return out, nil
}

// applyBackfillAssignments writes a whole batch with one UPDATE ... FROM
// (VALUES ...) statement, only for rows whose slug is still NULL.
func (r *GormUserRepository) applyBackfillAssignments(ctx context.Context, assignments []backfillAssignment) (int, error) {
	if len(assignments) == 0 {
		return 0, nil
	}
	values := make([]string, len(assignments))
	args := make([]any, 0, len(assignments)*2+1)
	args = append(args, timex.NowUnix())
	for i, a := range assignments {
		values[i] = "(?::uuid, ?)"
		args = append(args, a.UserID, a.Slug)
	}
	stmt := fmt.Sprintf(
		"UPDATE users AS u SET slug = v.slug, updated_at = ? FROM (VALUES %s) AS v(id, slug) WHERE u.id = v.id AND u.slug IS NULL",
		strings.Join(values, ", "),
	)
	res := r.db.WithContext(ctx).Exec(stmt, args...)
	return int(res.RowsAffected), res.Error
}

// backfillRowByRow resolves each user with the database-backed resolver; used
// only when the set-based UPDATE for a batch lost a unique-index race.
func (r *GormUserRepository) backfillRowByRow(ctx context.Context, rows []backfillUserRow) (int, error) {
	updated := 0
	for _, row := range rows {
		n, err := r.backfillOne(ctx, row)
		if err != nil {
			return updated, err
		}
		updated += n
	}
	return updated, nil
}

func (r *GormUserRepository) backfillOne(ctx context.Context, row backfillUserRow) (int, error) {
	updated := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("users").Where("id = ? AND slug IS NULL", row.ID).Count(&count).Error; err != nil || count == 0 {
			return err
		}
		return gormx.RetryInSavepoint(tx, "user_slug_backfill_retry", gormslug.RetryAttempts, isUserSlugDuplicateKey, func() error {
			final, err := resolveUserCreateSlug(ctx, tx, row.DisplayName)
			if err != nil {
				return err
			}
			res := tx.Table("users").Where("id = ? AND slug IS NULL", row.ID).
				Updates(map[string]any{"slug": final, "updated_at": timex.NowUnix()})
			updated = int(res.RowsAffected)
			return res.Error
		})
	})
	return updated, err
}
