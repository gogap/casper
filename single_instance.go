package casper

import (
	"fmt"
	"io/ioutil"
	"syscall"
)

func SingleInstane(pidfile string) {
	if e := lockPidFile(pidfile); e != nil {
		pid, _ := ioutil.ReadFile(pidfile)
		panic(fmt.Errorf("process already run: %v", string(pid)))
	}

}

func lockPidFile(pidfile string) error {
	fd, e := syscall.Open(pidfile, syscall.O_CREAT|syscall.O_RDWR, 0777)
	if e != nil {
		return e
	}

	e = syscall.Flock(fd, syscall.LOCK_NB|syscall.LOCK_EX)
	if e != nil {
		return e
	}

	e = syscall.Ftruncate(fd, 0)
	if e != nil {
		return e
	}

	_, e = syscall.Write(fd, []byte(fmt.Sprintf("%d", syscall.Getpid())))
	if e != nil {
		return e
	}

	return nil
}

/*
func main() {
	SingleInstane("/tmp/pid")
	time.Sleep(100 * time.Second)
}
*/
