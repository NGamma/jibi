package jibi

// A ClockType is simply the type used for all clocks
type ClockType uint32

// A Clock sends number of clock cycle since last successful send
// so if a non-blocking send fails, the cycles accumulate
// on successful send the cycles is reset
// sends happen on machine cycle end
type Clock struct {
	v ClockType
	c chan ClockType
}

// NewClock creates a new clock that will send on the provided channel.
func NewClock(c chan ClockType) *Clock {
	return &Clock{ClockType(0), c}
}

// AddCycles tries to send the number of accumulated cycles on the channel,
// if that is successful it resets the accumulation.
func (c *Clock) AddCycles(cycles uint8) {
	c.v += ClockType(cycles)
	//v := uint8(c.v)
	//if c.v > 255 {
	//  v = 255
	//}

	select {
	case c.c <- c.v:
		//c.v -= ClockType(v)
		c.v = 0
	default:
	}
}
