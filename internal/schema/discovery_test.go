package schema

import (
	"context"
	"testing"

	"github.com/jv/twenty-crm-cli/internal/client"
)

type metadataFetcherStub struct {
	objects []client.MetadataObject
	err     error
}

func (s metadataFetcherStub) MetadataObjects(context.Context) ([]client.MetadataObject, error) {
	return s.objects, s.err
}

func TestDiscoverNormalizesMetadata(t *testing.T) {
	workspace, err := Discover(context.Background(), metadataFetcherStub{
		objects: []client.MetadataObject{
			{
				ID:                             " obj_1 ",
				UniversalIdentifier:            " u_person ",
				NameSingular:                   " person ",
				NamePlural:                     " people ",
				LabelSingular:                  " Person ",
				LabelPlural:                    " People ",
				Description:                    " Contacts ",
				IsActive:                       true,
				LabelIdentifierFieldMetadataID: " fld_1 ",
				ImageIdentifierFieldMetadataID: " fld_2 ",
				Fields: []client.MetadataField{
					{
						ID:                  " fld_1 ",
						UniversalIdentifier: " u_email ",
						Type:                " EMAIL ",
						Name:                " email ",
						Label:               " Email ",
						Description:         " Primary email ",
						IsActive:            true,
					},
					{
						ID:       "fld_ignored",
						Name:     "archived",
						Label:    "Archived",
						IsActive: false,
					},
				},
			},
			{
				ID:           "obj_2",
				NameSingular: "inactive",
				IsActive:     false,
			},
		},
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(workspace.Objects) != 1 {
		t.Fatalf("len(Objects) = %d, want 1", len(workspace.Objects))
	}

	object := workspace.Objects[0]
	if object.NameSingular != "person" || object.NamePlural != "people" {
		t.Fatalf("object names = %#v", object)
	}
	if object.LabelIdentifierID != "fld_1" || object.ImageIdentifierID != "fld_2" {
		t.Fatalf("identifier fields = %#v", object)
	}
	if len(object.Fields) != 1 {
		t.Fatalf("len(Fields) = %d, want 1", len(object.Fields))
	}
	if object.Fields[0].Name != "email" || object.Fields[0].Label != "Email" {
		t.Fatalf("field = %#v", object.Fields[0])
	}
}

func TestDiscoverNormalizesRelations(t *testing.T) {
	workspace, err := Discover(context.Background(), metadataFetcherStub{
		objects: []client.MetadataObject{
			{
				ID:            "obj_1",
				NameSingular:  "meeting",
				NamePlural:    "meetings",
				LabelSingular: "Meeting",
				LabelPlural:   "Meetings",
				IsActive:      true,
				Fields: []client.MetadataField{
					{
						ID:       "fld_company",
						Name:     "company",
						Label:    "Company",
						Type:     "RELATION",
						IsActive: true,
						Relation: &client.MetadataRelation{
							Type: "MANY_TO_ONE",
							SourceObject: client.MetadataObjectRef{
								ID:           " obj_1 ",
								NameSingular: " meeting ",
								NamePlural:   " meetings ",
							},
							TargetObject: client.MetadataObjectRef{
								ID:           " obj_2 ",
								NameSingular: " company ",
								NamePlural:   " companies ",
							},
							SourceField: client.MetadataFieldRef{ID: " fld_company ", Name: " company "},
							TargetField: client.MetadataFieldRef{ID: " fld_meetings ", Name: " meetings "},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	relation := workspace.Objects[0].Fields[0].Relation
	if relation == nil {
		t.Fatal("Relation = nil, want relation")
	}
	if relation.SourceObjectName != "meeting" || relation.TargetObjectName != "company" {
		t.Fatalf("relation = %#v", relation)
	}
	if relation.SourceFieldName != "company" || relation.TargetFieldName != "meetings" {
		t.Fatalf("relation = %#v", relation)
	}
}
