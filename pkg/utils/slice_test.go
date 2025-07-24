package utils

import (
	"reflect"
	"testing"
)

func TestContainsString(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		str   string
		want  bool
	}{
		{
			name:  "slice is empty",
			slice: []string{},
			str:   "a",
			want:  false,
		},
		{
			name:  "searching object exists the slice",
			slice: []string{"a", "b"},
			str:   "a",
			want:  true,
		},
		{
			name:  "searching object doesn't exist the slice",
			slice: []string{"a", "b"},
			str:   "c",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.slice, tt.str); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveString(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		str   string
		want  []string
	}{
		{
			name:  "removing object exists the slice",
			slice: []string{"a", "b", "c"},
			str:   "b",
			want:  []string{"a", "c"},
		},
		{
			name:  "remove object doesn't exist in the slice",
			slice: []string{"a", "b", "c"},
			str:   "d",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "the slice is empty",
			slice: []string{},
			str:   "d",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveString(tt.slice, tt.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveString() = %v, want %v", got, tt.want)
			}
		})
	}
}
