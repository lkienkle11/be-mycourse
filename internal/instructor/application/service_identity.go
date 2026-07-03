package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
)

func hydrateAvatarURLsByAccessor[T any](
	ctx context.Context,
	hydrator AvatarHydrator,
	rows []T,
	getFileID func(T) string,
	setAvatarURL func(*T, string),
) error {
	if hydrator == nil || len(rows) == 0 {
		return nil
	}
	fileIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if id := strings.TrimSpace(getFileID(row)); id != "" {
			fileIDs = append(fileIDs, id)
		}
	}
	if len(fileIDs) == 0 {
		return nil
	}
	urls, err := hydrator.ResolveAvatarURLs(ctx, fileIDs)
	if err != nil {
		return err
	}
	for i := range rows {
		if u, ok := urls[strings.TrimSpace(getFileID(rows[i]))]; ok {
			setAvatarURL(&rows[i], u)
		}
	}
	return nil
}

func listAndHydrate[T any](
	rows []T,
	total int64,
	err error,
	hydrateFn func([]T) error,
) ([]T, int64, error) {
	if err != nil {
		return nil, 0, err
	}
	if err := hydrateFn(rows); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func loadAndHydrateOne[T any](
	row *T,
	err error,
	hydrateFn func([]T) error,
) (*T, error) {
	if err != nil {
		return nil, err
	}
	items := []T{*row}
	if err := hydrateFn(items); err != nil {
		return nil, err
	}
	*row = items[0]
	return row, nil
}

func listWithIdentity[T any](
	s *InstructorService,
	ctx context.Context,
	listFn func() ([]T, int64, error),
	getFileID func(T) string,
	setAvatarURL func(*T, string),
) ([]T, int64, error) {
	rows, total, err := listFn()
	return listAndHydrate(rows, total, err, func(items []T) error {
		return hydrateAvatarURLsByAccessor(ctx, s.hydrator, items, getFileID, setAvatarURL)
	})
}

func loadOneWithIdentity[T any](
	s *InstructorService,
	ctx context.Context,
	loadFn func() (*T, error),
	getFileID func(T) string,
	setAvatarURL func(*T, string),
) (*T, error) {
	row, err := loadFn()
	return loadAndHydrateOne(row, err, func(items []T) error {
		return hydrateAvatarURLsByAccessor(ctx, s.hydrator, items, getFileID, setAvatarURL)
	})
}

func applicationAvatarFileID(row domain.Application) string { return row.AvatarFileID }

func setApplicationAvatarURL(row *domain.Application, url string) { row.AvatarURL = url }

func profileAvatarFileID(row domain.Profile) string { return row.AvatarFileID }

func setProfileAvatarURL(row *domain.Profile, url string) { row.AvatarURL = url }

func ticketAvatarFileID(row domain.Ticket) string { return row.AvatarFileID }

func setTicketAvatarURL(row *domain.Ticket, url string) { row.AvatarURL = url }
