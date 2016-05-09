package jibi

import ()

// A Gpu is the graphics processing unit. It handles drawing the background,
// window and sprites. It also triggers interrutps.
type Gpu struct {
	CommanderInterface

	// 0x0000-0x07FF tile set 1 0-127
	// 0x0800-0x0FFF tile set 1 128-255, set 0 (-1)-(-128)
	// 0x1000-0x17FF tile set 0 0-127
	// 0x1800-0x1BFF tile map 0
	// 0x1C00-0x1FFF tile map 1

	mmu     Mmu
	mmuKeys AddressKeys
	lcd     Lcd
	clk     chan ClockType

	bgBuffer []Byte // 256x256 background 2bit bitmap buffer
	fgBuffer []Byte // 144x160 foreground 2bit bitmap buffer

	// metrics
	frameCounters []*Clock
}

// NewGpu creates a Gpu and starts a goroutine.
func NewGpu(mmu Mmu, lcd Lcd, clk chan ClockType) *Gpu {
	commander := NewCommander("gpu")
	gpu := &Gpu{CommanderInterface: commander,
		mmu: mmu, lcd: lcd, clk: clk,
		bgBuffer: make([]Byte, 256*256),
		fgBuffer: make([]Byte, int(lcdWidth)*int(lcdHeight)),
	}
	cmdHandlers := map[Command]CommandFn{
		CmdFrameCounter: gpu.cmdFrameCounter,
	}
	commander.start(gpu.stateScanlineOam, cmdHandlers, clk)
	mmu.SetGpu(gpu)
	return gpu
}

func (g *Gpu) cmdFrameCounter(resp interface{}) {
	if resp, ok := resp.(chan chan ClockType); !ok {
		panic("invalid command response type")
	} else {
		clk := make(chan ClockType, 1)
		g.frameCounters = append(g.frameCounters, NewClock(clk))
		resp <- clk
	}
}

func (g *Gpu) readByte(addr Worder) Byte {
	return g.mmu.ReadByteAt(addr, g.mmuKeys)
}

func (g *Gpu) writeByte(addr Worder, b Byter) {
	g.mmu.WriteByteAt(addr, b, g.mmuKeys)
}

/*
func paintTile(frameBuffer []Byte, tileData []Byte, x, y uint8, above, xflip, yflip bool, palette Byte) {
	addr := 0
	// convert tile data into 2bpp bitmap
	for yOff := uint8(0); yOff < 8; yOff++ {
		yInd := (uint16(y) + uint16(yOff)) * uint16(256)
		l := tileData[addr]
		h := tileData[addr+1]
		addr += 2

		for xOff := uint8(0); xOff < 8; xOff++ {
			px := (((h >> (7 - xOff)) & 0x01) << 1) + (l>>(7-xOff))&0x01
			ind := uint16(x) + uint16(xOff) + yInd
			if uint32(ind) < uint32(len(frameBuffer)) {
				frameBuffer[ind] = px
			}
		}
	}
}
*/
func (g *Gpu) generateLine(line Byte) []Byte {
	// get background
	// TODO: bg wraps to the same X, not to X+1, same with Y
	scy := g.readByte(AddrSCY)
	scx := g.readByte(AddrSCX)
	offset := uint16(line+scy)*256 + uint16(scx)
	lbs := g.bgBuffer[offset : offset+uint16(lcdWidth)-1]
	// TODO: draw up to 10 sprites

	offset = uint16(line) * uint16(lcdWidth)
	for i := range lbs {
		b := g.fgBuffer[offset+uint16(i)]
		if b > 0 {
			lbs[i] = b
		}
	}
	return lbs
}

type sprite struct {
	t tile
	x uint8
	y uint8
	// TODO: implement attribs
}

func newSprite(spriteData, tileData, palette []Byte) sprite {
	y := uint8(spriteData[0]) - 16
	x := uint8(spriteData[1]) - 8
	t := newTile(tileData, palette)
	spr := sprite{t, x, y}
	return spr
}

func (spr sprite) Paint(buffer []Byte) {
	spr.t.Paint(buffer, spr.x, spr.y)
}

