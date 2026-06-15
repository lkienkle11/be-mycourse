package application

import "testing"

func TestVideoDurationSecondsFromTelemetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		typed   float64
		bunny   float64
		want    int64
	}{
		{name: "typed wins", typed: 120, bunny: 190, want: 120},
		{name: "bunny fallback", typed: 0, bunny: 190.4, want: 190},
		{name: "both zero", typed: 0, bunny: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := videoDurationSecondsFromTelemetry(tt.typed, tt.bunny)
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}
