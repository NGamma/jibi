package jibi

import (
	"fmt"
)

// A Command is any command the is read by a Commander
type Command int

// A list of all commands and command indicators.
const (
	CmdNil Command = iota

	CmdUnloadBios
	CmdSetInterrupt
	CmdClockAccumulator // accumulating clock
	CmdOnInstruction    // blocking clock channel that ticks after every instruction
	cmdCPU

	CmdFrameCounter
	cmdGPU

	CmdKeyDown
	CmdKeyUp
	CmdKeyCheck
	cmdKEYPAD

	CmdCmdCounter  // a clock that outputs number of commands processed
	CmdLoopCounter // a clock that outputs number of loops run
	CmdString
	CmdPlay
	CmdPause
	CmdStop
	cmdALL
)

func (c Command) String() string {
	switch c {
	case CmdNil:
		return "CmdNil"
	case CmdUnloadBios:
		return "CmdUnloadBios"
	case CmdClockAccumulator:
		return "CmdClockAccumulator"
	case CmdOnInstruction:
		return "CmdOnInstruction"
	case cmdCPU:
		return "cmdCPU"
	case CmdFrameCounter:
		return "CmdFrameCounter"
	case cmdGPU:
		return "cmdGPU"
	case CmdKeyDown:
		return "CmdKeyDown"
	case CmdKeyUp:
		return "CmdKeyUp"
	case CmdKeyCheck:
		return "CmdKeyCheck"
	case cmdKEYPAD:
		return "cmdKEYPAD"
	case CmdCmdCounter:
		return "CmdCmdCounter"
	case CmdLoopCounter:
		return "CmdLoopCounter"
	case CmdString:
		return "CmdString"
	case CmdPlay:
		return "CmdPlay"
	case CmdPause:
		return "CmdPause"
	case CmdStop:
		return "CmdStop"
	case cmdALL:
		return "cmdALL"
	}
	return fmt.Sprintf("CmdUNKNOWN-%d", int(c))
}

// A CommandResponse holds a command and response data (usually a channel).
type CommandResponse struct {
	cmd  Command
	resp interface{}
}

// A CommanderStateFn is a chained state function that returns the next state.
type CommanderStateFn func(bool, uint32) (CommanderStateFn, bool, uint32, uint32)

// A CommanderInterface is an interface that lists what a Commander implements
// so it can be used as an emebedded type.
type CommanderInterface interface {
	RunCommand(Command, interface{})
	start(CommanderStateFn, map[Command]CommandFn, chan ClockType)
	yield()
	play()
	pause()
}

// A Commander handles an event loop in a goroutine that processes and
// dispatches commands.
type Commander struct {
	name         string
	c            chan CommandResponse
	cmdCounters  []*Clock
	loopCounters []*Clock
	playing      bool
	running      bool
	handlerFns   map[Command]CommandFn
}

// NewCommander returns a new named Commander object.
func NewCommander(name string) *Commander {
	c := &Commander{name,
		make(chan CommandResponse, 1024), // HACK
		nil, nil, false, false, nil,
	}
	return c
}

// start creates the goroutine.
func (c *Commander) start(state CommanderStateFn, handlerFns map[Command]CommandFn, clk chan ClockType) {
	c.handlerFns = handlerFns
	go c.loopCommander(state, clk)
}

// yield gives the commander an opportunity to process any pending commands
// TODO: verify that there is 0 change of deadlock after a yield
func (c *Commander) yield() {
	c.processCommands()
}

// RunCommand queues the given command for processing.
func (c *Commander) RunCommand(cmd Command, resp interface{}) {
	c.c <- CommandResponse{cmd, resp}
}

func (c *Commander) String() string {
	return c.name
}

// A MemoryCommander is the generic interface for something that is both a
// MemoryDevice and a Commander embedded interface.
type MemoryCommander interface {
	ReadByteAt(Worder, chan Byte)
	WriteByteAt(Worder, Byter)
	CommanderInterface
}

// A CommandFn is a map from the Command to the handler function.
type CommandFn func(interface{})

func nilFunc(a int) int {
	return a + a
}

func (c *Commander) loopCommander(state CommanderStateFn, clk chan ClockType) {
	c.playing = false
	c.running = true
	first := true
	t := uint32(0)
	tnext := uint32(0) // time needed to run next state
	var cmdr CommandResponse
	to := ClockType(0)
	for c.running {
		cmdr.cmd = CmdNil
		for _, clk := range c.loopCounters {
			clk.AddCycles(1)
		}
		if !c.playing || state == nil {
			cmdr = <-c.c
			c.processCommand(cmdr)
		} else if t >= tnext {
			// we have enough cycles to run the next state without waiting for the clock
			select {
			case cmdr = <-c.c:
			default:
			}
			c.processCommand(cmdr)
		} else if t < tnext {
			if clk == nil {
				panic(fmt.Sprintf("Commander %s requires a clock", c))
			}
			select {
			case cmdr = <-c.c:
			case to = <-clk:
				t += uint32(to)
			}
			c.processCommand(cmdr)
		}
		if state != nil && c.playing && (t >= tnext || first) {
			state, first, t, tnext = state(first, t)
		} else if !c.playing {
			t = 0
		}
	}
}

func (c *Commander) processCommands() {
	var cmdr CommandResponse

	for loop := true; loop; {
		cmdr.cmd = CmdNil
		select {
		case cmdr = <-c.c:
		default:
			loop = false
		}
		c.processCommand(cmdr)
	}
}

func (c *Commander) processCommand(cmdr CommandResponse) {
	if cmdr.cmd != CmdNil {
		for _, clk := range c.cmdCounters {
			clk.AddCycles(1)
		}
		if cmdr.cmd == CmdPlay {
			c.playing = true
		} else if cmdr.cmd == CmdPause {
			c.playing = false
		} else if cmdr.cmd == CmdCmdCounter {
			c.cmdCmdCounter(cmdr.resp)
		} else if cmdr.cmd == CmdLoopCounter {
			c.cmdLoopCounter(cmdr.resp)
		} else {
			if _, ok := c.handlerFns[cmdr.cmd]; !ok {
				if cmdr.cmd != CmdStop {
					panic(fmt.Sprintf("Commander %s requires handler for %s", c, cmdr.cmd))
				}
			} else {
				c.handlerFns[cmdr.cmd](cmdr.resp)
			}
			if cmdr.cmd == CmdStop {
				c.running = false
				c.playing = false
			}
		}
	}
}

func (c *Commander) cmdCmdCounter(resp interface{}) {
	if resp, ok := resp.(chan chan ClockType); !ok {
		panic("invalid command response type")
	} else {
		clk := make(chan ClockType, 1)
		c.cmdCounters = append(c.cmdCounters, NewClock(clk))
		resp <- clk
	}
}

func (c *Commander) cmdLoopCounter(resp interface{}) {
	if resp, ok := resp.(chan chan ClockType); !ok {
		panic("invalid command response type")
	} else {
		clk := make(chan ClockType, 1)
		c.loopCounters = append(c.loopCounters, NewClock(clk))
		resp <- clk
	}
}

func (c *Commander) play() {
	c.playing = true
}

func (c *Commander) pause() {
	c.playing = false
}
