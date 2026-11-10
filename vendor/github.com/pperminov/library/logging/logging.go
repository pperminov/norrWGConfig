package libraryLogging

import (
	"fmt"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var (
	Info  func(string, ...any)
	Warn  func(string, ...any)
	Error func(string, ...any)
	Panic func(string, ...any)
	Fatal func(string, ...any)
	Debug func(string, ...any)
)

func init() {
	Info = logAndCapture("info", false)
	Warn = logAndCapture("warning", true)
	Error = logAndCapture("error", true)
	Debug = logAndCapture("debug", false)
	Panic = logAndCapture("panic", true)
	Fatal = logAndCapture("fatal", true)
}

func logAndCapture(level string, sendToSentry bool) func(string, ...any) {
	var printFunc = func(v ...any) {
		fmt.Println(v[0])
	}

	return func(msg string, args ...any) {
		printFunc(fmt.Sprintf("%s [%s] %s", time.Now().Format("2006-01-02 15:04:05.000"), strings.ToUpper(level), fmt.Sprintf(msg, args...)))
	}
}

func Dumper(args ...any) {
	spew.Dump(args...)
}
