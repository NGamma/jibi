package jibi

import (
	"fmt"
	"sync"
)

// A list of all the special memory addresses.
const (
	AddrRom    Word = 0x0000
	AddrVRam   Word = 0x8000
	AddrERam   Word = 0xA000
	AddrRam    Word = 0xC000
	AddrOam    Word = 0xFE00
	AddrOamEnd Word = 0xFEA0

	AddrP1  Word = 0xFF00
	AddrDIV Word = 0xFF04
	AddrTMA Word = 0xFF06
	AddrIF  Word = 0xFF0F

	AddrGpuRegs    Word = 0xFF40
	AddrLCDC       Word = 0xFF40
	AddrSTAT       Word = 0xFF41
	AddrSCY        Word = 0xFF42
	AddrSCX        Word = 0xFF43
	AddrLY         Word = 0xFF44
	AddrLYC        Word = 0xFF45
	AddrDMA        Word = 0xFF46
	AddrBGP        Word = 0xFF47
	AddrOBP0       Word = 0xFF48
	AddrOBP1       Word = 0xFF49
	AddrWY         Word = 0xFF4A
	AddrWX         Word = 0xFF4B
	AddrGpuRegsEnd Word = 0xFF4C

	AddrZero Word = 0xFF80
	AddrIE   Word = 0xFFFF
)

// An Mmu is the memory management unit. Its purpose is to dispatch read and
// write requeststo the appropriate module (cpu, gpu, etc) based on the memory
// address. The Mmu is controlled by the cpu.
type Mmu struct {
	// memory blocks
	rom     []Byte
	vram    []Byte
	ram     []Byte
	oam     []Byte
	gpuregs []Byte
	zero    []Byte
	ie      Byte

	// memory mapped io
	ioIF *mmio
	ioP1 *mmio

	// memory locks
	locks []*sync.Mutex

	// internal state
	kp *Keypad
}

// NewMmu creates a new Mmu with an optional bios that replaces 0x0000-0x00FF.
func NewMmu(cart *Cartridge) *Mmu {
	var rom []Byte
	if cart != nil {
		rom = cart.Rom
	}
	locks := make([]*sync.Mutex, abLast+1)
	for i := uint16(1); i <= uint16(abLast); i = i << 1 {
		locks[i] = new(sync.Mutex)
	}
	mmu := &Mmu{
		rom:     rom,
		vram:    make([]Byte, 0x2000),
		ram:     make([]Byte, 0x2000),
		oam:     make([]Byte, 0xA0),
		ioIF:    newMmio(AddrIF),
		ioP1:    newMmio(AddrP1),
		gpuregs: make([]Byte, 12),
		zero:    make([]Byte, 0x100),
		locks:   locks,
	}
	return mmu
}

type addressBlock uint16
type AddressKeys uint16

const (
	abNil addressBlock = iota
	abRom addressBlock = 1 << iota
	abVRam
	abERam
	abRam
	abOam
	abP1
	abIF
	abGpuRegs
	abZero
	abIE
	abLast = abIE
)

func (a addressBlock) String() string {
	switch a {
	case abNil:
		return "abNil"
	case abRom:
		return "abRom"
	case abVRam:
		return "abVRam"
	case abERam:
		return "abERam"
	case abRam:
		return "abRam"
	case abOam:
		return "abOam"
	case abIF:
		return "abIF"
	case abGpuRegs:
		return "abGpuRegs"
	case abZero:
		return "abZero"
	case abIE:
		return "abIE"
	}
	return "abUNKNOWN"
}

func (m *Mmu) SetKeypad(kp *Keypad) {
	m.kp = kp
}

func (m *Mmu) selectAddressBlock(addr Worder, rw string) (addressBlock, Word) {
	a := addr.Word()
	if a < AddrVRam {
		return abRom, 0
	} else if AddrVRam <= a && a < AddrERam {
		return abVRam, AddrVRam
	} else if AddrERam <= a && a < AddrRam {
		return abERam, AddrERam
	} else if AddrRam <= a && a < AddrOam {
		return abRam, AddrRam
	} else if AddrOam <= a && a < AddrOamEnd {
		return abOam, AddrOam
	} else if AddrP1 == a {
		return abP1, AddrP1
	} else if AddrIF == a {
		return abIF, AddrIF
	} else if AddrGpuRegs <= a && a < AddrGpuRegsEnd {
		return abGpuRegs, AddrGpuRegs
	} else if AddrZero <= a && a < AddrIE {
		return abZero, AddrZero
	} else if AddrIE == a {
		return abIE, AddrIE
	}

	u, v := m.getAddressInfo(addr)
	if !v {
		if rw == "" {
			rw = "access"
		}
		panic(fmt.Sprintf("unhandled memory %s: 0x%04X - %s", rw, a, u))
	}
	return abNil, 0
}

