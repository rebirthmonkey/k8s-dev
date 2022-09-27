package utils

import "github.com/thoas/go-funk"

// IsNotEmpty checks if the specified object is empty
func IsNotEmpty(obj interface{}) bool {
	return !funk.IsEmpty(obj)
}

func Equal(obj1, obj2 interface{}) bool {
	return funk.Equal(obj1, obj2) || (funk.IsEmpty(obj1) && funk.IsEmpty(obj2))
}
