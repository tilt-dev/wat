package slices

func Contains(arr []string, s string) bool {
	for _, elem := range arr {
		if elem == s {
			return true
		}
	}
	return false
}

// Does arrA contain all elements of arrB? (Ignores order and # of occurences of elems.)
func ContainsAllElems(arrA, arrB []string) bool {
	for _, elemB := range arrB {
		if !Contains(arrA, elemB) {
			return false
		}
	}
	return true
}
