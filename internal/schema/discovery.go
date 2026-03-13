package schema

import (
	"context"
	"sort"
	"strings"

	"github.com/jv/twenty-crm-cli/internal/client"
)

type metadataFetcher interface {
	MetadataObjects(context.Context) ([]client.MetadataObject, error)
}

func Discover(ctx context.Context, fetcher metadataFetcher) (*Workspace, error) {
	objects, err := fetcher.MetadataObjects(ctx)
	if err != nil {
		return nil, err
	}

	workspace := &Workspace{Objects: make([]Object, 0, len(objects))}
	for _, raw := range objects {
		if !raw.IsActive {
			continue
		}
		workspace.Objects = append(workspace.Objects, normalizeObject(raw))
	}

	sort.Slice(workspace.Objects, func(i, j int) bool {
		return workspace.Objects[i].NameSingular < workspace.Objects[j].NameSingular
	})

	return workspace, nil
}

func normalizeObject(raw client.MetadataObject) Object {
	fields := make([]Field, 0, len(raw.Fields))
	for _, field := range raw.Fields {
		if !field.IsActive {
			continue
		}
		fields = append(fields, normalizeField(field))
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return Object{
		ID:                strings.TrimSpace(raw.ID),
		UniversalID:       strings.TrimSpace(raw.UniversalIdentifier),
		NameSingular:      strings.TrimSpace(raw.NameSingular),
		NamePlural:        strings.TrimSpace(raw.NamePlural),
		LabelSingular:     strings.TrimSpace(raw.LabelSingular),
		LabelPlural:       strings.TrimSpace(raw.LabelPlural),
		Description:       strings.TrimSpace(raw.Description),
		Fields:            fields,
		LabelIdentifierID: strings.TrimSpace(raw.LabelIdentifierFieldMetadataID),
		ImageIdentifierID: strings.TrimSpace(raw.ImageIdentifierFieldMetadataID),
	}
}

func normalizeField(raw client.MetadataField) Field {
	field := Field{
		ID:          strings.TrimSpace(raw.ID),
		UniversalID: strings.TrimSpace(raw.UniversalIdentifier),
		Type:        strings.TrimSpace(raw.Type),
		Name:        strings.TrimSpace(raw.Name),
		Label:       strings.TrimSpace(raw.Label),
		Description: strings.TrimSpace(raw.Description),
	}

	if raw.Relation != nil {
		field.Relation = &Relation{
			Type:               strings.TrimSpace(raw.Relation.Type),
			SourceObjectID:     strings.TrimSpace(raw.Relation.SourceObject.ID),
			SourceObjectName:   strings.TrimSpace(raw.Relation.SourceObject.NameSingular),
			SourceObjectPlural: strings.TrimSpace(raw.Relation.SourceObject.NamePlural),
			TargetObjectID:     strings.TrimSpace(raw.Relation.TargetObject.ID),
			TargetObjectName:   strings.TrimSpace(raw.Relation.TargetObject.NameSingular),
			TargetObjectPlural: strings.TrimSpace(raw.Relation.TargetObject.NamePlural),
			SourceFieldID:      strings.TrimSpace(raw.Relation.SourceField.ID),
			SourceFieldName:    strings.TrimSpace(raw.Relation.SourceField.Name),
			TargetFieldID:      strings.TrimSpace(raw.Relation.TargetField.ID),
			TargetFieldName:    strings.TrimSpace(raw.Relation.TargetField.Name),
		}
	}

	return field
}
