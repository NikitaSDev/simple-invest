package servicelog

import (
	"log"
	"os"
)

var (
	infoLog  *log.Logger
	errorLog *log.Logger
)

func init() {
	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
}

func InfoLog() *log.Logger {
	return infoLog
}

func ErrorLog() *log.Logger {
	return errorLog
}
