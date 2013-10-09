package jibi

import (
	"fmt"
	"time"
)

// A Cpu is the central proecessing unit. This one is similar to a z80. Its
// purpose is to handle interrupts, fetch and execute instructions, and
// manage the clock.
type Cpu struct {
	CommanderInterface

	// registers
	a  register8
	b  register8
	c  register8
	d  register8
	e  register8
	f  register8 // 8 bits, but lower 4 bits always read zero
	h  register8
	l  register8
	sp register16
	pc register16

	// clocks
	tClocks []*Clock // t clock cycle exported clocks
	m       uint8    // machine cycles
	t       uint8    // clock cycles

	// current instruction buffer
	inst instruction

	// interrupt master enable
	ime Bit

	mmu     *Mmu
	mmuKeys AddressKeys

	// internal state
	bios         []Byte
	biosFinished bool

	// notifications
	notifyInst []chan string

	// cpu information
	hz     float64
	period time.Duration
}

// NewCpu creates a new Cpu with mmu connection.
func NewCpu(mmu *Mmu, bios []Byte) *Cpu {
	// use internal clock
	// 1 machine cycle = 4 clock cycles
	// machine cycles: 1.05MHz nop: 1 cycle
	// clock cycles: 4.19MHz nop: 4 cycles
	hz := 4.194304 * 1e6 / 4.0 // 4.19MHz clock to 1.05 machine cycles
	period := time.Duration(1e9 / hz)

	f := newFlagsRegister8()
	a := newRegister8(&f)
	c := newRegister8(nil)
	b := newRegister8(&c)
	e := newRegister8(nil)
	d := newRegister8(&e)
	l := newRegister8(nil)
	h := newRegister8(&l)

	biosFinished := true
	if len(bios) > 0 {
		biosFinished = false
		biosN := make([]Byte, 0x100)
		copy(biosN, bios)
		bios = biosN
	}

	mmuKeys := AddressKeys(0)

	if mmu != nil {
		mmuKeys = mmu.LockAddr(AddrRom, mmuKeys)
		mmuKeys = mmu.LockAddr(AddrRam, mmuKeys)
		mmuKeys = mmu.LockAddr(AddrIF, mmuKeys)
		mmuKeys = mmu.LockAddr(AddrZero, mmuKeys)
		mmuKeys = mmu.LockAddr(AddrIE, mmuKeys)
	}

	commander := NewCommander("cpu")
	cpu := &Cpu{CommanderInterface: commander,
		a: a, b: b, c: c, d: d, e: e, f: f, l: l, h: h,
		ime:          Bit(1),
		mmu:          mmu,
		mmuKeys:      mmuKeys,
		bios:         bios,
		biosFinished: biosFinished,
		hz:           hz, period: period,
	}
	cmdHandlers := map[Command]CommandFn{
		CmdClockAccumulator: cpu.cmdClock,
		CmdString:           cpu.cmdString,
		CmdOnInstruction:    cpu.cmdOnInstruction,
	}

	commander.start(cpu.step, cmdHandlers, nil)
	return cpu
}

func (c *Cpu) cmdClock(resp interface{}) {
	if resp, ok := resp.(chan chan ClockType); !ok {
		panic("invalid command response type")
	} else {
		clk := make(chan ClockType, 1)
		c.tClocks = append(c.tClocks, NewClock(clk))
		resp <- clk
	}
}

func (c *Cpu) cmdOnInstruction(resp interface{}) {
	if resp, ok := resp.(chan chan string); !ok {
		panic("invalid command response type")
	} else {
		inst := make(chan string)
		c.notifyInst = append(c.notifyInst, inst)
		resp <- inst
	}
}

func (c *Cpu) cmdString(resp interface{}) {
	if resp, ok := resp.(chan string); !ok {
		panic("invalid command response type")
	} else {
		resp <- c.str()
	}
}

func (c *Cpu) str() string {
	return fmt.Sprintf(`%s
a:%s f:%s b:%s c:%s d:%s e:%s h:%s l:%s sp:%s pc:%s
ime:%d %s`,
		c.inst, c.a, c.f, c.b, c.c, c.d, c.e, c.h, c.l, c.sp, c.pc,
		c.ime, c.f.flagsString())
}

func (c *Cpu) String() string {
	resp := make(chan string)
	c.RunCommand(CmdString, resp)
	return <-resp
}

func (c *Cpu) lockAddr(addr Worder) {
	c.mmuKeys = c.mmu.LockAddr(addr, c.mmuKeys)
}

func (c *Cpu) unlockAddr(addr Worder) {
	c.mmuKeys = c.mmu.UnlockAddr(addr, c.mmuKeys)
}

