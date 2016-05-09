package jibi

import (
	"fmt"
)

// A Cartridge holds the game rom as well as information about the rom
// capabilities.
type Cartridge struct {
	Rom []Byte

	// rom info
	name    string
	color   bool
	super   bool
	ct      cartridgeType
	romSize cartridgeRomSize
	ramSize cartridgeRamSize
}

// NewCartridge reads and parses a rom and returns a new cartridge object.
func NewCartridge(rom []Byte) *Cartridge {
	name := ""
	for _, c := range rom[0x0134 : 0x0142+1] {
		if c == 0 {
			break
		}
		name += string(c)
	}
	romN := make([]Byte, 0x10000)
	copy(romN, rom)
	color := rom[0x0143] == 0x80
	super := rom[0x0146] == 0x03
	ct := cartridgeType(rom[0x0147])
	romSize := cartridgeRomSize(rom[0x0148])
	ramSize := cartridgeRamSize(rom[0x0149])
	cart := &Cartridge{romN, name, color, super, ct, romSize, ramSize}
	return cart
}

func (c *Cartridge) String() string {
	return fmt.Sprintf(`name: %s
romSize: %s
ramSize: %s
color: %v
super: %v
type: %s`, c.name, c.romSize, c.ramSize, c.color, c.super, c.ct)
}

type cartridgeType uint8

func (ct cartridgeType) String() string {
	switch ct {
	case 0x00:
		return "00-ROM"
	case 0x01:
		return "01-ROM+MBC1"
	case 0x02:
		return "02-ROM+MBC1+RAM"
	case 0x03:
		return "03-ROM+MBC1+RAM+BATT"
	case 0x05:
		return "05-ROM+MBC2"
	case 0x06:
		return "06-ROM+MBC2+BATT"
	case 0x08:
		return "08-ROM+RAM"
	case 0x09:
		return "09-ROM+RAM+BATT"
	case 0x0B:
		return "0B-ROM+MMMO1"
	case 0x0C:
		return "0C-ROM+MMMO1+SRAM"
	case 0x0D:
		return "0D-ROM+MMMO1+SRAM+BATT"
	case 0x0F:
		return "0F-ROM+MBC3+TIMER+BATT"
	case 0x10:
		return "10-ROM+MBC3+TIMER+RAM+BATT"
	case 0x11:
		return "11-ROM+MBC3"
	case 0x12:
		return "12-ROM+MBC3+RAM"
	case 0x13:
		return "13-ROM+MBC3+RAM+BATT"
	case 0x19:
		return "19-ROM+MBC5"
	case 0x1A:
		return "1A-ROM+MBC5+RAM"
	case 0x1B:
		return "1B-ROM+MBC5+RAM+BATT"
	case 0x1C:
		return "1C-ROM+MBC5+RUMBLE"
	case 0x1D:
		return "1D-ROM+MBC5+RUMBLE+SRAM"
	case 0x1E:
		return "1E-ROM+MBC5+RUMBLE+SRAM+BATT"
	case 0x1F:
		return "1F-PocketCamera"
	case 0xFD:
		return "FD-BandaiTAMA5"
	case 0xFE:
		return "FE-HudsonHuC_3"
	default:
		return fmt.Sprintf("%0X-UNKNOWN", ct)
	}
}

type cartridgeRomSize uint8

func (cs cartridgeRomSize) banks() int {
	switch cs {
	case 0x00:
		return 2
	case 0x01:
		return 4
	case 0x02:
		return 8
	case 0x03:
		return 16
	case 0x04:
		return 32
	case 0x05:
		return 64
	case 0x06:
		return 128
	case 0x52:
		return 72
	case 0x53:
		return 80
	case 0x54:
		return 96
	}
	return 0
}

func (cs cartridgeRomSize) String() string {
	return fmt.Sprintf("%02X-%dKbit,%dKByte,%dbanks",
		uint8(cs), cs.banks()*128, cs.banks()*16, cs.banks())
}

type cartridgeRamSize uint8

func (cs cartridgeRamSize) banks() int {
	switch cs {
	case 0x00:
		return 0
	case 0x01:
		return 1
	case 0x02:
		return 2
	case 0x03:
		return 4
	case 0x04:
		return 16
	}
	return 0
}

func (cs cartridgeRamSize) String() string {
	return fmt.Sprintf("%02X-%dKbit,%dKByte,%dbanks",
		uint8(cs), cs.banks()*128, cs.banks()*16, cs.banks())
}
