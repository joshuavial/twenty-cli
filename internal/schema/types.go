package schema

import "errors"

var (
	ErrObjectNotFound  = errors.New("schema object not found")
	ErrObjectAmbiguous = errors.New("schema object mapping is ambiguous")
	ErrFieldNotFound   = errors.New("schema field not found")
	ErrFieldAmbiguous  = errors.New("schema field mapping is ambiguous")
)

type Workspace struct {
	Objects []Object
}

type Object struct {
	ID                string
	UniversalID       string
	NameSingular      string
	NamePlural        string
	LabelSingular     string
	LabelPlural       string
	Description       string
	Fields            []Field
	LabelIdentifierID string
	ImageIdentifierID string
}

type Field struct {
	ID          string
	UniversalID string
	Type        string
	Name        string
	Label       string
	Description string
	Relation    *Relation
}

type Relation struct {
	Type               string
	SourceObjectID     string
	SourceObjectName   string
	SourceObjectPlural string
	TargetObjectID     string
	TargetObjectName   string
	TargetObjectPlural string
	SourceFieldID      string
	SourceFieldName    string
	TargetFieldID      string
	TargetFieldName    string
}

type ResolvedObject struct {
	Domain string
	Object Object
}

type ObjectResolutionError struct {
	Domain     string
	Candidates []string
	Err        error
}

func (e *ObjectResolutionError) Error() string {
	if e == nil {
		return ""
	}
	if len(e.Candidates) == 0 {
		return e.Err.Error() + ": " + e.Domain
	}
	return e.Err.Error() + ": " + e.Domain + " (" + joinNames(e.Candidates) + ")"
}

func (e *ObjectResolutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type FieldResolutionError struct {
	Object     string
	Field      string
	Candidates []string
	Err        error
}

func (e *FieldResolutionError) Error() string {
	if e == nil {
		return ""
	}
	if len(e.Candidates) == 0 {
		return e.Err.Error() + ": " + e.Object + "." + e.Field
	}
	return e.Err.Error() + ": " + e.Object + "." + e.Field + " (" + joinNames(e.Candidates) + ")"
}

func (e *FieldResolutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