func (g *Gpu) getSprites(sizeId Byte) []sprite {
	height := uint8(8)
	if sizeId == 1 {
		height = 16
	}
	sprites := []sprite{}
	obp0 := g.readByte(AddrOBP0)
	obp1 := g.readByte(AddrOBP1)
	for spriteAddr := AddrOam; spriteAddr < AddrOamEnd; spriteAddr += 4 {
		spriteData := make([]Byte, 4)
		spriteData[0] = g.readByte(spriteAddr)
		spriteData[1] = g.readByte(spriteAddr + 1)
		tileInd := g.readByte(spriteAddr + 2)
		if height == 16 {
			tileInd = tileInd & 0xFE
		}
		spriteData[2] = tileInd
		spriteData[3] = g.readByte(spriteAddr + 3)
		addrTile := 0x8800 + Word(Byte(tileInd+0x80))*16
		obp := Byte(0)
		if spriteData[3]&0x10 == 0 {
			obp = obp0
		} else {
			obp = obp1
		}
		palette := byteToPalette(obp)
		tileData := make([]Byte, height*2)
		for i := range tileData {
			tileData[i] = g.readByte(addrTile)
			addrTile++
		}
		sprites = append(sprites, newSprite(spriteData, tileData, palette))
	}
	return sprites
}

type tile struct {
	bitmap []Byte // 2bpp bitmap
}

func newTile(tileData []Byte, palette []Byte) tile {
	height := uint8(len(tileData) / 2)
	bitmap := []Byte{}
	// 8x8 tiles
	// convert tile data into 2bpp bitmap
	addr := 0
	xMax := uint8(len(tileData) / 2)
	for yOff := uint8(0); yOff < height; yOff++ {
		l := tileData[addr]
		h := tileData[addr+1]
		addr += 2

		for xOff := uint8(0); xOff < xMax; xOff++ {
			px := (((h >> (7 - xOff)) & 0x01) << 1) + (l>>(7-xOff))&0x01
			bitmap = append(bitmap, palette[px])
		}
	}
	return tile{bitmap}
}

func (t tile) Paint(buffer []Byte, x, y uint8) {
	width := uint16(0)
	if len(buffer) == 65536 {
		width = uint16(256)
	} else if len(buffer) == int(lcdWidth)*int(lcdHeight) {
		width = uint16(lcdWidth)
	} else {
		panic("unknown buffer type")
	}
	yMax := uint16(len(t.bitmap) / 8)
	addr := 0
	for yOff := uint16(0); yOff < yMax; yOff++ {
		for xOff := uint16(0); xOff < 8; xOff++ {
			px := t.bitmap[addr]
			addr++
			buffOff := uint16(x) + xOff + (uint16(y)+yOff)*width
			if int(buffOff) < len(buffer) {
				buffer[buffOff] = px
			}
		}
	}
}

func (g *Gpu) getWinTiles(tilemap, tileset Byte, palette []Byte) []tile {
	addrTilemap := Word(0x9800)
	if tilemap == 1 {
		addrTilemap = 0x9C00
	}
	addrTileset := Word(0x8800)
	if tileset == 1 {
		addrTileset = 0x8000
	}

	tiles := []tile{}
	for t := Word(0x0000); t < 0x0400; t++ {
		tileData := make([]Byte, 16)
		tileInd := g.readByte(addrTilemap + t)
		addrTile := Word(0)
		if tileset == 0 {
			addrTile = addrTileset + Word(Byte(tileInd+0x80))*16
		} else {
			addrTile = addrTileset + Word(tileInd)*16
		}
		for i := Word(0); i < 16; i++ {
			tileData[i] = g.readByte(addrTile + i)
		}
		tiles = append(tiles, newTile(tileData, palette))
	}

	return tiles
}

func (g *Gpu) getBgTiles(tilemap, tileset Byte, palette []Byte) []tile {
	addrTilemap := Word(0x9800)
	if tilemap == 1 {
		addrTilemap = 0x9C00
	}
	addrTileset := Word(0x8800)
	if tileset == 1 {
		addrTileset = 0x8000
	}

	tiles := []tile{}
	for t := Word(0x0000); t < 0x0400; t++ {
		tileData := make([]Byte, 16)
		tileInd := g.readByte(addrTilemap + t)
		addrTile := Word(0)
		if tileset == 0 {
			addrTile = addrTileset + Word(Byte(tileInd+0x80))*16
		} else {
			addrTile = addrTileset + Word(tileInd)*16
		}
		for i := Word(0); i < 16; i++ {
			tileData[i] = g.readByte(addrTile + i)
		}
		tiles = append(tiles, newTile(tileData, palette))
	}

	return tiles
}

