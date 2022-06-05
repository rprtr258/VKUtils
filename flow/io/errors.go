package io

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
			err4 := fmt.Errorf("%s: Recover from panic: %w", name, err2)
			*err = err4
		case string:
			err4 := fmt.Errorf("%s: Recover from string-panic: %s", name, err2)
			*err = err4
		default:
			err4 := fmt.Errorf("%s: Recover from unknown-panic: %+v", name, err2)
			*err = err4
		}
	}
}

// RecoverToLog in case of error just logs it.
func RecoverToLog(name string) {
	if err2 := recover(); err2 != nil {
		log.Printf("RecoverToLog(%s) err2: %+v\n", name, err2)
	}
}
