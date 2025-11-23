package utils

import (
	"regexp"
	"testing"
)

func TestGenerateRandomId_Length(t *testing.T) {
	id, err := GenerateRandomId()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != 10 {
		t.Errorf("expected length 10, got %d", len(id))
	}
}

func TestGenerateRandomId_Charset(t *testing.T) {
	id, err := GenerateRandomId()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", id)
	if !match {
		t.Errorf("generated ID contains invalid characters: %s", id)
	}
}
