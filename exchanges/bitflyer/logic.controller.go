package bitflyer

import "sync"

// Controllers is 各種ロジックの同時稼働数管理
type Controllers struct {
	Profit  *Controller
	Basic   *Controller
	Special *Controller
	VPIN    *Controller
	Swing   *Controller
}

// Controller ロジックの稼働数量管理
type Controller struct {
	mux   sync.Mutex
	IsDo  bool
	Count int
	Limit int
}

// IsOK check worker limit, if ok subtracte count
func (p *Controller) IsOK() bool {
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.Count < p.Limit {
		p.Count++
		return true
	}

	return false
}

// Close pull buck count
func (p *Controller) Close() {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.Count--
	if p.Count < 0 {
		p.Count = 0
	}
}
