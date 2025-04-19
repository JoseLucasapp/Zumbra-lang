package vm

import (
	"vaja/code"
	"vaja/object"
)

type Frame struct {
	fct *object.CompiledFunction
	ip  int
}

func NewFrame(fct *object.CompiledFunction) *Frame {
	return &Frame{fct: fct, ip: -1}
}

func (f *Frame) Instructions() code.Instructions {
	return f.fct.Instructions
}
