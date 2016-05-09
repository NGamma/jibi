package jibi

import (
	"fmt"
	"time"
)

// Options holds various options.
type Options struct {
	Status   bool
	Skipbios bool
	Render   bool
	Keypad   bool
	Quick    bool
	Squash   bool
	Every    bool
}

// Jibi is the glue that holds everything together.
type Jibi struct {
	O Options

	mmu  Mmu
	cpu  *Cpu
	lcd  Lcd
	gpu  *Gpu
	cart *Cartridge
	kp   *Keypad
}

// New returns a new Jibi in a Paused state.
func New(rom []Byte, options Options) Jibi {
	cart := NewCartridge(rom)
	mmu := NewMmu(cart)
	cpu := NewCpu(mmu, bios)
	lcd := NewLcd(options.Squash)
	gpu := NewGpu(mmu, lcd, cpu.Clock())
	kp := NewKeypad(mmu, options.Keypad)

	if options.Skipbios {
		cpu.RunCommand(CmdUnloadBios, nil)
	}
	if !options.Render {
		lcd.DisableRender()
	}

	return Jibi{options, mmu, cpu, lcd, gpu, cart, kp}
}

// RunCommand displatches a command to the correct piece.
func (j Jibi) RunCommand(cmd Command, resp chan string) {
	if cmd < cmdCPU {
		j.cpu.RunCommand(cmd, resp)
	} else if cmd < cmdGPU {
		j.gpu.RunCommand(cmd, resp)
	} else if cmd < cmdKEYPAD {
		j.kp.RunCommand(cmd, resp)
	} else if cmd < cmdALL {
		j.cpu.RunCommand(cmd, resp)
		j.gpu.RunCommand(cmd, resp)
		j.kp.RunCommand(cmd, resp)
	}
}

// Run starts the Jibi and waits till it ends before returning.
func (j Jibi) Run() {
	// metrics
	cpuClk := j.cpu.Clock()
	resp := make(chan chan ClockType)
	j.cpu.RunCommand(CmdCmdCounter, resp)
	cpuCmds := <-resp
	j.cpu.RunCommand(CmdLoopCounter, resp)
	cpuLoops := <-resp
	j.gpu.RunCommand(CmdCmdCounter, resp)
	gpuCmds := <-resp
	j.gpu.RunCommand(CmdLoopCounter, resp)
	gpuLoops := <-resp
	j.gpu.RunCommand(CmdFrameCounter, resp)
	gpuFrames := <-resp
	j.kp.RunCommand(CmdCmdCounter, resp)
	kpCmds := <-resp
	j.kp.RunCommand(CmdLoopCounter, resp)
	kpLoops := <-resp

	j.Play()
	ticker := time.NewTicker(1 * time.Second)
	tickerC := ticker.C

	var inst chan string
	if j.O.Every {
		respStr := make(chan chan string)
		j.cpu.RunCommand(CmdOnInstruction, respStr)
		inst = <-respStr
		tickerC = nil
	}
	if !j.O.Status {
		tickerC = nil
	}
	var timeout <-chan time.Time
	if j.O.Quick {
		timeout = time.After(2 * time.Second)
	}
	cpuHz := float64(0)
	cpuCps := ClockType(0)
	cpuLps := ClockType(0)
	gpuCps := ClockType(0)
	gpuLps := ClockType(0)
	gpuFps := float64(0)
	kpCps := ClockType(0)
	kpLps := ClockType(0)
	count := float64(-1)
	for running := true; running; {
		select {
		case <-timeout:
			fmt.Println("timeout")
			running = false
		case u := <-inst:
			fmt.Println(u)
		case <-tickerC:
			if count >= 10.0 {
				cpuHz *= 0.9
				gpuFps *= 0.9
				count = 9.0
			}
			cpuCps = 0
			cpuLps = 0
			gpuCps = 0
			gpuLps = 0
			kpCps = 0
			kpLps = 0
			for loop := true; loop; {
				select {
				case t := <-cpuClk:
					cpuHz += float64(t)
				case t := <-cpuCmds:
					cpuCps += t
				case t := <-cpuLoops:
					cpuLps += t
				case t := <-gpuCmds:
					gpuCps += t
				case t := <-gpuLoops:
					gpuLps += t
				case t := <-gpuFrames:
					gpuFps += float64(t)
				case t := <-kpCmds:
					kpCps += t
				case t := <-kpLoops:
					kpLps += t
				default:
					loop = false
				}
			}
			count++

			// skip first tick
			if count > 0 {
				to := time.After(2 * time.Second)
				sc := make(chan string)
				go func() {
					s := ""
					if j.O.Render {
						s = fmt.Sprintf("\x1B[s\x1B[58;0H" +
							"\x1B[K\n" + // cpu instruction
							"\x1B[K\n" + // cpu
							"\x1B[K\n" + // cpu
							"\x1B[K\n" + // cpu flags
							"\x1B[K\n" + // keypad
							"\x1B[K\n" + // metrics 1
							"\x1B[K\n" + // metrics 2
							"\x1B[58;0H")
					}
					s += fmt.Sprintf("%s\n%s\n"+
						"   cpu: %5.2fMhz cpuCps: %8d cpuLps: %8d "+
						"gpuFps: %8.2f gpuCps: %8d gpuLps: %8d\n"+
						" kpCps: %8d  kpLps: %8d "+
						"\n",
						j.cpu, j.kp,
						cpuHz/(1e6*count), cpuCps, cpuLps,
						gpuFps/count, gpuCps, gpuLps,
						kpCps, kpLps)
					if j.O.Render {
						s += fmt.Sprintf("\x1B[u")
					}
					sc <- s
				}()
				select {
				case <-to:
					panic("timeout")
				case s := <-sc:
					fmt.Println(s)
				}
			}
			if j.O.Quick {
				running = false
			}
		}
	}
	ticker.Stop()
	j.Stop()
}

// Play starts the Jibi and returns immediately.
func (j Jibi) Play() {
	j.RunCommand(CmdPlay, nil)
}

// Pause pauses the Jibi and returns immediately.
func (j Jibi) Pause() {
	j.RunCommand(CmdPause, nil)
}

// Stop stops the Jibi and all its goroutines and returns immediately.
func (j Jibi) Stop() {
	j.RunCommand(CmdStop, nil)
}
