package jibi

import (
	"fmt"
	"testing"
)

func TestOr(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xB1, 0xB1})
	defer cpu.RunCommand(CmdStop, nil)

	// OR C -- NZ
	cpu.a.set(Byte(0x0F))
	cpu.c.set(Byte(0xF0))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xFF) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	cpu.f.reset()

	// OR C -- Z
	cpu.f.reset()
	cpu.a.set(Byte(0x00))
	cpu.c.set(Byte(0x00))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	cpu.f.reset()
}

func TestXor(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xA8, 0xA8})
	defer cpu.RunCommand(CmdStop, nil)

	// XOR B -- NZ
	cpu.a.set(Byte(0x0F))
	cpu.b.set(Byte(0xFF))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xF0) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	cpu.f.reset()

	// XOR B -- Z
	cpu.f.reset()
	cpu.a.set(Byte(0x0F))
	cpu.b.set(Byte(0x0F))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	cpu.f.reset()
}

func TestBit(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xCB, 0x7C, 0xCB, 0x7C})
	defer cpu.RunCommand(CmdStop, nil)

	// BIT 7, H -- NZ
	cpu.h.set(Byte(0x80))
	cpu.fetch()
	cpu.execute()
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	cpu.f.reset()

	// BIT 7, H -- Z
	cpu.h.set(Byte(0x7F))
	cpu.fetch()
	cpu.execute()
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	cpu.f.reset()
}

func TestJp(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xC3, 0x21, 0x67})
	defer cpu.RunCommand(CmdStop, nil)

	// JP nn
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x6721) {
		t.Error()
	}
}

func TestJr(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x00, 0x00, 0x00, 0x00, 0x18, 0x05, 0x18, 0xFC})
	defer cpu.RunCommand(CmdStop, nil)

	// JR * -- positive offset
	cpu.pc = register16(0x04)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x04+0x02+0x05) {
		t.Error()
	}

	// JR * -- negative offset
	cpu.pc = register16(0x06)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x06+0x02-0x04) {
		t.Error()
	}
}

func TestJrNF(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x00, 0x00, 0x00, 0x00, 0x20, 0x05, 0x20, 0xFC})
	defer cpu.RunCommand(CmdStop, nil)

	// JR NZ, 05 -- Z
	cpu.pc = register16(0x04)
	cpu.f.setFlag(flagZ)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x04+0x02) {
		t.Error()
	}

	// JR NZ, * -- NZ, positive offset
	cpu.pc = register16(0x04)
	cpu.f.resetFlag(flagZ)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x04+0x02+0x05) {
		t.Error()
	}

	// JR NZ, * -- NZ, negative offset
	cpu.pc = register16(0x06)
	cpu.f.resetFlag(flagZ)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x06+0x02-0x04) {
		t.Error()
	}
}

func TestJrF(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x00, 0x00, 0x00, 0x00, 0x28, 0x05, 0x28, 0xFC})
	defer cpu.RunCommand(CmdStop, nil)

	// JR Z, 05 -- NZ
	cpu.pc = register16(0x04)
	cpu.f.resetFlag(flagZ)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x04+0x02) {
		t.Error()
	}

	// JR Z, * -- Z, positive offset
	cpu.pc = register16(0x04)
	cpu.f.setFlag(flagZ)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x04+0x02+0x05) {
		t.Error()
	}

	// JR Z, * -- Z, negative offset
	cpu.pc = register16(0x06)
	cpu.f.setFlag(flagZ)
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x06+0x02-0x04) {
		t.Error()
	}
}

func TestCall(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xCD, 0x40, 0x01})
	defer cpu.RunCommand(CmdStop, nil)

	cpu.sp = register16(0xFFFE)

	// CALL nn
	cpu.fetch()
	cpu.execute()
	if cpu.pc.Word() != Word(0x0140) {
		t.Error()
	}
	if cpu.sp.Word() != Word(0xFFFC) {
		t.Error()
	}
	w := BytesToWord(cpu.readByte(cpu.sp.Word()+1), cpu.readByte(cpu.sp.Word()))
	if w != Word(0x0003) {
		t.Errorf(fmt.Sprintf("0x%04X", w))
	}
}