// LockAddr gets a lock for an address if not already in the provided
// AddressKeys and appends it and returns this new key set.
func (m *Mmu) LockAddr(addr Worder, ak AddressKeys) AddressKeys {
	blk, _ := m.selectAddressBlock(addr, "lock")
	if addressBlock(ak)&blk == blk {
		// already have the key
		return ak
	}
	m.locks[blk].Lock()
	return ak | AddressKeys(blk)
}

func (m *Mmu) UnlockAddr(addr Worder, ak AddressKeys) AddressKeys {
	blk, _ := m.selectAddressBlock(addr, "unlock")
	if addressBlock(ak)&blk != blk {
		// don't have the key
		return ak
	}
	m.locks[blk].Unlock()
	return ak & AddressKeys(blk^0xFFFF)
}

func (m *Mmu) ReadByteAt(addr Worder, ak AddressKeys) Byte {
	blk, start := m.selectAddressBlock(addr, "read")
	owner := addressBlock(ak)&blk == blk
	if blk == abRom {
		if owner {
			return m.rom[addr.Word()-start]
		}
	}
	if blk == abVRam {
		if owner {
			return m.vram[addr.Word()-start]
		}
	} else if blk == abRam {
		if owner {
			return m.ram[(addr.Word()-start)&0x1FFF]
		}
	} else if blk == abOam {
		if owner {
			return m.oam[addr.Word()-start]
		}
	} else if blk == abP1 {
		return m.ioP1.readByte(owner)
	} else if blk == abIF {
		return m.ioIF.readByte(owner)
	} else if blk == abGpuRegs {
		if owner {
			return m.gpuregs[addr.Word()-start]
		}
	} else if blk == abZero {
		if owner {
			return m.zero[addr.Word()-start]
		}
	} else if blk == abIE {
		if owner {
			return m.ie
		}
	}
	if !owner {
		panic(fmt.Sprintf("unauthorized read: 0x%04X", addr.Word()))
	}
	if u, v := m.getAddressInfo(addr); !v {
		panic(fmt.Sprintf("unhandled memory read: 0x%04X - %s", addr.Word(), u))
	}
	return 0
}

func (m *Mmu) WriteByteAt(addr Worder, b Byter, ak AddressKeys) {
	blk, start := m.selectAddressBlock(addr, "write")
	owner := addressBlock(ak)&blk == blk
	if blk == abRom {
		return
	} else if blk == abVRam {
		if owner {
			m.vram[addr.Word()-start] = b.Byte()
			return
		}
	} else if blk == abRam {
		if owner {
			m.ram[(addr.Word()-start)&0x1FFF] = b.Byte()
			return
		}
	} else if blk == abOam {
		if owner {
			m.oam[addr.Word()-start] = b.Byte()
			return
		}
	} else if blk == abP1 {
		m.ioP1.writeByte(b, owner)
		if !owner {
			m.kp.RunCommand(CmdKeyCheck, nil)
		}
		return
	} else if blk == abIF {
		m.ioIF.writeByte(b, owner)
		return
	} else if blk == abGpuRegs {
		if owner {
			m.gpuregs[addr.Word()-start] = b.Byte()
			return
		}
	} else if blk == abZero {
		if owner {
			m.zero[addr.Word()-start] = b.Byte()
			return
		}
	} else if blk == abIE {
		if owner {
			m.ie = b.Byte()
			return
		}
	}
	if !owner {
		panic(fmt.Sprintf("unauthorized write: 0x%04X 0x%02X", addr.Word(), b.Byte()))
	}
	if u, v := m.getAddressInfo(addr); !v {
		panic(fmt.Sprintf("unhandled memory write: 0x%04X - %s", addr.Word(), u))
	}
}

