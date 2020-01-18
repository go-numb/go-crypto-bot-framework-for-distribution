package bitflyer

import (
	"fmt"
	"time"

	v1 "github.com/go-numb/go-bitflyer/v1"
)

type LogicByBasic struct {
	Name string
}

// IgniteBasic ignition to Logic
func (p *Client) IgniteBasic() {
	p.Logic <- &LogicByBasic{
		Name: "mm_basic",
	}
}

// GetSide return sideInt for this logic
func (logic *LogicByBasic) getSide(isBuy bool) int {
	if isBuy {
		return 1
	}
	return -1
}

// Order do order and execution for MM Basic
func (logic *LogicByBasic) Order(p *Client) {
	if !p.Controllers.Basic.IsOK() {
		return
	}
	defer p.Controllers.Basic.Close()

	if p.isDelay() {
		return
	}

	var (
		isMarket bool
	)

	side := logic.getSide(p.E.IsBuy)
	if side == 0 {
		return
	}

	price, ok := p.setLimitPrice(side)
	if !ok {
		return
	}

	isFull, size := p.O.GetOrderSize(side, p.Setting.Size)
	if isFull {
		return
	}

	if p.Setting.IsTestNet { // simple backtest
		if err := logic.BackTest(p, side, price, size); err != nil {
			p.Logger.Error(err)
		}
		return
	}

	o, err := p.OrderBySimple(isMarket, side, price, size, nil)
	if err != nil {
		p.SetError(true, err)
		return
	}
	p.O.Set(*o)

	// 注文のタイムアウト
	go func() {
		timeout := time.After(time.Duration(CHECKRESOUCEPERIOD*MMBASICCANCELORDER) * time.Second)
		for {
			select {
			case <-timeout:
				p.CancelByID(o.OrderID)
				goto EXIT
			}
		}
	EXIT:
		return
	}()

	// 約定監視
	if !p.isExecution(side, logic.Name, o) {
		if !p.O.IsCancel(o) {
			time.Sleep(p.E.Delay + o.OnAccept)
		}
	}

	return
}

var (
	BUYPRICE, SELLPRICE float64
)

// setLimitPrice 指値価格の決定
func (p *Client) setLimitPrice(side int) (price float64, ok bool) {
	base := p.E.Price * 0.0002
	ok = true

	if 0 < side {
		price = p.E.BestBid - p.E.Spread()
		if BUYPRICE+base > price && BUYPRICE-base < price {
			ok = false // 同価格帯なら拒否
		}
		BUYPRICE = price
	} else if side < 0 {
		price = p.E.BestAsk + p.E.Spread()
		if SELLPRICE+base > price && SELLPRICE-base < price {
			ok = false // 同価格帯なら拒否
		}
		SELLPRICE = price
	}

	return price, ok
}

// BackTest check execution, and set database
func (logic *LogicByBasic) BackTest(p *Client, side int, price, size float64) error {
	o := &Order{
		Side:   v1.ToSide(side),
		Price:  price,
		Size:   size,
		IsDone: false,
	}

	func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		/*
			Timeout: 9sec
			order length: 366,	exec rate: 34.699％
			BUY	866702.6470588231	0.6800000000000004
			SELL	867249.6949152538	0.5900000000000003
			CreateAt: 2020/1/10 3:02:32 JST
		*/
		timeout := time.After(time.Duration(CHECKRESOUCEPERIOD*MMBASICCANCELORDER) * time.Second)
		for {
			select {
			case <-ticker.C:
				if 0 < side {
					if price > p.E.Price {
						fmt.Println("is done")
						o.IsDone = true
						return
					}
				} else if side < 0 {
					if price < p.E.Price {
						fmt.Println("is done")
						o.IsDone = true
						return
					}
				}
			case <-timeout:
				return
			}
		}
	}()

	// BackTest用table
	if err := p.DB.Set(DBTABLEORDERSFORTEST, o); err != nil {
		return err
	}

	return nil
}
