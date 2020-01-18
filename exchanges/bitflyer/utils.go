package bitflyer

import "math"

const float64EqualityThreshold = 1e-9

// isFloatEqual
func isFloatEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

// isDelay check delay for websocket executions delay
func (p *Client) isDelay() bool {
	if DELAYTHRESHHOLD > p.E.Delay {
		return false
	}

	return true
}
