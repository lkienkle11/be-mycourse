package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
)

func (s *InstructorService) ListTickets(ctx context.Context, f domain.TicketFilter) ([]domain.Ticket, int64, error) {
	return s.repo.ListTickets(ctx, f)
}

func (s *InstructorService) GetTicket(ctx context.Context, id string) (*domain.Ticket, error) {
	return s.repo.GetTicketByID(ctx, id)
}

func (s *InstructorService) CreateTicket(ctx context.Context, userID string, subject string) (*domain.Ticket, error) {
	subject = strings.TrimSpace(subject)
	return s.repo.CreateTicket(ctx, userID, subject)
}

func (s *InstructorService) CloseTicket(ctx context.Context, id string) (*domain.Ticket, error) {
	t, err := s.repo.GetTicketByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status == domain.TicketStatusClosed {
		return t, nil
	}
	if err := s.repo.CloseTicket(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.GetTicketByID(ctx, id)
}

func (s *InstructorService) ListTicketMessages(ctx context.Context, ticketID string) ([]domain.TicketMessage, error) {
	return s.repo.ListMessages(ctx, ticketID)
}

func (s *InstructorService) AddTicketMessage(ctx context.Context, ticketID string, authorUserID string, body string) (*domain.TicketMessage, error) {
	t, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t.Status == domain.TicketStatusClosed {
		return nil, domain.ErrTicketClosed
	}
	body = strings.TrimSpace(body)
	return s.repo.AddMessage(ctx, ticketID, authorUserID, body)
}

// ComingSoon is a placeholder for assignments / activity log APIs.
func (s *InstructorService) ComingSoon() {}
