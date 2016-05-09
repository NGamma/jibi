package jibi

import ()

// An Interrupt is what it is.
type Interrupt uint8

// A list of all five interrupts.
const (
	InterruptVblank Interrupt = 0x01 << iota
	InterruptLCDC
	InterruptTimer
	InterruptSerial
	InterruptKeypad
)

// Address returns the restart address associated with the interrupt.
func (i Interrupt) Address() Word {
	switch i {
	case InterruptVblank:
		return Word(0x0040)
	case InterruptLCDC:
		return Word(0x0048)
	case InterruptTimer:
		return Word(0x0050)
	case InterruptSerial:
		return Word(0x0058)
	case InterruptKeypad:
		return Word(0x0060)
	default:
		return Word(0)
	}
}

func (i Interrupt) String() string {
	switch i {
	case InterruptVblank:
		return "Vblank"
	case InterruptLCDC:
		return "LCDC"
	case InterruptTimer:
		return "Timer"
	case InterruptSerial:
		return "Serial"
	case InterruptKeypad:
		return "Keypad"
	default:
		return "UNKNOWN"
	}
}

