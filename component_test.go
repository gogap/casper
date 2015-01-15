package casper_test

import (
	"fmt"
	"testing"

	"github.com/gogap/casper"
)

func TestNewComponent(t *testing.T) {
	com, _ := casper.NewComponent("com1", "this is com1", "zmq", "tcp://127.0.0.1:5001")
	fmt.Println(com)
}
