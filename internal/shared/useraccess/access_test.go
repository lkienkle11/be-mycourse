package useraccess_test

import (
	"testing"

	"mycourse-io-be/internal/shared/useraccess"
)

func TestCheckAccessible(t *testing.T) {
	runCheckAccessibleCases(t, buildCheckAccessibleCases())
}

type checkAccessibleCase struct {
	name     string
	snapshot *useraccess.Snapshot
	wantErr  error
}

func runCheckAccessibleCases(t *testing.T, tests []checkAccessibleCase) {
	t.Helper()
	now := int64(100)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := useraccess.CheckAccessible(tt.snapshot, now)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if err != tt.wantErr {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func buildCheckAccessibleCases() []checkAccessibleCase {
	deletedAt := int64(1)
	futureBan := int64(200)
	pastBan := int64(50)

	return []checkAccessibleCase{
		{
			name:     "nil snapshot",
			snapshot: nil,
			wantErr:  useraccess.ErrUserNotFound,
		},
		{
			name: "soft deleted",
			snapshot: &useraccess.Snapshot{
				DeletedAt: &deletedAt,
			},
			wantErr: useraccess.ErrUserNotFound,
		},
		{
			name: "disabled user",
			snapshot: &useraccess.Snapshot{
				IsDisabled: true,
			},
			wantErr: useraccess.ErrUserDisabled,
		},
		{
			name: "active ban",
			snapshot: &useraccess.Snapshot{
				BannedUntil: &futureBan,
			},
			wantErr: useraccess.ErrUserBanned,
		},
		{
			name: "past ban is accessible",
			snapshot: &useraccess.Snapshot{
				BannedUntil: &pastBan,
			},
			wantErr: nil,
		},
		{
			name:     "accessible",
			snapshot: &useraccess.Snapshot{},
			wantErr:  nil,
		},
	}
}
