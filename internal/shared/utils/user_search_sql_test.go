package utils

import "testing"

func TestUserDisplayNameEmailSearchSQL(t *testing.T) {
	clause, args := UserDisplayNameEmailSearchSQL("")
	if clause != "" || args != nil {
		t.Fatalf("expected empty search clause, got %q args=%v", clause, args)
	}

	clause, args = UserDisplayNameEmailSearchSQL("  alice  ")
	if clause == "" || args["search"] != "%alice%" {
		t.Fatalf("expected ilike clause with trimmed search, got %q args=%v", clause, args)
	}

	clause, args = UserDisplayNameEmailSearchSQL("bob@example.com")
	if clause == "" || args["search"] != "%bob@example.com%" {
		t.Fatalf("expected email search clause, got %q args=%v", clause, args)
	}
}
