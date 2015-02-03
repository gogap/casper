package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/gogap/casper"
	"github.com/gogap/casper/utils"
)

func main() {
	flag.Parse()

	casper.BuildComponent("component.conf.example")
	casper.GetComponentByName("com1").SetHandler(handler).Run()
	utils.IamWorking()
}

func handler(msg *casper.Payload) (result interface{}, err error) {
	fmt.Println(">>>", msg)

	cookie := http.Cookie{
		Name:    "testcookie",
		Value:   "cookievalue",
		Expires: time.Now().Add(60 * time.Second),
		MaxAge:  60}

	header := casper.NameValue{"component1", "value"}

	cookies := []interface{}{cookie}
	headers := []interface{}{header}
	msg.SetCommand(casper.CMD_HTTP_COOKIES_SET, cookies)
	msg.SetCommand(casper.CMD_HTTP_HEADERS_SET, headers)

	rst := &struct {
		Name string
		Age  int
	}{
		Name: "小明",
		Age:  6}

	return rst, nil
}
