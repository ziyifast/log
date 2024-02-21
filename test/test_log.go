package main

import (
	"myTest/log"
	"os"
)

type User struct {
	Name string
	Age  int64
}

func main() {
	os.Setenv("DEBUG", "true")

	log.Infof("%s", "hello world")
	log.Errorf("%v", 5)
	user := &User{
		Name: "jackson",
		Age:  28,
	}
	log.Debugf("%v", user)
	log.Debugf("%+v", user)

}
