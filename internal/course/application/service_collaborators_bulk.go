package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/course/domain"
)

func prepareCollaboratorBulkInput(userIDs []string, role string) ([]string, string) {
	normalizedRole := strings.ToUpper(strings.TrimSpace(role))
	if normalizedRole == "" {
		normalizedRole = domain.CollaboratorRoleEditor
	}
	seen := make(map[string]struct{}, len(userIDs))
	out := make([]string, 0, len(userIDs))
	for _, rawID := range userIDs {
		userID := strings.TrimSpace(rawID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		out = append(out, userID)
	}
	return out, normalizedRole
}

func (s *CourseService) AddCollaboratorsBulk(
	ctx context.Context,
	courseID string,
	actorUserID string,
	userIDs []string,
	role string,
) (domain.CollaboratorBulkResult, error) {
	preparedIDs, normalizedRole := prepareCollaboratorBulkInput(userIDs, role)
	if len(preparedIDs) == 0 {
		return domain.CollaboratorBulkResult{
			Added:  []domain.Collaborator{},
			Failed: []domain.CollaboratorBulkFailure{},
		}, nil
	}
	return s.repo.AddCollaboratorsBulk(ctx, courseID, actorUserID, preparedIDs, normalizedRole)
}
