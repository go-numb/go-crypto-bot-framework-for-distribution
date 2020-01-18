package bitflyer

import "sync"

// LogicChecker is check for ignition, par sec.
func (p *Client) LogicChecker() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if !p.Controllers.Profit.IsDo {
			return
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !p.Controllers.Basic.IsDo {
			return
		}

		base := p.E.Price * p.Setting.DiffRatio
		spread := p.E.Spread()
		if spread < base {
			return
		}

		go p.IgniteBasic()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !p.Controllers.Special.IsDo {
			return
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !p.Controllers.VPIN.IsDo {
			return
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if !p.Controllers.Swing.IsDo {
			return
		}

	}()

	wg.Wait()
}
