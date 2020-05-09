package log

import (
	"fmt"
	"strings"

	"../msg"
)

const (
	Info  = 2
	Error = 4
	Debug = 8
	None  = 0
	All   = 255
	// extra (not info)
	Chain = 256
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

func (l Logger) Chain(address string, chain []msg.Block) {
	if l.types&Chain > 0 {
		var sb strings.Builder
		sb.WriteString(Simplify(address))
		sb.WriteString(" : ")
		for i, block := range chain {
			sb.WriteString("[")
			sb.WriteString(Simplify(block.BlockHash))
			sb.WriteString("]")
			if i != len(chain)-1 {
				sb.WriteString(" -> ")
			}
		}
		fmt.Println(sb.String())
	}
}