func (m *Mmu) ReadIoByte(addr Worder, ak AddressKeys) (Byte, bool) {
	blk, _ := m.selectAddressBlock(addr, "write")
	owner := addressBlock(ak)&blk == blk
	if blk == abP1 {
		return m.ioP1.readIoByte(owner)
	} else if blk == abIF {
		return m.ioIF.readIoByte(owner)
	}
	panic(fmt.Sprintf("unhandled queued write: 0x%04X", addr.Word()))
}

// incomplete, used for debugging
func (m *Mmu) getAddressInfo(addr Worder) (string, bool) {
	a := addr.Word()
	if 0x9C00 <= a && a <= 0x9FFF {
		return "Background Map Data 2", false
	} else if 0xFEA0 <= a && a <= 0xFEFF {
		return "unusable memory", true
	} else if a == 0xFF00 {
		return "Register for reading joy pad info and determining system type. (R/W)", false
	} else if a == 0xFF01 {
		return "Serial transfer data (R/W)", true
	} else if a == 0xFF02 {
		return "SIO control (R/W)", true
	} else if a == 0xFF03 {
		return "no clue", true
	} else if a == 0xFF04 {
		return "DIV", true // TODO: priority
	} else if a == 0xFF05 {
		return "TIMA", true // TODO: priority
	} else if a == 0xFF06 {
		return "TMA", true // TODO: priority
	} else if a == 0xFF07 {
		return "TAC", true // TODO: priority
	} else if 0xFF08 <= a && a <= 0xFF0E {
		return "no clue", true
	} else if a == 0xFF10 {
		return "Sound Mode 1 register, Sweep register (R/W)", true
	} else if a == 0xFF11 {
		return "Sound Mode 1 register, Sound length/Wave pattern duty (R/W)", true
	} else if a == 0xFF12 {
		return "Sound Mode 1 register, Envelope (R/W)", true
	} else if a == 0xFF13 {
		return "Sound Mode 1 register, Frequency lo (W)", true
	} else if a == 0xFF14 {
		return "Sound Mode 1 register, Frequency hi (R/W)", true
	} else if a == 0xFF17 {
		return "Sound Mode 2 register, envelope (R/W)", true
	} else if a == 0xFF19 {
		return "Sound Mode 2 register, frequency", true
	} else if a == 0xFF1A {
		return "Sound Mode 3 register, Sound on/off (R/W)", true
	} else if a == 0xFF20 {
		return "Sound Mode 4 register, sound length (R/W)", true
	} else if a == 0xFF21 {
		return "Sound Mode 4 register, envelope (R/W)", true
	} else if a == 0xFF23 {
		return "Sound Mode 4 register, counter/consecutive; inital (R/W)", true
	} else if a == 0xFF24 {
		return "Channel control / ON-OFF / Volume (R/W)", true
	} else if a == 0xFF25 {
		return "Selection of Sound output terminal (R/W)", true
	} else if a == 0xFF26 {
		return "Sound on/off (R/W)", true
	} else if 0xFF30 <= a && a <= 0xFF3F {
		return "Sound Sample RAM", true
	} else if a == 0xFF47 {
		return "BGP", false
	} else if 0xFF4D <= a && a <= 0xFF7F {
		return "GBC", true
	} else if a == 0xFFFF {
		return "IE", false
	}
	return "unknown", false
}

// memory mapped io
type mmio struct {
	addr Word

	// accessed by owner
	value Byte

	// accessed through lock
	read   Byte
	write  Byte
	queued bool
	lock   *sync.Mutex
}

func newMmio(addr Worder) *mmio {
	m := &mmio{addr: addr.Word(),
		lock: new(sync.Mutex)}
	return m
}

func (m *mmio) readByte(owner bool) Byte {
	if owner {
		return m.value
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.read
}

func (m *mmio) writeByte(b Byter, owner bool) {
	if owner {
		m.lock.Lock()
		defer m.lock.Unlock()
		m.value = b.Byte()
		m.read = m.value
		if !m.queued {
			m.write = m.value
		}
	} else {
		m.lock.Lock()
		defer m.lock.Unlock()
		if m.queued {
			//panic(fmt.Sprintf("overwritten io write: 0x%04X", m.addr))
		}
		m.queued = true
		m.write = b.Byte()
	}
}

func (m *mmio) readIoByte(owner bool) (Byte, bool) {
	if owner {
		m.lock.Lock()
		defer m.lock.Unlock()
		q := m.queued
		m.queued = false
		return m.write, q
	}
	panic(fmt.Sprintf("unhandled io read: 0x%04X", m.addr))
}
