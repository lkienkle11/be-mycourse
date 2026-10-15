package slug

import "testing"

func TestIsReserved(t *testing.T) {
	for _, s := range []string{"admin", "me", "api", "instructors", "user", "users"} {
		if !IsReserved(s) {
			t.Fatalf("%q should be reserved", s)
		}
	}
	for _, s := range []string{"nguyen-van-a", "admin-x7k92ab", "user-abc1234", ""} {
		if IsReserved(s) {
			t.Fatalf("%q should not be reserved", s)
		}
	}
}
