package utils

import (
	"reflect"
	"time"
)

func IamWorking() {
	for {
		time.Sleep(1 * time.Second)
	}
}

func IsStruct(s interface{}) bool {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Invalid {
		return false
	}

	return v.Kind() == reflect.Struct
}
