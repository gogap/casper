package utils

import (
	"fmt"
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

func IsStructArray(s interface{}) bool {
	v := reflect.ValueOf(s)

	if v.Kind() == reflect.Invalid {
		return false
	}

	if v.Kind() != reflect.Slice {
		return false
	}

	if v.Len() < 1 {
		return false
	}

	v = v.Index(0)

	return IsStruct(v)
}