func byteToPalette(p Byte) []Byte {
	return []Byte{p & 0x03, p & 0x0C >> 2, p & 0x30 >> 4, p & 0xC0 >> 6}
}

func (g *Gpu) generateFrame() {
	g.lockAddr(AddrVRam) // TODO: this should be in scanline vram
	defer g.unlockAddr(AddrVRam)

	// clear foreground buffer
	for i := range g.fgBuffer {
		g.fgBuffer[i] = 0
	}

	lcdc := g.readByte(AddrLCDC)
	// read in map, tileset data
	windowTilemap := (lcdc & 0x40) >> 6
	windowDisplay := lcdc&0x20 == 0x20
	bgTileset := (lcdc & 0x10) >> 4
	bgTilemap := (lcdc & 0x08) >> 3
	objSpriteSize := (lcdc & 0x04) >> 2
	objDisplay := lcdc&0x02 == 0x02
	bgWinDisplay := lcdc&0x01 == 0x01

	// draw background
	if bgWinDisplay {
		x := uint8(0)
		y := uint8(0)
		bgp := g.readByte(AddrBGP)
		palette := byteToPalette(bgp)
		for _, bgtile := range g.getBgTiles(bgTilemap, bgTileset, palette) {
			bgtile.Paint(g.bgBuffer, x, y)
			x += 8
			if x == 0 {
				y += 8
			}
		}

		if windowDisplay {
			// TODO: this has to be handled line by line
			// wx is read on screen redraw and after a scan line interrupt
			// wy is read on screen redraw
			wx := g.readByte(AddrWX)
			wy := g.readByte(AddrWY)
			x = uint8(wx) - 7
			y = uint8(wy)
			palette := byteToPalette(bgp)
			for _, wintile := range g.getWinTiles(windowTilemap, bgTileset, palette) {
				wintile.Paint(g.fgBuffer, x, y)
				x += 8
				if x == 0 {
					y += 8
				}
			}
		}
	}

	// draw sprites (oam)
	if objDisplay {
		g.lockAddr(AddrOam) // TODO: this should be in scanline oam
		sprites := g.getSprites(objSpriteSize)
		g.unlockAddr(AddrOam)
		for _, spr := range sprites {
			spr.Paint(g.fgBuffer)
		}
	}
	/*
		// draw sprites (oam)
		if objDisplay {
			addrTileset := Word(0x8000)
			oamaddr := Word(0xFE00)
			spriteSize := Word(16)
			if bgSpriteSize == 1 {
				spriteSize = 32
			}
			tileData := make([]Byte, spriteSize)

			for sprite := Word(0x0000); sprite < 0x00A0; sprite += 4 {
				x := uint8(g.readByte(oamaddr+sprite) - 8)
				y := uint8(g.readByte(oamaddr+sprite+1) - 16)
				tile := g.readByte(oamaddr + sprite + 2)
				if bgSpriteSize == 1 {
					tile = tile & 0xEF
				}
				attr := g.readByte(oamaddr + sprite + 3)
				above := attr&0x80 == 0
				yflip := attr&0x40 == 0x80
				xflip := attr&0x20 == 0x40
				palette := attr & 0x10 >> 1
				ind := addrTileset + Word(tile)*spriteSize
				for j := Word(0); j < spriteSize; j++ {
					tileData[j] = g.readByte(ind + j)
				}
				paintTile(g.frameBuffer, tileData, x, y, above, xflip, yflip, palette)
			}
		}
	*/
}

func (g *Gpu) lockAddr(addr Worder) {
	g.mmuKeys = g.mmu.LockAddr(addr, g.mmuKeys)
}

func (g *Gpu) unlockAddr(addr Worder) {
	g.mmuKeys = g.mmu.UnlockAddr(addr, g.mmuKeys)
}

