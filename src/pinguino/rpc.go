package pinguino

import (
	"os"
	"strconv"
)

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/pinguino-"
	s += strconv.Itoa(os.Getuid())
	return s
}
