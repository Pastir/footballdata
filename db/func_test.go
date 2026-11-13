package db

import "testing"

func TestCastInterfaceString(t *testing.T) {
	bTrue := true
	bFalse := false
	var bNil *bool

	if got := CastInterfaceString(&bTrue); got != "'t'" {
		t.Fatalf("expected 't', got %q", got)
	}
	if got := CastInterfaceString(&bFalse); got != "'f'" {
		t.Fatalf("expected 'f', got %q", got)
	}
	if got := CastInterfaceString(bNil); got != "NULL" {
		t.Fatalf("expected NULL, got %q", got)
	}
	if got := CastInterfaceString(true); got != "'t'" {
		t.Fatalf("expected 't', got %q", got)
	}
	if got := CastInterfaceString(false); got != "'f'" {
		t.Fatalf("expected 'f', got %q", got)
	}
}

