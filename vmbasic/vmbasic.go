package vmbasic

import (
	"bufio"
	. "eForth/rom"
	"fmt"
	"os"
	"time"
)

const (
	// FALSE flag set to 0
	FALSE = 0
	// TRUE flag set to -1, any non-zero are considered true
	TRUE = -1
	// Rom size definition for a variable size program
	romsize = 4096
)

var (
	// Rack is the Return Stack, a 256 cell circular buffer
	rack [256]int32
	// Stack is the Data Stack, a 256 Cell circular buffer
	Stack [256]int32
	// D is a Scractchegisters for 64 bit integers for multiply and divide
	d int64
	// N is a Scractchegisters for 64 bit integers for multiply and divide
	n int64
	// M is a Scractchegisters for 64 bit integers for multiply and divide
	m int64
	// R is the one byte return Stack pointer
	R byte
	// S is the one byte Stack pointer
	S byte
	// Top is the Cached Top element of the Data Stack
	Top int32
	// P is the Program Counter, pointing to byte code in Data[]
	P int32
	// IP is the Instruction Pointer for address interpreter for token lists
	IP int32
	// WP is a Scratch register, generally pointing to parameter field
	WP int32
	// cData Pointer to Data array
	CData []byte
	// bytecode register for byte code to be executed
	Bytecode byte
	// User prompt imput source.
	inputtext []byte
	// String counts
	strCount int
	// input set tags
	reading bool
	// Q keeps track of string input char index
	q int
)

/* There are 67 functions defined in VFM as shown before.
Each of these functions is assigned a unique byte code,
which become the pseudo instructions of this VFM. In the
dictionary, there are primitive commands which have these
byte code in their code field. The byte code may spill over
into the subsequent parameter field, if a primitive command
is very complicated. VFM has a byte code sequencer, which
will be discussed shortly, to sequence through byte code
list. The numbering of these byte code in the following
primitives[] array does not follow any perceived order.

Only 67 byte code are defined. You can extend them to 256
if you wanted. You have the options to write more functions
in go to extend the VFM, or to assemble more primitive
commands using the metacompiler I will discuss later, or to
compile more compound commands in Forth, which is far easier.
The same function defined in different ways should behave
identically. Only the execution speed may differ,
inversely proportional to the efforts in programming. */

// primivaties - Only 67 byte codes are defined.
type forth func()

var Primitives = map[byte]forth{
	0:  nop,
	1:  bye,
	2:  qrx,
	3:  txsto,
	4:  docon,
	5:  dolit,
	6:  dolist,
	7:  exitt,
	8:  execu,
	9:  donext,
	10: qbran,
	11: bran,
	12: store,
	13: at,
	14: cstor,
	15: cat,
	16: nop,
	17: nop,
	18: rfrom,
	19: rat,
	20: tor,
	21: nop,
	22: nop,
	23: drop,
	24: dup,
	25: swap,
	26: over,
	27: zless,
	28: andd,
	29: orr,
	30: xorr,
	31: uplus,
	32: nop,
	33: qdup,
	34: rot,
	35: ddrop,
	36: ddup,
	37: plus,
	38: inver,
	39: negat,
	40: dnega,
	41: subb,
	42: abss,
	43: equal,
	44: uless,
	45: less,
	46: ummod,
	47: msmod,
	48: slmod,
	49: mod,
	50: slash,
	51: umsta,
	52: star,
	53: mstar,
	54: ssmod,
	55: stasl,
	56: pick,
	57: pstor,
	58: dstor,
	59: dat,
	60: count,
	61: dovar,
	62: maxx,
	63: minn,
	64: great,
}

// LOGICAL is a macro enforcing the above policy for logic commands
// to return the correct TRUE and FALSE flags.
func LOGICAL(test bool) int32 {
	if test {
		return TRUE
	}
	return FALSE
}

// LOWER (x,y) returns a TRUE flag if x<y.
func LOWER(x, y int32) int32 {
	if uint32(x) < uint32(y) {
		return TRUE
	}
	return FALSE
}

// POP is a macros which stream-line the often used operations to
// pop the Data Stack to a register or a memory location. As the Top
// element of the Data tack is cached in register Top, popping is more
// complicated, and pop macro helps to clarify my intention.
func pop() {
	Top = Stack[S]
	S--
}

