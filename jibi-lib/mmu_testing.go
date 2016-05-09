package jibi

type TestMmu struct {
	ram []Byte
}

func newTestMmu() Mmu {
	return TestMmu{make([]Byte, 0x10000)}
}

func (tm TestMmu) LockAddr(addr Worder, ak AddressKeys) AddressKeys {
	return AddressKeys(0)
}

func (tm TestMmu) UnlockAddr(addr Worder, ak AddressKeys) AddressKeys {
	return AddressKeys(0)
}

func (tm TestMmu) ReadByteAt(addr Worder, ak AddressKeys) Byte {
	return tm.ram[addr.Word()]
}

func (tm TestMmu) WriteByteAt(addr Worder, b Byter, ak AddressKeys) {
	tm.ram[addr.Word()] = b.Byte()
}

func (tm TestMmu) ReadIoByte(addr Worder, ak AddressKeys) (Byte, bool) {
	return tm.ram[addr.Word()], true
}

func (tm TestMmu) SetGpu(gpu *Gpu) {
}

func (tm TestMmu) SetKeypad(kp *Keypad) {
}

func (tm TestMmu) SetInterrupt(in Interrupt, ak AddressKeys) {
}
