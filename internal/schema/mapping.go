package schema

import (
	"slices"
	"sort"
	"strings"
)

var standardObjectIDs = map[string]string{
	"person":  "20202020-c428-4f40-b6f3-86091511c41c",
	"company": "20202020-cff5-4682-8bf9-069169e08279",
	"deal":    "20202020-9549-49dd-b2b2-883999db8938",
	"task":    "20202020-1b1b-4b3b-8b1b-7f8d6a1d7d5c",
	"note":    "20202020-1f25-43fe-8b00-af212fdde824",
}

type domainRule struct {
	domain       string
	primaryNames []string
	aliases      []string
}

var domainRules = []domainRule{
	{domain: "person", primaryNames: []string{"person", "people"}, aliases: []string{"contact", "contacts"}},
	{domain: "company", primaryNames: []string{"company", "companies"}},
	{domain: "deal", primaryNames: []string{"opportunity", "opportunities"}, aliases: []string{"deal", "deals"}},
	{domain: "meeting", primaryNames: []string{"meeting", "meetings"}},
	{domain: "note", primaryNames: []string{"note", "notes"}},
	{domain: "task", primaryNames: []string{"task", "tasks"}},
}

func (w *Workspace) ResolveObject(noun string) (ResolvedObject, error) {
	rule := ruleFor(noun)

	if resolved, ok := w.resolveByStandardID(rule.domain); ok {
		return ResolvedObject{Domain: rule.domain, Object: resolved}, nil
	}

	for _, names := range [][]string{rule.primaryNames, rule.aliases} {
		if len(names) == 0 {
			continue
		}
		if matches := w.matchObjects(names); len(matches) == 1 {
			return ResolvedObject{Domain: rule.domain, Object: matches[0]}, nil
		} else if len(matches) > 1 {
			return ResolvedObject{}, &ObjectResolutionError{
				Domain:     rule.domain,
				Candidates: objectNames(matches),
				Err:        ErrObjectAmbiguous,
			}
		}
	}

	return ResolvedObject{}, &ObjectResolutionError{
		Domain: rule.domain,
		Err:    ErrObjectNotFound,
	}
}

func (w *Workspace) ResolveField(noun, fieldName string) (Field, error) {
	resolved, err := w.ResolveObject(noun)
	if err != nil {
		return Field{}, err
	}

	fieldMatches := matchFields(resolved.Object.Fields, fieldName)
	switch len(fieldMatches) {
	case 0:
		return Field{}, &FieldResolutionError{
			Object: resolved.Object.NameSingular,
			Field:  fieldName,
			Err:    ErrFieldNotFound,
		}
	case 1:
		return fieldMatches[0], nil
	default:
		return Field{}, &FieldResolutionError{
			Object:     resolved.Object.NameSingular,
			Field:      fieldName,
			Candidates: fieldNames(fieldMatches),
			Err:        ErrFieldAmbiguous,
		}
	}
}

func ruleFor(noun string) domainRule {
	normalized := normalizeKey(noun)
	for _, rule := range domainRules {
		if normalized == rule.domain {
			return rule
		}
		if slices.Contains(rule.primaryNames, normalized) || slices.Contains(rule.aliases, normalized) {
			return rule
		}
	}

	return domainRule{
		domain:       normalized,
		primaryNames: []string{normalized},
	}
}

func (w *Workspace) resolveByStandardID(domain string) (Object, bool) {
	wantID, ok := standardObjectIDs[domain]
	if !ok {
		return Object{}, false
	}

	for _, object := range w.Objects {
		if object.UniversalID == wantID {
			return object, true
		}
	}

	return Object{}, false
}

func (w *Workspace) matchObjects(names []string) []Object {
	var matches []Object
	normalizedNames := make([]string, 0, len(names))
	for _, name := range names {
		normalizedNames = append(normalizedNames, normalizeKey(name))
	}

	for _, object := range w.Objects {
		candidates := []string{
			object.NameSingular,
			object.NamePlural,
			object.LabelSingular,
			object.LabelPlural,
		}
		for _, candidate := range candidates {
			if slices.Contains(normalizedNames, normalizeKey(candidate)) {
				matches = append(matches, object)
				break
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].NameSingular < matches[j].NameSingular
	})

	return matches
}

func matchFields(fields []Field, fieldName string) []Field {
	var matches []Field
	normalized := normalizeKey(fieldName)
	for _, field := range fields {
		if normalizeKey(field.Name) == normalized || normalizeKey(field.Label) == normalized {
			matches = append(matches, field)
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

	return matches
}

func normalizeKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	return replacer.Replace(value)
}

func joinNames(names []string) string {
	return strings.Join(names, ", ")
}

func objectNames(objects []Object) []string {
	names := make([]string, 0, len(objects))
	for _, object := range objects {
		names = append(names, object.NameSingular)
	}
	return names
}

func fieldNames(fields []Field) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}