// PUSH is a macro to push a register or contents in a memory
// location on the Data Stack. Actually, the contents in Top register
// must be pushed on the Data Stack, and the source Data is copied into
// Top register.
func push(char int32) {
	S++
	Stack[S] = Top
	Top = char
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func outputRom() {
	f, err := os.Create(time.Now().Format("01-02-2006 15_04_05") + "_rom.txt")
	check(err)
	defer f.Close()

	w := bufio.NewWriter(f)

	fmt.Fprintln(w, "package updated")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "var Data = [%d]uint32{\n", romsize)
	for i := 0; i < romsize; i++ {
		fmt.Fprintf(w, "    /* %08X */ 0x%08X, \n", i*4, Data[i])
	}
	fmt.Fprintln(w, "}")

	fmt.Printf("\n wrote *_rom.txt")

	w.Flush()
}

// Windows Console opened for Forth.
func bye() {
	outputRom()
	os.Exit(0)
}

// readinput ( -- ) Read command line console input. Store strings as char
// in global parameter for later processing.
// verify if char needs to be decimal or octal.
func readinput() {
	reader := bufio.NewReader(os.Stdin)
	inputtext, _ = reader.ReadBytes('\n')
	strCount = len(inputtext)
	//fmt.Println(strCount, " ", inputtext)
	//fmt.Println(R, P, bytecode, IP, WP, Top, S)
}

// getchar ( -- d ) Get one char from command line.  Intiallized readinput routine
// then scan thru all strings. Note: return command from command line returns
// two char 1) 13 "CR" and 2) 10 "LF".  As a result, every input will always have,
// as a minimum, two inputs "13 & 10".
func getchar() int32 {
	var output int32
	if !reading {
		readinput()
		q = 0
		if strCount == 2 {
			output = int32(inputtext[1])
		} else if strCount > 2 {
			reading = true
			output = int32(inputtext[q])
			q++
		}
	} else {
		if q < strCount-2 {
			output = int32(inputtext[q])
			q++
		} else {
			reading = false
			output = int32(inputtext[q]) // output EOF (??)
		}
	}
	return output
}

// putchar ( d -- ) Print Char onto Terminal output.
func putchar(char byte) {
	fmt.Printf("%s", string(char))
}

// qrx ( -- c T|F ) Return a character and a true flag if the character
// has been received. If no character was received, return a false flag
func qrx() {
	push(getchar())
	if Top != 0 {
		push(TRUE)
	} /* else {
		push(FALSE)
	} */
}

// txsto ( c -- ) Send a character to the serial terminal.
func txsto() {
	putchar((byte)(Top))
	pop()
}

// next() is the inner interpreter of the Virtual Forth Machine. Execute
// the next token in a token list. It reads the next token, which is a code
// field address, and deposits it into Program Counter P. The sequencer then
// jumps to this address, and executes the byte code in it. It also deposits
// P+4 into the Work Register WP, pointing to the parameter field of this command.
// WP helps retrieving the token list of a compound command, or Data stored
// in parameter field.
func next() {
	P = int32(Data[(IP >> 2)])
	WP = P + 4
	IP += 4
}

// dovar( -- a ) Return the parameter field address saved in WP register.
func dovar() {
	push(WP)
}

// docon ( -- n ) Return integer stores in the parameter field of a constant
// command. void docon(void) { push Data[WP>>2]};
func docon() {
	push(int32(Data[(WP >> 2)]))
}

// dolit ( -- w) Push the next token onto the Data Stack as an integer literal.
// It allows numbers to be compiled as in-line literals, supplying Data to Data
// Stack at run time.
func dolit() {
	push(int32(Data[(IP >> 2)]))
	IP += 4
	next()
}

// dolist ( -- ) Push the current Instruction Pointer (IP) the return Stack and
// then pops the Program Counter P into IP from the Data Stack. When next() is
// executed, the tokens in the list are executed consecutively.
func dolist() {
	R++
	rack[R] = IP
	IP = WP
	next()
}

// exitt ( -- ) Terminate all token lists in compound commands. EXIT pops the
// execution address saved on the return Stack back into the IP register and
// thus restores the condition before the compound command was entered. Execution
// of the calling token list will continue.
func exitt() {
	IP = rack[R]
	R--
	next()
}

// execu ( a -- ) Take the execution address from Data Stack and executes that token.
// This powerful command allows you to execute any token which is not a part of a
// token list.
func execu() {
	P = Top
	WP = P + 4
	pop()
}

// donext ( -- ) Terminate a FOR-NEXT loop. The loop count was pushed on return Stack,
// and is decremented by donext. If the count is not negative, jump to the address
// following donext; otherwise, pop the count off return Stack and exit the loop.
func donext() {
	if rack[R] != 0 {
		rack[R]--
		IP = int32(Data[(IP >> 2)])
	} else {
		IP += 4
		R--
	}
	next()
}

