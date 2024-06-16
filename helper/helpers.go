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

func DeleteByIndex[T any](s []T, index uint) (error, []T) {

	if index >= uint(len(s)) {
		return errors.New("index out of bounds"), nil
	}

	if index == 0 {
		return nil, s[1:]
	}

	if index == uint(len(s)-1) {
		return nil, s[:len(s)-1]
	}

	/*
		ensures a memory leak where the backing array of original size is still referenced
		does not occur
	*/
	slice1 := make([]T, index)
	copy(slice1, s)

	slice2 := s[index+1:] // memory leak cannot occur in this scenario

	return nil, append(slice1, slice2...)
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
