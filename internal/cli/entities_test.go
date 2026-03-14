package cli

import (
	"reflect"
	"testing"
)

func TestFiltersForEntityQuery(t *testing.T) {
	tests := []struct {
		name   string
		entity entityDef
		query  string
		want   []string
	}{
		{
			name:   "person uses name and email probes",
			entity: entityDefs[0],
			query:  "Ada",
			want: []string{
				"name.firstName[ilike]:%Ada%",
				"name.lastName[ilike]:%Ada%",
				"emails.primaryEmail[ilike]:%Ada%",
			},
		},
		{
			name:   "person email query uses email only",
			entity: entityDefs[0],
			query:  "ada@example.com",
			want:   []string{"emails.primaryEmail[ilike]:%ada@example.com%"},
		},
		{
			name:   "company searches name and domain",
			entity: entityDefs[1],
			query:  "Acme",
			want: []string{
				"name[ilike]:%Acme%",
				"domainName.primaryLinkUrl[ilike]:%Acme%",
			},
		},
		{
			name:   "deal searches name",
			entity: entityDefs[2],
			query:  "Expansion",
			want:   []string{"name[ilike]:%Expansion%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filtersForEntityQuery(tt.entity, tt.query)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("filtersForEntityQuery() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