// qbran ( f -- ) Test Top as a flag on Data Stack. If it is zero, branch to the address
// following qbran; otherwise, continue execute the token list following the address.
func qbran() {
	if Top == 0 {
		IP = int32(Data[(IP >> 2)])
	} else {
		IP += 4
	}
	pop()
	next()
}

// bran ( -- ) Branch to the address following bran.
func bran() {
	IP = int32(Data[(IP >> 2)])
	next()
}

// store ( n a -- ) Store integer n into memory location a.
func store() {
	Data[(Top >> 2)] = uint32(Stack[S])
	S--
	pop()
}

// at ( a -- n ) Replace memory address a with its integer contents fetched from this location.
func at() {
	Top = int32(Data[(Top >> 2)])
}

// cstor ( c b -- ) Store a byte value c into memory location b.
func cstor() {
	CData[Top] = byte(Stack[S])
	S--
	pop()
}

// cat ( b -- n) Replace byte memory address b with its byte contents fetched from this location.
func cat() {
	Top = int32(CData[Top])
}

// rfrom ( n -- ) Pop a number off the Data Stack and pushes it on the return Stack.
func rfrom() {
	push(rack[R])
	R--
}

// rat ( -- n ) Copy a number off the return Stack and pushes it on the return Stack.
func rat() {
	push(rack[R])
}

// tor ( -- n ) Pop a number off the return Stack and pushes it on the Data Stack.
func tor() {
	R++
	rack[R] = Top
	pop()
}

// drop ( w -- ) Discard Top Stack item.
func drop() {
	pop()
}

// dup ( w -- w w ) Duplicate the Top Stack item.
func dup() {
	S++
	Stack[S] = Top
}

// swap ( w1 w2 -- w2 w1 ) Exchange Top two Stack items.
func swap() {
	WP = Top
	Top = Stack[S]
	Stack[S] = WP
}

// over ( w1 w2 -- w1 w2 w1 ) Copy second Stack item to Top.
func over() {
	push(Stack[S])
}

// zless ( n – f ) Examine the Top item on the Data Stack
// for its negativeness. If it is negative, return a 1 for true.
// If it is 0 or positive, return a 0 for false.
func zless() {
	Top = LOGICAL(Top < 0)
}

// andd ( w {w -- w ) Bitwise AND.}
func andd() {
	Top &= Stack[S]
	S--
}

// orr ( w w -- w ) Bitwise inclusive OR.
func orr() {
	Top |= Stack[S]
	S--
}

// xorr ( w w -- w ) Bitwise exclusive OR.
func xorr() {
	Top ^= Stack[S]
	S--
}

// uplus ( w w -- w cy ) Add two numbers, return the sum and carry flag.
func uplus() {
	Stack[S] += Top
	Top = LOWER(Stack[S], Top)
}

// Nop ( -- ) No operation.
func nop() {
	next()
}

// qdup ( w -- w w | 0 ) Dup Top of Stack if it is not zero.
func qdup() {
	if Top != 0 {
		S++
		Stack[S] = Top
	}
}

// rot ( w1 w2 w3 -- w2 w3 w1 ) Rot 3rd item to Top.
func rot() {
	WP = Stack[(S - 1)]
	Stack[(S - 1)] = Stack[S]
	Stack[S] = Top
	Top = WP
}

// ddrop ( w w -- ) Discard two items on Stack.
func ddrop() {
	drop()
	drop()
}

// ddup ( w1 w2 -- w1 w2 w1 w2 ) Duplicate Top two items.
func ddup() {
	over()
	over()
}

// plus ( w w -- sum ) Add Top two items.
func plus() {
	Top += Stack[S]
	S--
}

// inver ( w -- w ) One's complement of Top.
func inver() {
	Top = -Top - 1
}

// negat ( n -- -n ) Two's complement of Top.
func negat() {
	Top = 0 - Top
}

// dnega ( d -- -d ) Two's complement of Top double.
func dnega() {
	inver()
	tor()
	inver()
	push(1)
	uplus()
	rfrom()
	plus()
}

// subb ( n1 n2 -- n1-n2 ) Subtraction.
func subb() {
	Top = Stack[S] - Top
	S--
}

// abss ( n -- n ) Return the absolute value of n.
func abss() {
	if Top < 0 {
		Top = -Top
	}
}

// great ( n1 n2 -- t ) Signed compare of Top two items.
// Return true if n1>n2.
func great() {
	Top = LOGICAL(Stack[S] > Top)
	S--
}

