package casper

import (
	"fmt"
	"testing"
)

func TestSessionSetByte(t *testing.T) {
	SessionSetByte("demo", "key", []byte("val"))
}

func TestSessionGetByte(t *testing.T) {
	rst, _ := SessionGetByte("demo", "key")
	fmt.Println(string(rst))
}
