package schema

import (
	"errors"
	"testing"
)

func TestResolveObjectUsesDeterministicMappings(t *testing.T) {
	workspace := &Workspace{
		Objects: []Object{
			{
				NameSingular:  "company",
				NamePlural:    "companies",
				LabelSingular: "Company",
				LabelPlural:   "Companies",
				UniversalID:   standardObjectIDs["company"],
			},
			{
				NameSingular:  "opportunity",
				NamePlural:    "opportunities",
				LabelSingular: "Deal",
				LabelPlural:   "Deals",
				UniversalID:   standardObjectIDs["deal"],
			},
			{
				NameSingular:  "person",
				NamePlural:    "people",
				LabelSingular: "Contact",
				LabelPlural:   "Contacts",
				UniversalID:   standardObjectIDs["person"],
				Fields: []Field{
					{Name: "email", Label: "Primary Email"},
					{Name: "phone", Label: "Phone"},
				},
			},
			{
				NameSingular:  "meeting",
				NamePlural:    "meetings",
				LabelSingular: "Meeting",
				LabelPlural:   "Meetings",
			},
			{
				NameSingular:  "note",
				NamePlural:    "notes",
				LabelSingular: "Note",
				LabelPlural:   "Notes",
				UniversalID:   standardObjectIDs["note"],
			},
			{
				NameSingular:  "task",
				NamePlural:    "tasks",
				LabelSingular: "Task",
				LabelPlural:   "Tasks",
				UniversalID:   standardObjectIDs["task"],
			},
		},
	}

	tests := []struct {
		noun string
		want string
	}{
		{noun: "person", want: "person"},
		{noun: "people", want: "person"},
		{noun: "company", want: "company"},
		{noun: "companies", want: "company"},
		{noun: "deal", want: "opportunity"},
		{noun: "deals", want: "opportunity"},
		{noun: "meeting", want: "meeting"},
		{noun: "note", want: "note"},
		{noun: "task", want: "task"},
	}

	for _, tc := range tests {
		resolved, err := workspace.ResolveObject(tc.noun)
		if err != nil {
			t.Fatalf("ResolveObject(%q) error = %v", tc.noun, err)
		}
		if resolved.Object.NameSingular != tc.want {
			t.Fatalf("ResolveObject(%q) = %q, want %q", tc.noun, resolved.Object.NameSingular, tc.want)
		}
	}
}

func TestResolveObjectMissingAndAmbiguous(t *testing.T) {
	tests := []struct {
		name      string
		workspace *Workspace
		noun      string
		targetErr error
	}{
		{
			name: "missing",
			workspace: &Workspace{
				Objects: []Object{{NameSingular: "company", NamePlural: "companies"}},
			},
			noun:      "person",
			targetErr: ErrObjectNotFound,
		},
		{
			name: "ambiguous alias match",
			workspace: &Workspace{
				Objects: []Object{
					{NameSingular: "contact", NamePlural: "contacts", LabelSingular: "Contact", LabelPlural: "Contacts"},
					{NameSingular: "contactProfile", NamePlural: "contactProfiles", LabelSingular: "Contacts", LabelPlural: "Contacts"},
				},
			},
			noun:      "contacts",
			targetErr: ErrObjectAmbiguous,
		},
	}

	for _, tc := range tests {
		_, err := tc.workspace.ResolveObject(tc.noun)
		if !errors.Is(err, tc.targetErr) {
			t.Fatalf("%s: ResolveObject(%q) error = %v, want %v", tc.name, tc.noun, err, tc.targetErr)
		}
	}
}

func TestResolveFieldUsesNormalizedNameAndLabel(t *testing.T) {
	workspace := &Workspace{
		Objects: []Object{
			{
				NameSingular: "person",
				NamePlural:   "people",
				UniversalID:  standardObjectIDs["person"],
				Fields: []Field{
					{Name: "email", Label: "Primary Email"},
					{Name: "mobilePhone", Label: "Mobile Phone"},
				},
			},
		},
	}

	tests := []struct {
		field string
		want  string
	}{
		{field: "email", want: "email"},
		{field: "primary email", want: "email"},
		{field: "mobile-phone", want: "mobilePhone"},
	}

	for _, tc := range tests {
		field, err := workspace.ResolveField("person", tc.field)
		if err != nil {
			t.Fatalf("ResolveField(%q) error = %v", tc.field, err)
		}
		if field.Name != tc.want {
			t.Fatalf("ResolveField(%q) = %q, want %q", tc.field, field.Name, tc.want)
		}
	}
}

func TestResolveFieldMissingAndAmbiguous(t *testing.T) {
	workspace := &Workspace{
		Objects: []Object{
			{
				NameSingular: "person",
				NamePlural:   "people",
				UniversalID:  standardObjectIDs["person"],
				Fields: []Field{
					{Name: "email", Label: "Email"},
					{Name: "primaryEmail", Label: "Email"},
				},
			},
		},
	}

	if _, err := workspace.ResolveField("person", "phone"); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("ResolveField(missing) error = %v, want %v", err, ErrFieldNotFound)
	}

	if _, err := workspace.ResolveField("person", "email"); !errors.Is(err, ErrFieldAmbiguous) {
		t.Fatalf("ResolveField(ambiguous) error = %v, want %v", err, ErrFieldAmbiguous)
	}
}
