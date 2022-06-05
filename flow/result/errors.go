package result

import (
	"fmt"
	"log"
)

// RecoverToErrorVar recovers and places the recovered error into the given variable.
func RecoverToErrorVar(name string, err *error) {
	if err2 := recover(); err2 != nil {
		log.Printf("RecoverToErrorVar(%s) (err=%v), (err2: %v)\n", name, *err, err2)
		switch err2 := err2.(type) {
		case error:
			*err = fmt.Errorf("%s: Recover from panic: %w", name, err2)
		case string:
			*err = fmt.Errorf("%s: Recover from string-panic: %s", name, err2)
		default:
			*err = fmt.Errorf("%s: Recover from unknown-panic: %+v", name, err2)
		}
	}
}

// RecoverToLog in case of error just logs it.
func RecoverToLog(name string) {
	if err := recover(); err != nil {
		log.Printf("RecoverToLog(%s): %+v\n", name, err)
	}
}
