package jibi

import (
	"fmt"
)

type register8 struct {
	vp   *Byte
	lrp  *register8 // lsb of register pair
	mask Byte
}

func newFlagsRegister8() register8 {
	return register8{new(Byte), nil, 0xF0}
}

func newRegister8(lrp *register8) register8 {
	return register8{new(Byte), lrp, 0xFF}
}

func (r register8) String() string {
	return fmt.Sprintf("0x%02X", *r.vp)
}

func (r register8) Word() Word {
	if r.lrp == nil {
		panic("lower register is nil")
	}
	return BytesToWord(r, r.lrp)
}

func (r register8) High() Byte {
	return r.Byte()
}

func (r register8) Low() Byte {
	if r.lrp == nil {
		panic("lower register is nil")
	}
	return r.lrp.Byte()
}

func (r register8) Byte() Byte {
	return *r.vp & r.mask
}

func (r register8) set(v Byter) {
	*r.vp = v.Byte()
}

func (r register8) reset() {
	*r.vp = 0
}

func (r register8) setWord(w Word) {
	if r.lrp == nil {
		panic("lower register is nil")
	}
	r.set(w.High())
	r.lrp.set(w.Low())
}

// flags
const (
	flagZ Byte = 0x80 >> iota
	flagN
	flagH
	flagC
)

func (r register8) flagsString() string {
	fZ := 0
	fN := 0
	fH := 0
	fC := 0
	if r.getFlag(flagZ) {
		fZ = 1
	}
	if r.getFlag(flagN) {
		fN = 1
	}
	if r.getFlag(flagH) {
		fH = 1
	}
	if r.getFlag(flagC) {
		fC = 1
	}
	return fmt.Sprintf("zero:%d sub:%d half:%d carry:%d", fZ, fN, fH, fC)
}

//func (r register8) getWord() Word {
//	return bytesToWord(r.get(), r.lrp.get())
//}

//func (r register8) get() uint8 {
//	return *r.vp & r.mask
//}

func (r register8) setFlag(f Byter) {
	*r.vp |= f.Byte()
}

func (r register8) resetFlag(f Byter) {
	*r.vp &= (f.Byte() ^ 0xFF)
}

func (r register8) getFlag(f Byter) bool {
	if *r.vp&f.Byte() == f.Byte() {
		return true
	}
	return false
}

type register16 uint16

func (r register16) Uint16() uint16 {
	return uint16(r)
}

func (r register16) Word() Word {
	return Word(r)
}

func (r register16) High() Byte {
	return Byte(r >> 8)
}

func (r register16) Low() Byte {
	return Byte(r)
}

func (r register16) String() string {
	return fmt.Sprintf("0x%04X", uint16(r))
}
