package application

import (
	"context"

	"mycourse-io-be/internal/instructor/domain"
)

func (s *InstructorService) ListExpertiseTopics(ctx context.Context, userID string, locale string) ([]domain.ExpertiseTopic, error) {
	v, err := s.repo.ListExpertise(ctx, userID, true, locale)
	if err != nil {
		return nil, err
	}
	return v.([]domain.ExpertiseTopic), nil
}

func (s *InstructorService) AddExpertiseTopic(ctx context.Context, userID string, topicID string) (*domain.ExpertiseTopic, error) {
	return castInsertedExpertise[*domain.ExpertiseTopic](s.repo.InsertExpertise(ctx, userID, topicID, true))
}

func (s *InstructorService) DeleteExpertiseTopic(ctx context.Context, id string) error {
	return s.repo.DeleteTopic(ctx, id)
}

func (s *InstructorService) DeleteAllExpertiseForUser(ctx context.Context, userID string) error {
	if err := s.repo.DeleteAllTopicsForUser(ctx, userID); err != nil {
		return err
	}
	return s.repo.DeleteAllSkillsForUser(ctx, userID)
}

func (s *InstructorService) ListExpertiseSkills(ctx context.Context, userID string, locale string) ([]domain.ExpertiseSkill, error) {
	v, err := s.repo.ListExpertise(ctx, userID, false, locale)
	if err != nil {
		return nil, err
	}
	return v.([]domain.ExpertiseSkill), nil
}

func (s *InstructorService) AddExpertiseSkill(ctx context.Context, userID string, skillID string) (*domain.ExpertiseSkill, error) {
	return castInsertedExpertise[*domain.ExpertiseSkill](s.repo.InsertExpertise(ctx, userID, skillID, false))
}

func castInsertedExpertise[T any](v any, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	return v.(T), nil
}

func (s *InstructorService) DeleteExpertiseSkill(ctx context.Context, id string) error {
	return s.repo.DeleteSkill(ctx, id)
}
