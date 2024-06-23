package helper

import (
	"testing"
)

func TestContains(t *testing.T) {
	contains := Contains[int]

	testSlice := []int{
		1, 25, 71, 89, 34, 108,
	}

	if contains(testSlice, 99) {
		t.Error("recognized number not in testSlice")
	}

	if !contains(testSlice, 108) {
		t.Error("failed to recognize number in testSlice")
	}
}

func TestDeleteByIndex(t *testing.T) {

	stringSlice := []string{
		"data", "test", "water", "benchai", "huan", "success",
	}

	del := DeleteByIndex[string]

	t.Run("index out of bounds", func(t *testing.T) {
		defer func(t *testing.T) {
			if r := recover(); r == nil {
				t.Error("did not panic for out of bounds request")
			}
		}(t)

		del(stringSlice, 10)
	})

	tests := []struct {
		index        int
		currentSlice []string
		resultSlice  []string
		name         string
	}{
		{
			index:        2,
			currentSlice: stringSlice,
			resultSlice:  []string{"data", "test", "benchai", "huan", "success"},
			name:         "delete middle",
		},
		{
			index:        0,
			currentSlice: stringSlice,
			resultSlice:  []string{"test", "water", "benchai", "huan", "success"},
			name:         "delete first",
		},
		{
			index:        5,
			currentSlice: stringSlice,
			resultSlice:  []string{"data", "test", "water", "benchai", "huan"},
			name:         "delete last",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjusted := del(tt.currentSlice, uint(tt.index))
			for index := range adjusted {
				if tt.resultSlice[index] != adjusted[index] {
					t.Error("expected given slice: ", adjusted, "to match: ", tt.resultSlice)
				}
			}

			if cap(adjusted) != len(tt.resultSlice) {
				t.Errorf("expected capacity to be %d, received %d", cap(adjusted), len(tt.resultSlice))
			}
		})
	}

	t.Run("test delete all", func(t *testing.T) {

		defer func(t *testing.T) {
			if r := recover(); r != nil {
				t.Error("had to recover")
			}
		}(t)

		startLength := len(stringSlice)

		newSlice := del(stringSlice, 2)
		newSlice = del(newSlice, 1)

		if len(newSlice) != startLength-2 {
			t.Error("failed to remove 2 indices sequentially")
		}

		length := len(newSlice)

		for range len(newSlice) {
			newSlice = del(newSlice, 0)

			length--

			if length != len(newSlice) && cap(newSlice) != length {
				t.Errorf("string has improper length or capacity when doing sequenctail removal")
			}
		}
	})
}

func TestIsLte(t *testing.T) {
	lte := IsLte[int]

	if !lte(-10, 100, false) {
		t.Error("did not detect that -10 is < 100")
	}

	if !lte(-10, -10, true) {
		t.Error("did not detect that -10 is <= -10")
	}

	if lte(-10, -100, false) {
		t.Error("did not detect that -10 is > -100")
	}
}

func TestIsGte(t *testing.T) {
	gte := IsGte[int]

	if !gte(-10, 100, false) {
		t.Error("did not detect that 100 is > 100")
	}

	if !gte(10, 10, true) {
		t.Error("did not detect that 10 is <= 10")
	}

	if gte(-10, -100, false) {
		t.Error("did not detect that -10 is > -100")
	}
}

func TestIsBetween(t *testing.T) {
	bte := IsBetween[int]

	if !bte(0, 100, 20, false, false) {
		t.Error("failed to detect value 20 that is between 0 and 100")
	}

	if !bte(0, 0, 0, true, true) {
		t.Error("failed to detect value that 0 is between / equal to 0 and 0")
	}

	if bte(10, 100, -10, true, true) {
		t.Error("failed to detect value that -10 is not between 10 and 100")
	}
}

func getSet() Set[uint8] {
	dataSet := make(Set[uint8], 10)
	return dataSet
}

func TestSet_Has(t *testing.T) {
	set := Set[uint8]{
		108: struct{}{},
	}

	if !set.Has(108) {
		t.Fatal("has functionality failed")
	}
}

func TestSet_Insert(t *testing.T) {
	set := getSet()
	set.Insert(10)

	if !set.Has(10) {
		t.Fatal("insert failed")
	}
}

func TestSet_Delete(t *testing.T) {
	set := getSet()
	set.Insert(10)
	set.Delete(10)

	if set.Has(10) {
		t.Fatal("failed to delete")
	}
}