func TestRet(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xC9, 0x40, 0x01})
	defer cpu.RunCommand(CmdStop, nil)

	cpu.sp = register16(0x01)

	// RET
	cpu.fetch()
	cpu.execute()
	if cpu.sp.Word() != Word(0x03) {
		t.Error()
	}
	if cpu.pc.Word() != Word(0x0140) {
		t.Error()
	}
}

func TestPush(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xC5})
	defer cpu.RunCommand(CmdStop, nil)

	cpu.sp = register16(0xFFFE)

	// PUSH BC
	cpu.b.setWord(0x6004)
	cpu.fetch()
	cpu.execute()
	if cpu.sp.Word() != Word(0xFFFC) {
		t.Error()
	}
	w := BytesToWord(cpu.readByte(cpu.sp.Word()+1), cpu.readByte(cpu.sp.Word()))
	if w != Word(0x6004) {
		t.Errorf(fmt.Sprintf("0x%04X", w))
	}
}

func TestPop(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xC1, 0x00, 0x00, 0x00, 0x03, 0x20})
	defer cpu.RunCommand(CmdStop, nil)

	cpu.sp = register16(0x04)

	// POP BC
	cpu.fetch()
	cpu.execute()
	if cpu.sp.Word() != Word(0x06) {
		t.Error()
	}
	if cpu.b.Word() != Word(0x2003) {
		t.Errorf(fmt.Sprintf("0x%04X", cpu.b.Word()))
	}
}

func TestRr(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x1F})
	defer cpu.RunCommand(CmdStop, nil)

	// RLA -- C, bit0 low
	cpu.f.setFlag(flagC)
	cpu.a.set(Byte(0xFE))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xFF) {
		t.Error(fmt.Sprintf("0x%02X", cpu.a.Byte()))
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// RLA -- NC, bit0 low
	cpu.pc = 0
	cpu.f.resetFlag(flagC)
	cpu.a.set(Byte(0xFE))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x7F) {
		t.Error(fmt.Sprintf("0x%02X", cpu.a.Byte()))
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// RLA -- C, bit0 high
	cpu.pc = 0
	cpu.f.setFlag(flagC)
	cpu.a.set(Byte(0x7F))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xBF) {
		t.Error(fmt.Sprintf("0x%02X", cpu.a.Byte()))
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// RLA -- NC, bit0 high
	cpu.pc = 0
	cpu.f.resetFlag(flagC)
	cpu.a.set(Byte(0x7F))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x3F) {
		t.Error(fmt.Sprintf("0x%02X", cpu.a.Byte()))
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}
}

