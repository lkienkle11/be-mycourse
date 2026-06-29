package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/course/domain"
	sharedutils "mycourse-io-be/internal/shared/utils"
)

func prepareCollaboratorBulkInput(userIDs []string, role string) ([]string, string) {
	normalizedRole := strings.ToUpper(strings.TrimSpace(role))
	if normalizedRole == "" {
		normalizedRole = domain.CollaboratorRoleEditor
	}
	return sharedutils.PrepareBulkUserIDs(userIDs), normalizedRole
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
