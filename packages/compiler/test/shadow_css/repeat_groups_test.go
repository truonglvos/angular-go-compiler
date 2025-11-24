package shadow_css_test

import (
	"ngc-go/packages/compiler/src/css"
	"reflect"
	"testing"
	"unsafe"
)

// isSameSlice checks if two slices are the same reference (same underlying array)
func isSameSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	// Get pointer to first element of each slice
	ptrA := unsafe.Pointer(&a[0])
	ptrB := unsafe.Pointer(&b[0])
	return ptrA == ptrB
}

func TestRepeatGroups(t *testing.T) {
	t.Run("should do nothing if multiples is 0", func(t *testing.T) {
		groups := [][]string{
			{"a1", "b1", "c1"},
			{"a2", "b2", "c2"},
		}
		originalGroups := make([][]string, len(groups))
		for i := range groups {
			originalGroups[i] = make([]string, len(groups[i]))
			copy(originalGroups[i], groups[i])
		}
		css.RepeatGroups(&groups, 0)
		if !reflect.DeepEqual(groups, originalGroups) {
			t.Errorf("Expected groups to remain unchanged, got %v", groups)
		}
	})

	t.Run("should do nothing if multiples is 1", func(t *testing.T) {
		groups := [][]string{
			{"a1", "b1", "c1"},
			{"a2", "b2", "c2"},
		}
		originalGroups := make([][]string, len(groups))
		for i := range groups {
			originalGroups[i] = make([]string, len(groups[i]))
			copy(originalGroups[i], groups[i])
		}
		css.RepeatGroups(&groups, 1)
		if !reflect.DeepEqual(groups, originalGroups) {
			t.Errorf("Expected groups to remain unchanged, got %v", groups)
		}
	})

	t.Run("should add clones of the original groups if multiples is greater than 1", func(t *testing.T) {
		group1 := []string{"a1", "b1", "c1"}
		group2 := []string{"a2", "b2", "c2"}
		groups := [][]string{group1, group2}
		css.RepeatGroups(&groups, 3)
		expected := [][]string{group1, group2, group1, group2, group1, group2}
		if len(groups) != len(expected) {
			t.Fatalf("Expected %d groups, got %d", len(expected), len(groups))
		}
		// Check that first two are the original references
		if !isSameSlice(groups[0], group1) {
			t.Error("Expected groups[0] to be the original group1 reference")
		}
		if !isSameSlice(groups[1], group2) {
			t.Error("Expected groups[1] to be the original group2 reference")
		}
		// Check that clones are not the same references
		if isSameSlice(groups[2], group1) {
			t.Error("Expected groups[2] to be a clone, not the original reference")
		}
		if isSameSlice(groups[3], group2) {
			t.Error("Expected groups[3] to be a clone, not the original reference")
		}
		if isSameSlice(groups[4], group1) {
			t.Error("Expected groups[4] to be a clone, not the original reference")
		}
		if isSameSlice(groups[5], group2) {
			t.Error("Expected groups[5] to be a clone, not the original reference")
		}
		// Check that clones have the same content
		if !reflect.DeepEqual(groups[2], group1) {
			t.Errorf("Expected groups[2] to equal group1, got %v", groups[2])
		}
		if !reflect.DeepEqual(groups[3], group2) {
			t.Errorf("Expected groups[3] to equal group2, got %v", groups[3])
		}
		if !reflect.DeepEqual(groups[4], group1) {
			t.Errorf("Expected groups[4] to equal group1, got %v", groups[4])
		}
		if !reflect.DeepEqual(groups[5], group2) {
			t.Errorf("Expected groups[5] to equal group2, got %v", groups[5])
		}
	})
}

