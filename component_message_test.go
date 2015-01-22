package casper_test

import (
	"fmt"
	"testing"
	"encoding/base64"
	
	"github.com/gogap/casper"
)

func TestNewComponentMessage(t *testing.T) {
	msg, _ := casper.NewComponentMessage("entrance")
	bmsg, _ := msg.Serialize()
	fmt.Println(string(bmsg))
}

func TestBase64(t *testing.T) {
	str := "ewoJImNlbGxwaG9uZSI6IjE1MjE1MjE4NzE1MjU4NSIsCgkicGFzc3dvcmQiOiIxNTIxNTIxODc2NTIyIgp9"
	dst, err := base64.StdEncoding.DecodeString(str)
	fmt.Println(string(dst), err)
}