// less ( n1 n2 -- t ) Signed compare of Top two items.
// Return true if n1<n2.
func less() {
	Top = LOGICAL(Stack[S] < Top)
	S--
}

// equal ( w w -- t ) Return true if Top two are equal.
func equal() {
	Top = LOGICAL(Stack[S] == Top)
	S--
}

// uless ( u1 u2 -- t ) Unsigned compare of Top two items.
func uless() {
	Top = LOGICAL(LOWER(Stack[S], Top) == TRUE)
	S--
}

// ummod ( udl udh u -- ur uq ) Unsigned divide of a double
// by a single. Return mod and quotient.
func ummod() {
	d = int64(uint32(Top))
	m = int64(uint32(Stack[S]))
	n = int64(uint32(Stack[(S - 1)]))
	n += m << 32
	pop()
	Top = int32(n / d)
	Stack[S] = int32(n % d)
}

// msmod ( d n -- r q ) Signed floored divide of double by
// single. Return mod and
func msmod() {
	d = int64(int32(Top))
	m = int64(int32(Stack[S]))
	n = int64(int32(Stack[(S - 1)]))
	n += m << 32
	pop()
	Top = int32(n / d)      //work on using Top int32 (signed!!!!)
	Stack[S] = int32(n % d) //work on using Stack[] int32 (signed!!!!)
}

// slmod ( n1 n2 -- r q ) Signed divide. Return mod and quotient.
func slmod() {
	if Top != 0 {
		WP = Stack[S] / Top
		Stack[S] %= Top
		Top = WP
	}
}

// mod ( n n -- r ) Signed divide. Return mod only.
func mod() {
	if Top != 0 {
		Top = Stack[S] % Top
	} else {
		Top = Stack[S]
	}
	S--
}

// slash ( n n -- q ) Signed divide. Return quotient only.
func slash() {
	if Top != 0 {
		Top = Stack[S] / Top
	} else {
		Top = Stack[S]
		Stack[S] = 0
	}
	S--
}

//umsta ( u1 u2 -- ud ) Unsigned multiply. Return double product.
func umsta() {
	d = int64(uint32(Top))
	m = int64(uint32(Stack[S]))
	m *= d
	Top = int32(m >> 32)
	Stack[S] = int32(m)
}

//star ( n n -- n ) Signed multiply. Return single product.
func star() {
	Top *= Stack[S]
	S--
}

//mstar ( n1 n2 -- d ) Signed multiply. Return double product.
func mstar() {
	d = int64(Top)
	m = int64(Stack[S])
	m *= d
	Top = int32(m >> 32)
	Stack[S] = int32(m)
}

// ssmod ( n1 n2 n3 -- r q ) Multiply n1 and n2, then divide by n3.
// Return mod and quotient.
func ssmod() {
	d = int64(Top)
	m = int64(Stack[S])
	n = int64(Stack[(S - 1)])
	n += m << 32
	pop()
	Top = int32(n / d)
	Stack[S] = int32(n % d)
}

// stasl ( n1 n2 n3 -- q ) Multiply n1 by n2, then divide by n3.
// Return quotient only.
func stasl() {
	d = int64(Top)
	m = int64(Stack[S])
	n = int64(Stack[(S - 1)])
	n += m << 32
	pop()
	pop()
	Top = int32(n / d)
}

// pick ( ... +n -- ... w ) Copy the nth Stack item to Top.
func pick() {
	Top = Stack[S-(byte)(Top)]
}

// pstor ( n a -- ) Add n to the contents at address a.
func pstor() {
	Data[(Top >> 2)] += uint32(Stack[S])
	S--
	pop()
}

// dstor ( d a -- ) Store the double integer to address a.
func dstor() {
	Data[((Top >> 2) + 1)] = uint32(Stack[S])
	S--
	Data[(Top >> 2)] = uint32(Stack[S])
	S--
	pop()
}

// dat ( a -- d ) Fetch double integer from address a.
func dat() {
	push(int32(Data[(Top >> 2)]))
	Top = int32(Data[((Top >> 2) + 1)])
}

// count ( b -- b+1 +n ) Return count byte of a string and
// add 1 to byte address.
func count() {
	S++
	Stack[S] = Top + 1
	Top = int32(CData[Top])
}

// max ( n1 n2 -- n ) Return the greater of two Top Stack items.
func maxx() {
	if Top < Stack[S] {
		pop()
	} else {
		S--
	}
}

// min ( n1 n2 -- n ) Return the smaller of Top two Stack items.
func minn() {
	if Top < Stack[S] {
		S--
	} else {
		pop()
	}
}
