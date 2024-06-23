package helper

import (
	"cmp"
	"errors"
)

func Contains[T comparable](slice []T, item T) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func DeleteByIndex[T any](s []T, index uint) []T {

	if index >= uint(len(s)) {
		panic(errors.New("index out of bounds"))
	}

	if index == 0 {
		ret := make([]T, len(s)-1)
		copy(ret, s[1:])
		return ret
	}

	if index == uint(len(s)-1) {
		ret := make([]T, len(s)-1)
		copy(ret, s[:len(s)-1])
		return ret
	}

	slice3 := make([]T, 0, len(s)-1)

	/*
		ensures a memory leak where the backing array of original size is still referenced
		does not occur
	*/
	slice1 := make([]T, index)
	copy(slice1, s)

	slice2 := make([]T, len(s)-int(index+1))
	copy(slice2, s[int(index+1):])

	slice3 = append(slice3, slice1...)
	slice3 = append(slice3, slice2...)

	return slice3
}

func IsLte[T cmp.Ordered](lowVal, highVal T, checkEqual bool) bool {

	if checkEqual && lowVal == highVal {
		return true
	}

	return lowVal < highVal
}

func IsGte[T cmp.Ordered](lowVal, highVal T, checkEqual bool) bool {
	if checkEqual && lowVal == highVal {
		return true
	}
	return highVal > lowVal
}

func IsBetween[T cmp.Ordered](lowRange, highRange, val T, lte, gte bool) bool {
	return IsGte[T](lowRange, val, gte) && IsLte[T](val, highRange, lte)
}

type Set[T comparable] map[T]struct{}

func (s *Set[T]) Insert(item T) {
	(*s)[item] = struct{}{}
}

func (s *Set[T]) Delete(item T) {
	delete(*s, item)
}

func (s *Set[T]) Has(item T) bool {
	_, ok := (*s)[item]
	return ok
}
