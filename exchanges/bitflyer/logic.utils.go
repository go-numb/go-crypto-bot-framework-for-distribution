package bitflyer

import (
	"math"
	"time"

	"github.com/go-numb/go-bitflyer/v1/private/positions"
)

// GetOrderSize is new Size adjustor
// include go-bitflyer private positions
func (p *Orders) GetOrderSize(side int, sizes []float64) (bool, float64) {
	limit := sizes[len(sizes)-1]
	m := positions.NewT(sizes[0], limit)

	// 売りならマイナスが入る
	m.Set(p.Size())

	return m.Lot(side, TANHTENSION)
}

// isExecution check if order was done
func (p *Client) isExecution(side int, oType string, o *Order) bool {
	var (
		isDone   bool
		isCancel bool
	)

	// check
	check := time.NewTicker(10 * time.Millisecond)
	defer check.Stop()
	// timeout
	timeout := time.After(time.Duration(WAITEXECUTE) * time.Second)

	for { // 約定検出
		select {
		case <-check.C:
			v, ok := p.O.Positions.Load(o.OrderID)
			if ok { // 約定あり
				pos, ok := v.(Order)
				if ok { // 型チェック
					// // 下桁を捨てないと誤差で  == しない
					if isFloatEqual(o.Size, pos.Size) { // 完全約定
						// p.O.Rate.Set(false, o)
						isDone = true
						goto EXIT
					}

				}
			}

			// AllCancelなどの実行が関数外部から行われた
			if p.O.IsCancel(o) {
				isDone = true
				isCancel = true
				goto EXIT
			}

		case <-timeout:
			goto EXIT
		}
	}

EXIT:

	if !isDone { // 一定時間経過後に約定しなければキャンセルしスレッドから離脱
		go p.DB.Set(DBTABLEORDERS, o)
		p.CancelByID(o.OrderID)
		for i := 0; i < 30; i++ { // キャンセル検出
			time.Sleep(time.Second)
			if p.O.IsCancel(o) {
				break
			}
		}

		return false
	}

	if !isCancel {
		o.IsDone = true
	}
	go p.DB.Set(DBTABLEORDERS, o)
	return true
}

// isDistance 指値からLTPが離れた指値をキャンセルする
func (p *Client) isDistance(side int, add, price float64) bool {
	if 0 < side {
		if p.E.Price > price+math.Abs(add*2) {
			return true
		}
		return false
	}

	if p.E.Price < price-math.Abs(add*2) {
		return true
	}
	return false
}
