package logger

import "fmt"

const (
	Info  = 2
	Error = 4
	Debug = 8

	None = 0
	All  = 255
)

type Logger struct {
	types int
}

func New(types ...int) Logger {
	l := Logger{}
	for _, t := range types {
		l.types |= t
	}
	return l
}

func (l Logger) Errorf(format string, a ...interface{}) {
	if l.types&Error > 0 {
		fmt.Printf(format+"\n", a...)
	}
}

func (l Logger) Infof(format string, a ...interface{}) {
	if l.types&Info > 0 {
		fmt.Printf(format+"\n", a...)
	}
}

func (l Logger) Debugf(format string, a ...interface{}) {
	if l.types&Debug > 0 {
		fmt.Printf(format+"\n", a...)
	}
}
