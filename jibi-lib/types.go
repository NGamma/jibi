package jibi

// A Bit is a single bit.
type Bit uint8

// A Byte is an 8 bit byte.
type Byte uint8

// Byte returns the Byte representation of b.
func (b Byte) Byte() Byte {
	return b
}

// A Byter is anything that can be represented by a Byte.
type Byter interface {
	Byte() Byte
}

// A Word is a 16 bit word.
type Word uint16

// Word returns the Word representation of w.
func (w Word) Word() Word {
	return w
}

// High returns the high Byte.
func (w Word) High() Byte {
	return Byte(w >> 8)
}

// Low returns the low Byte.
func (w Word) Low() Byte {
	return Byte(w)
}

// A Worder is anything that can be represented by a Word.
type Worder interface {
	Word() Word
	High() Byte
	Low() Byte
}