func TestRl(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0xCB, 0x11})
	defer cpu.RunCommand(CmdStop, nil)

	// RL C -- C, bit7 low
	cpu.f.setFlag(flagC)
	cpu.c.set(Byte(0x7F))
	cpu.fetch()
	cpu.execute()
	if cpu.c.Byte() != Byte(0xFF) {
		t.Error(fmt.Sprintf("0x%02X", cpu.c.Byte()))
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// RL C -- NC, bit7 low
	cpu.pc = 0
	cpu.f.resetFlag(flagC)
	cpu.c.set(Byte(0x7F))
	cpu.fetch()
	cpu.execute()
	if cpu.c.Byte() != Byte(0xFE) {
		t.Error(fmt.Sprintf("0x%02X", cpu.c.Byte()))
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// RL C -- C, bit7 high
	cpu.pc = 0
	cpu.f.setFlag(flagC)
	cpu.c.set(Byte(0xBF))
	cpu.fetch()
	cpu.execute()
	if cpu.c.Byte() != Byte(0x7F) {
		t.Error(fmt.Sprintf("0x%02X", cpu.c.Byte()))
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// RL C -- NC, bit7 high
	cpu.pc = 0
	cpu.f.resetFlag(flagC)
	cpu.c.set(Byte(0xBF))
	cpu.fetch()
	cpu.execute()
	if cpu.c.Byte() != Byte(0x7E) {
		t.Error(fmt.Sprintf("0x%02X", cpu.c.Byte()))
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}
}

func TestRlc(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x07, 0xCB, 0x01})
	defer cpu.RunCommand(CmdStop, nil)

	// RLCA -- bit7 low
	cpu.pc = 0
	cpu.a.set(Byte(0x7F))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xFE) {
		t.Error(fmt.Sprintf("0x%02X", cpu.a.Byte()))
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// RLCA -- bit7 high
	cpu.pc = 0
	cpu.a.set(Byte(0xFE))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xFD) {
		t.Error(fmt.Sprintf("0x%02X", cpu.a.Byte()))
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// RLC C -- bit7 high
	cpu.pc = 1
	cpu.c.set(Byte(0xFE))
	cpu.fetch()
	cpu.execute()
	if cpu.c.Byte() != Byte(0xFD) {
		t.Error(fmt.Sprintf("0x%02X", cpu.c.Byte()))
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// RLC C -- bit7 high
	cpu.pc = 1
	cpu.c.set(Byte(0xFE))
	cpu.fetch()
	cpu.execute()
	if cpu.c.Byte() != Byte(0xFD) {
		t.Error(fmt.Sprintf("0x%02X", cpu.c.Byte()))
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}
}

func TestDec(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x05})
	defer cpu.RunCommand(CmdStop, nil)

	// DEC B -- NZ, NH
	cpu.b.set(Byte(0x44))
	cpu.fetch()
	cpu.execute()
	if cpu.b.Byte() != Byte(0x43) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}

	// DEC B -- NZ, H
	cpu.pc = 0
	cpu.f.reset()
	cpu.b.set(Byte(0x10))
	cpu.fetch()
	cpu.execute()
	if cpu.b.Byte() != Byte(0x0F) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}

	cpu.pc = 0
	cpu.f.reset()
	cpu.b.set(Byte(0x00))
	cpu.fetch()
	cpu.execute()
	if cpu.b.Byte() != Byte(0xFF) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}

	// DEC B -- Z, NH
	cpu.pc = 0
	cpu.f.reset()
	cpu.b.set(Byte(0x01))
	cpu.fetch()
	cpu.execute()
	if cpu.b.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
}

func TestInc(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x04})
	defer cpu.RunCommand(CmdStop, nil)

	// INC B -- NZ, NH
	cpu.b.set(Byte(0x44))
	cpu.fetch()
	cpu.execute()
	if cpu.b.Byte() != Byte(0x45) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}

	// INC B -- NZ, H
	cpu.pc = 0
	cpu.f.reset()
	cpu.b.set(Byte(0x1F))
	cpu.fetch()
	cpu.execute()
	if cpu.b.Byte() != Byte(0x20) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}

	// INC B -- Z, H
	cpu.pc = 0
	cpu.f.reset()
	cpu.b.set(Byte(0xFF))
	cpu.fetch()
	cpu.execute()
	if cpu.b.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
}

func TestAdd(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x86})
	defer cpu.RunCommand(CmdStop, nil)

	// ADD A, (HL) -- NZ, NH, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x24))
	cpu.h.setWord(Word(0xFF80))
	cpu.writeByte(Word(0xFF80), Byte(0x40))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x64) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// ADD A, (HL) -- NZ, NH, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x24))
	cpu.h.setWord(Word(0xFF80))
	cpu.writeByte(Word(0xFF80), Byte(0xF0))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x14) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// ADD A, (HL) -- NZ, H, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x24))
	cpu.h.setWord(Word(0xFF80))
	cpu.writeByte(Word(0xFF80), Byte(0x3C))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x60) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// ADD A, (HL) -- NZ, H, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x24))
	cpu.h.setWord(Word(0xFF80))
	cpu.writeByte(Word(0xFF80), Byte(0xEC))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x10) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// ADD A, (HL) -- Z, NH, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x00))
	cpu.h.setWord(Word(0xFF80))
	cpu.writeByte(Word(0xFF80), Byte(0x00))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// ADD A, (HL) -- Z, NH, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x20))
	cpu.h.setWord(Word(0xFF80))
	cpu.writeByte(Word(0xFF80), Byte(0xE0))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// ADD A, (HL) -- Z, H, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0xFF))
	cpu.h.setWord(Word(0xFF80))
	cpu.writeByte(Word(0xFF80), Byte(0x01))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}
}

