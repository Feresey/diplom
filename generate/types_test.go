package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeTwoSortedIntArrays(t *testing.T) {
	tests := []struct {
		name     string
		arr1     []int
		arr2     []int
		expected []int
	}{
		{
			name:     "Test merging two sorted int arrays",
			arr1:     []int{1, 3, 5},
			arr2:     []int{2, 4, 6},
			expected: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:     "Test merging arrays with duplicate elements",
			arr1:     []int{1, 2, 3, 4},
			arr2:     []int{3, 4, 5, 6},
			expected: []int{1, 2, 3, 3, 4, 4, 5, 6},
		},
		{
			name:     "Test merging arrays with negative numbers",
			arr1:     []int{-10, -5, 0},
			arr2:     []int{-8, -3, 7},
			expected: []int{-10, -8, -5, -3, 0, 7},
		},
		{
			name:     "Test merging arrays with one empty array",
			arr1:     []int{1, 2, 3},
			arr2:     []int{},
			expected: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeTwoSortedArrays(tt.arr1, tt.arr2)
			assert.Equal(t, tt.expected, result, "The merged array is not correct")
		})
	}
}

func TestMergeTwoSortedStringArrays(t *testing.T) {
	tests := []struct {
		name     string
		arr1     []string
		arr2     []string
		expected []string
	}{
		{
			name:     "Test merging two sorted string arrays",
			arr1:     []string{"a", "c", "e"},
			arr2:     []string{"b", "d", "f"},
			expected: []string{"a", "b", "c", "d", "e", "f"},
		},
		{
			name:     "Test merging arrays with duplicate elements",
			arr1:     []string{"apple", "banana", "cherry"},
			arr2:     []string{"cherry", "orange", "pear"},
			expected: []string{"apple", "banana", "cherry", "cherry", "orange", "pear"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeTwoSortedArrays(tt.arr1, tt.arr2)
			assert.Equal(t, tt.expected, result, "The merged array is not correct")
		})
	}
}
