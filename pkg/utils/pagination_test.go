package utils

import "testing"

func TestNewPagination_NormalizesValues(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		limit        int
		defaultLimit int
		maxLimit     int
		want         Pagination
	}{
		{
			name:         "valid values",
			page:         3,
			limit:        10,
			defaultLimit: 20,
			maxLimit:     100,
			want:         Pagination{Page: 3, Limit: 10, Offset: 20},
		},
		{
			name:         "invalid page and limit use defaults",
			page:         0,
			limit:        0,
			defaultLimit: 20,
			maxLimit:     100,
			want:         Pagination{Page: 1, Limit: 20, Offset: 0},
		},
		{
			name:         "limit over max uses default",
			page:         2,
			limit:        101,
			defaultLimit: 20,
			maxLimit:     100,
			want:         Pagination{Page: 2, Limit: 20, Offset: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPagination(tt.page, tt.limit, tt.defaultLimit, tt.maxLimit)
			if got != tt.want {
				t.Fatalf("NewPagination() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
