package utils

import "testing"

func TestSlugifyName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"36 Thanh Hóa", "36-thanh-hoa"},
		{"Đi học", "di-hoc"},
		{"  React Basics  ", "react-basics"},
		{"hello__world", "hello-world"},
		{"", ""},
		{"---", ""},
		{"Course #1!", "course-1"},
	}
	for _, tc := range tests {
		if got := SlugifyName(tc.in); got != tc.want {
			t.Errorf("SlugifyName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