func (c *Cpu) readByte(addr Worder) Byte {
	a := addr.Word()
	if !c.biosFinished && a <= 0xFF {
		return c.bios[a]
	}
	if AddrVRam <= a && a <= AddrRam {
		c.lockAddr(AddrVRam)
		defer c.unlockAddr(AddrVRam)
	} else if AddrOam <= a && a <= AddrOamEnd {
		c.lockAddr(AddrOam)
		defer c.unlockAddr(AddrOam)
	}
	return c.mmu.ReadByteAt(addr, c.mmuKeys)
}

func (c *Cpu) writeByte(addr Worder, b Byter) {
	a := addr.Word()
	if AddrVRam <= a && a <= AddrRam {
		c.lockAddr(AddrVRam)
		defer c.unlockAddr(AddrVRam)
	} else if AddrOam <= a && a <= AddrOamEnd {
		c.lockAddr(AddrOam)
		defer c.unlockAddr(AddrOam)
	}
	c.mmu.WriteByteAt(addr, b, c.mmuKeys)
}

func (c *Cpu) readWord(addr Worder) Word {
	l := c.readByte(addr)
	h := c.readByte(addr.Word() + 1)
	return BytesToWord(h, l)
}

func (c *Cpu) writeWord(addr Worder, w Worder) {
	c.writeByte(addr, w.Low())
	c.writeByte(addr.Word()+1, w.High())
}

// Clock returns a new channel that holds acumulating clock ticks.
func (c *Cpu) Clock() chan ClockType {
	resp := make(chan chan ClockType)
	c.RunCommand(CmdClockAccumulator, resp)
	return <-resp
}

func (c *Cpu) fetch() {
	op := opcode(c.readByte(c.pc))
	c.pc++
	if op == 0xCB {
		op = opcode(0xCB00 + uint16(c.readByte(c.pc)))
		c.pc++
	}
	command := commandTable[op]
	p := []Byte{}
	for i := uint8(0); i < command.b; i++ {
		p = append(p, c.readByte(c.pc))
		c.pc++
	}
	c.inst = newInstruction(op, p...)
}

func (c *Cpu) execute() {
	if cmd, ok := commandTable[c.inst.o]; ok {
		cmd.f(c)
		c.t += cmd.t
		c.m += cmd.t * 4
	}
}

/*
// setInterrupt sets the specific interrupt.
func (cpu *Cpu) setInterrupt(in Interrupt) {
	if cpu.ime == 1 {
		ie := cpu.readByte(AddrIE)
		if ie&Byte(in) == Byte(in) {
			iflag
			cpu.iflag |= Byte(in)
			cpu.writeByte(AddrIF, cpu.iflag)
		}
	}
}
*/

// resetInterrupt resets the specific interrupt.
func (cpu *Cpu) resetInterrupt(i Interrupt, iflag Byte) {
	iflag &= (Byte(i) ^ 0xFF)
	cpu.writeByte(AddrIF, iflag)
}

// getInterrupt returns the highest priority enabled interrupt.
func (cpu *Cpu) getInterrupt(ie, iflag Byte) Interrupt {
	if Byte(InterruptVblank)&ie&iflag != 0 {
		return InterruptVblank
	} else if Byte(InterruptLCDC)&ie&iflag != 0 {
		return InterruptLCDC
	} else if Byte(InterruptTimer)&ie&iflag != 0 {
		return InterruptTimer
	} else if Byte(InterruptSerial)&ie&iflag != 0 {
		return InterruptSerial
	} else if Byte(InterruptKeypad)&ie&iflag != 0 {
		return InterruptKeypad
	}
	return 0
}

func (cpu *Cpu) io() {
	iflag := cpu.mmu.ReadIoByte(AddrIF, cpu.mmuKeys)
	if cpu.ime == 0 {
		iflag = 0 // mask all interrupts
	} else {
		ie := cpu.readByte(AddrIE)
		iflag &= ie // mask interrupts
	}
	cpu.writeByte(AddrIF, iflag)
}

func (cpu *Cpu) interrupt() {
	if cpu.ime == 1 {
		ie := cpu.readByte(AddrIE)
		iflag := cpu.readByte(AddrIF)
		in := cpu.getInterrupt(ie, iflag)
		if in > 0 {
			cpu.ime = 0
			cpu.push(cpu.pc)
			cpu.jp(in.Address())
			cpu.resetInterrupt(in, iflag)
		}
	}
}

func (c *Cpu) step(first bool, t uint32) (CommanderStateFn, bool, uint32, uint32) {
	// reset clocks
	c.m = 0
	c.t = 0
	if !c.biosFinished && c.pc == 0x0100 {
		c.biosFinished = true
	}
	for _, inst := range c.notifyInst {
		inst <- c.str()
	}

	c.io()        // handle memory mapped io
	c.interrupt() // handle interrupts
	c.fetch()     // load next instruction into c.inst
	c.lockAddr(AddrGpuRegs)
	c.execute() // execute c.inst instruction
	c.unlockAddr(AddrGpuRegs)

	for _, clk := range c.tClocks {
		clk.AddCycles(c.t)
	}
	return c.step, false, 0, 0
}
