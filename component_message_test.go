package casper_test

import (
	"fmt"
	"testing"

	"github.com/gogap/casper"
)

func TestNewComponentMessage(t *testing.T) {
	msg, _ := casper.NewComponentMessage("entrance")
	bmsg, _ := msg.Serialize()
	fmt.Println(string(bmsg))
}
