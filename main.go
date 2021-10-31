package main

import (
	. "eForth/rom"
	. "eForth/vmbasic"
	"fmt"
	"unsafe"
)

func execute(code byte) {
	if code < 65 {
		Primitives[code]()
	} else {
		fmt.Printf("\n Illegal code= %x P= %x", code, P)
	}
}

// goForth Program
func main() {
	P = 0
	WP = 4
	IP = 0
	S = 0
	R = 0
	Top = 0
	tData := uintptr(unsafe.Pointer(&Data))
	CData = *(*[]byte)(unsafe.Pointer(&tData))
	fmt.Printf("\n goForth v1.0 15Aug17fda \n")
	debugon := false
	for {
		Bytecode = CData[P]
		P++
		execute(Bytecode)
		if debugon {
			fmt.Printf("\nP = %x, R=%d, code=%d, IP=%x, WP=%d, top=%d, S=%d, stack[S]=%d", P, R, Bytecode, IP, WP, Top, S, Stack[S])
		}
	}
}