func (g *Gpu) stateScanlineOam(first bool, t uint32) (CommanderStateFn, bool, uint32, uint32) {
	g.lockAddr(AddrGpuRegs)
	defer g.unlockAddr(AddrGpuRegs)
	if first {
		//g.lockAddr(AddrOam)
		stat := g.readByte(AddrSTAT)
		stat = stat&0x7C | 0x2 // mode 2
		ly := g.readByte(AddrLY)
		lyc := g.readByte(AddrLYC)
		if ly == lyc {
			stat |= 0x04
		} else {
			stat &= (0x04 ^ 0xFF)
		}
		g.writeByte(AddrSTAT, stat)
		if (ly == lyc) && (stat&(0x40|0x20) == (0x40 | 0x20)) { // lyc=ly and mode 2
			g.mmu.SetInterrupt(InterruptLCDC, g.mmuKeys)
		}
	}
	if t >= 80 {
		t -= 80
		//g.unlockAddr(AddrOam)
		return g.stateScanlineVram, true, t, 172
	}
	return g.stateScanlineOam, false, t, 80
}

func (g *Gpu) stateScanlineVram(first bool, t uint32) (CommanderStateFn, bool, uint32, uint32) {
	g.lockAddr(AddrGpuRegs)
	defer g.unlockAddr(AddrGpuRegs)
	if first {
		//g.lockAddr(AddrVRam)
		stat := g.readByte(AddrSTAT)
		stat = stat&0x7C | 0x3 // mode 3
		g.writeByte(AddrSTAT, stat)
		ly := g.readByte(AddrLY)
		g.lcd.DrawLine(g.generateLine(ly))
	}
	if t >= 172 {
		t -= 172
		//g.unlockAddr(AddrVRam)
		return g.stateHblank, true, t, 204
	}
	if !first {
		panic("wasted gpu cycle")
	}
	return g.stateScanlineVram, false, t, 172
}

func (g *Gpu) stateHblank(first bool, t uint32) (CommanderStateFn, bool, uint32, uint32) {
	g.lockAddr(AddrGpuRegs)
	defer g.unlockAddr(AddrGpuRegs)
	if first {
		stat := g.readByte(AddrSTAT)
		stat = stat&0x7C | 0x1 // mode 1
		ly := g.readByte(AddrLY)
		lyc := g.readByte(AddrLYC)
		if ly == lyc {
			stat |= 0x04
		} else {
			stat &= (0x04 ^ 0xFF)
		}
		g.writeByte(AddrSTAT, stat)
		if (ly == lyc) && (stat&(0x40|0x10) == (0x40 | 0x10)) { // lyc=ly and mode 1
			g.mmu.SetInterrupt(InterruptLCDC, g.mmuKeys)
		}
	}
	if t >= 204 {
		t -= 204
		ly := g.readByte(AddrLY)
		ly++
		g.mmu.WriteByteAt(AddrLY, ly, g.mmuKeys|AddressKeys(abElevated))
		if ly == lcdHeight-1 {
			return g.stateVblank, true, t, 456
		}
		return g.stateScanlineOam, true, t, 80
	}
	if !first {
		panic("wasted gpu cycle")
	}
	return g.stateHblank, false, t, 204
}

func (g *Gpu) stateVblank(first bool, t uint32) (CommanderStateFn, bool, uint32, uint32) {
	g.lockAddr(AddrGpuRegs)
	defer g.unlockAddr(AddrGpuRegs)
	if first {
		stat := g.readByte(AddrSTAT)
		stat = stat&0x7C | 0x0 // mode 0
		ly := g.readByte(AddrLY)
		lyc := g.readByte(AddrLYC)
		if ly == lyc {
			stat |= 0x04
		} else {
			stat &= (0x04 ^ 0xFF)
		}
		g.writeByte(AddrSTAT, stat)
		if (ly == lyc) && (stat&(0x40|0x04) == (0x40 | 0x04)) { // lyc=ly and mode 0
			g.mmu.SetInterrupt(InterruptLCDC, g.mmuKeys)
		}
		g.mmu.SetInterrupt(InterruptVblank, g.mmuKeys)
		g.lcd.Blank()
		g.generateFrame()
		for _, clk := range g.frameCounters {
			clk.AddCycles(1)
		}
	}
	if t >= 456 {
		t -= 456
		ly := g.readByte(AddrLY)
		ly++
		if ly > lcdHeight-1+10 {
			ly = 0
			g.mmu.WriteByteAt(AddrLY, ly, g.mmuKeys|AddressKeys(abElevated))
			return g.stateScanlineOam, true, t, 80
		}
		g.mmu.WriteByteAt(AddrLY, ly, g.mmuKeys|AddressKeys(abElevated))
		return g.stateVblank, false, t, 456
	}
	if !first {
		panic("wasted gpu cycle")
	}
	return g.stateVblank, false, t, 456
}
