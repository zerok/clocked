package form

import "testing"

func TestValidation(t *testing.T) {
	f := NewForm([]Field{Field{
		Code:       "a",
		Label:      "a",
		Value:      "",
		IsRequired: true,
	}})

	if f.Validate() {
		t.Fatalf("Validate should already return if the validation was successful or not")
	}
	if f.fields[0].IsValid() {
		t.Fatalf("The field should have been marked as invalid. Was %v instead", f.fields[0])
	}

	if f.IsValid() {
		t.Fatalf("The whole form should indicate that it's not valid")
	}
}