func TestAdc(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x8C})
	defer cpu.RunCommand(CmdStop, nil)

	// ADC A, H -- C, NZ, NH, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.f.setFlag(flagC)
	cpu.a.set(Byte(0x24))
	cpu.h.set(Byte(0x12))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x37) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// ADC A, H -- C, NZ, NH, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.f.setFlag(flagC)
	cpu.a.set(Byte(0x24))
	cpu.h.set(Byte(0xE2))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x07) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// ADC A, H -- NC, NZ, H, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.f.resetFlag(flagC)
	cpu.a.set(Byte(0x24))
	cpu.h.set(Byte(0x1E))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x42) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// ADC A, H -- NC, NZ, H, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.f.resetFlag(flagC)
	cpu.a.set(Byte(0x24))
	cpu.h.set(Byte(0xEE))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x12) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// ADC A, H -- NC, Z, NH, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.f.resetFlag(flagC)
	cpu.a.set(Byte(0x00))
	cpu.h.set(Byte(0x00))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// ADC A, H -- NC, Z, H, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.f.resetFlag(flagC)
	cpu.a.set(Byte(0x01))
	cpu.h.set(Byte(0xFF))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// ADC A, H -- C, Z, H, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.f.setFlag(flagC)
	cpu.a.set(Byte(0x01))
	cpu.h.set(Byte(0xFE))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}
}

func TestSub(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x90})
	defer cpu.RunCommand(CmdStop, nil)

	// SUB B -- NZ, NH, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x24))
	cpu.b.set(Byte(0x12))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x12) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// SUB B -- NZ, NH, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x24))
	cpu.b.set(Byte(0x32))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xF2) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// SUB B -- NZ, H, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x32))
	cpu.b.set(Byte(0x24))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x0E) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}

	// SUB B -- NZ, H, C
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x23))
	cpu.b.set(Byte(0x24))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0xFF) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != true {
		t.Error()
	}

	// SUB B -- Z, NH, NC
	cpu.pc = 0
	cpu.f.reset()
	cpu.a.set(Byte(0x32))
	cpu.b.set(Byte(0x32))
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x00) {
		t.Error()
	}
	if cpu.f.getFlag(flagZ) != true {
		t.Error()
	}
	if cpu.f.getFlag(flagH) != false {
		t.Error()
	}
	if cpu.f.getFlag(flagC) != false {
		t.Error()
	}
}

func TestLdd(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x3A, 0x32, 0x00, 0x89})
	defer cpu.RunCommand(CmdStop, nil)

	// LDD A, (HL)
	cpu.h.setWord(0x0003)
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x89) {
		t.Error()
	}
	if cpu.h.Word() != Word(0x0002) {
		t.Error()
	}

	// LDD (HL), A
	cpu.h.setWord(0xFF80)
	cpu.fetch()
	cpu.execute()
	b := cpu.readByte(Word(0xFF80))
	if b != Byte(0x89) {
		t.Error()
	}
	if cpu.h.Word() != Word(0xFF7F) {
		t.Error()
	}
}

func TestLdi(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x2A, 0x22, 0x00, 0x89})
	defer cpu.RunCommand(CmdStop, nil)

	// LDI A, (HL)
	cpu.h.setWord(0x0003)
	cpu.fetch()
	cpu.execute()
	if cpu.a.Byte() != Byte(0x89) {
		t.Error()
	}
	if cpu.h.Word() != Word(0x0004) {
		t.Error()
	}

	// LDI (HL), A
	cpu.h.setWord(0xFF80)
	cpu.fetch()
	cpu.execute()
	b := cpu.readByte(Word(0xFF80))
	if b != Byte(0x89) {
		t.Error()
	}
	if cpu.h.Word() != Word(0xFF81) {
		t.Error()
	}
}

func TestInc16(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x23})
	defer cpu.RunCommand(CmdStop, nil)

	// INC HL
	cpu.h.setWord(0x1FFF)
	cpu.fetch()
	cpu.execute()
	if cpu.h.Word() != Word(0x2000) {
		t.Error()
	}
}

func TestDec16(t *testing.T) {
	cpu := NewCpu(newTestMmu(), []Byte{0x0B})
	defer cpu.RunCommand(CmdStop, nil)

	// DEC BC
	cpu.b.setWord(0x2000)
	cpu.fetch()
	cpu.execute()
	if cpu.b.Word() != Word(0x1FFF) {
		t.Error()
	}
}
