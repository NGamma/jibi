package jibi

import (
	"fmt"
)

const (
	lcdWidth  Byte = 160
	lcdHeight Byte = 144
)

type Lcd interface {
	DrawLine(bl []Byte)
	Blank()
	DisableRender()
}

// An LcdASCII outputs as ascii characters to the terminal.
type LcdASCII struct {
	dr           bool
	prevLine     []Byte
	lineIndex    uint8
	prevDrawLine uint8
	squash       bool
}

func NewLcd(squash bool) Lcd {
	return &LcdASCII{squash: squash}
}

// DrawLine draws the Byte Slice to the current line index, then advances the
// index.
func (lcd *LcdASCII) DrawLine(bl []Byte) {

	blO := make([]Byte, len(bl))
	copy(blO, bl)
	drawLine := uint8(float64(lcd.lineIndex) * 50.0 / float64(lcdHeight))
	if lcd.squash {
		if drawLine == lcd.prevDrawLine && lcd.lineIndex != 0 {
			// compress previous line and this line into one
			for i := range bl {
				bl[i] = bl[i] | (lcd.prevLine[i]&0xC0 | (lcd.prevLine[i]&0x03)<<2)
			}
		}
	}
	ls := ""
	for _, c := range bl {
		o := " "
		if c == 1 {
			if lcd.squash {
				o = " " // 0001
			} else {
				o = "."
			}
		} else if c == 2 {
			if lcd.squash {
				o = "." // 0010
			} else {
				o = "_"
			}
		} else if c == 3 {
			if lcd.squash {
				o = "." // 0011
			} else {
				o = "*"
			}
		} else if c == 4 {
			o = " " // 0100
		} else if c == 5 {
			o = " " // 0101
		} else if c == 6 {
			o = "." //0110
		} else if c == 7 {
			o = "." // 0111
		} else if c == 8 {
			o = "'" // 1000
		} else if c == 9 {
			o = "'" // 1001
		} else if c == 10 {
			o = ":" // 1010
		} else if c == 11 {
			o = ":" // 1011
		} else if c == 12 {
			o = "'" // 1100
		} else if c == 13 {
			o = "'" // 1101
		} else if c == 14 {
			o = ":" // 1110
		} else if c == 15 {
			o = ":" // 1111
		}
		ls += o
	}
	if lcd.dr == false {
		if lcd.squash {
			fmt.Printf("\x1B[%d;H%s", drawLine, ls)
		} else {
			if lcd.lineIndex < 50 {
				fmt.Println(ls)
			}
		}
	}

	lcd.prevDrawLine = drawLine
	lcd.prevLine = blO
	lcd.lineIndex++
}

// Blank moves the cursor to the upper left.
func (lcd *LcdASCII) Blank() {
	if lcd.dr == false {
		fmt.Print("\x1B[0;0H")
	}
	lcd.lineIndex = 0
}

// DisableRender turns off rendering of lines. Only use while Paused.
func (lcd *LcdASCII) DisableRender() {
	lcd.dr = true
}
