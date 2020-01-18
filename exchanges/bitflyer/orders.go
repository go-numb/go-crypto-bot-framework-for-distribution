package bitflyer

import (
	"fmt"
	"github.com/go-numb/go-bitflyer/v1/jsonrpc"
	"math"
	"strings"
	"sync"
	"time"

	v1 "github.com/go-numb/go-bitflyer/v1"
	"github.com/go-numb/go-bitflyer/v1/private/orders/single"
	"github.com/go-numb/go-bitflyer/v1/types"
	"github.com/pkg/errors"

	"github.com/go-numb/go-bitflyer/v1/public/executions"
)

// Orders is orders/positions struct
type Orders struct {
	// Rate *Rate // 約定率

	LastBuyOnAccept  float64
	LastSellOnAccept float64

	Orders    *sync.Map
	Cancels   *sync.Map
	Positions *sync.Map

	Result chan string
}

// NewOrders managed Orders&Poistions
func NewOrders() *Orders {
	return &Orders{
		Orders:    new(sync.Map),
		Cancels:   new(sync.Map),
		Positions: new(sync.Map),

		Result: make(chan string),
	}
}

// Order informations
type Order struct {
	OrderID string
	Side    string
	Price   float64
	Size    float64
	IsDone  bool

	// 注文からAPI返り値までの時間
	OnAccept time.Duration
}

// OrderBySimple is order for bitflyer api
func (p *Client) OrderBySimple(isMarket bool, side int, price, size float64, timeInForce *string) (*Order, error) {
	if err := p.Private.CheckForOrder(); err != nil {
		return nil, err
	}
	start := time.Now()

	// TimeInForce引数があれば
	var tif = "GTC"
	if timeInForce != nil {
		tif = *timeInForce
	}

	// fmt.Printf("%f, ------ %f\n", size, v1.ToSize(size))
	req := &single.Request{
		ProductCode:    types.ProductCode(p.Setting.Code),
		ChildOrderType: v1.ToType(isMarket),
		Side:           v1.ToSide(side),
		Price:          v1.ToPrice(price),
		Size:           v1.ToSize(size),
		MinuteToExpire: p.Setting.ExpireTime,
		TimeInForce:    tif,
	}

	o, res, err := p.C.OrderSingle(req)
	if err != nil {
		return nil, errors.Wrapf(err, "API Remain: %d(%s), size: %.4f, %+v", p.Private.RemainForOrder, p.Private.ResetForOrder.Format("15:04:05"), p.O.Size(), req)
	}
	if res == nil {
		return nil, fmt.Errorf("API response is nil, API Remain: %d(%s), size: %.4f, %+v", p.Private.RemainForOrder, p.Private.ResetForOrder.Format("15:04:05"), p.O.Size(), req)
	}
	defer res.Body.Close()

	// 返り値からAPILIMITを取得
	p.Private.FromHeader(res.Header)

	// 注文開始から返り値取得までの時間を取得
	return ordered(o.ChildOrderAcceptanceId, req, start), nil
}

// ordered return time
func ordered(orderID string, o *single.Request, start time.Time) *Order {
	return &Order{ // OrderIDを保存
		OrderID:  orderID,
		Side:     o.Side,
		Price:    o.Price,
		Size:     o.Size,
		OnAccept: time.Now().Sub(start),
	}
}

// Set is {orderID: order}でmap
func (p *Orders) Set(o Order) {
	p.Orders.Store(o.OrderID, o)
}

// Check is check ws.Exec by orderID
func (p *Orders) Check(e []executions.Execution) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := range e {
			sid := e[i].BuyChildOrderAcceptanceID
			v, isThereB := p.Orders.Load(sid)
			if isThereB {
				pos, isThere := p.Positions.Load(sid)
				if !isThere { // 建玉にデータがない場合
					o, ok := v.(Order)
					if !ok {
						continue
					}
					if isFloatEqual(o.Size, e[i].Size) { // 注文枚数全約定なら
						p.Orders.Delete(sid)
						p.Positions.Store(sid, o)
					} else { // 部分約定なら注文を残しつつ、sizeを減らす
						o.Size -= e[i].Size
						p.Orders.Store(sid, o) // Orderは約定分減らしてset
						o.Size = e[i].Size
						p.Positions.Store(sid, o) // Positionsは約定した分でset
					}

					continue
				}

				// すでに建玉がある場合はsize追加
				po, ok := pos.(Order)
				if !ok {
					continue
				}
				po.Size += e[i].Size // 既存のものに約定を追加して再登録
				p.Positions.Store(sid, po)

				// 注文
				o, ok := v.(Order)
				if !ok {
					continue
				}

				if o.Size <= e[i].Size {
					p.Orders.Delete(sid)
				} else { // 部分約定なら戻す
					o.Size -= e[i].Size
					p.Orders.Store(sid, o)
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := range e {
			sid := e[i].SellChildOrderAcceptanceID
			v, isThereS := p.Orders.Load(sid)
			if isThereS {
				pos, isThere := p.Positions.Load(sid)
				if !isThere { // 建玉にデータがない場合
					o, ok := v.(Order)
					if !ok {
						continue
					}
					if isFloatEqual(o.Size, e[i].Size) { // 注文枚数全約定なら
						p.Orders.Delete(sid)
						p.Positions.Store(sid, o)
					} else { // 部分約定なら注文を残しつつ、sizeを減らす
						o.Size -= e[i].Size
						p.Orders.Store(sid, o) // Orderは約定分減らしてset
						o.Size = e[i].Size
						p.Positions.Store(sid, o) // Positionsは約定した分でset
					}

					continue
				}

				// すでに建玉がある場合はsize追加
				po, ok := pos.(Order)
				if !ok {
					continue
				}
				po.Size += e[i].Size
				p.Positions.Store(sid, po)

				// 注文
				o, ok := v.(Order)
				if !ok {
					continue
				}

				if o.Size <= e[i].Size {
					p.Orders.Delete(sid)
				} else { // 部分約定なら戻す
					o.Size -= e[i].Size
					p.Orders.Store(sid, o)
				}
			}
		}
	}()

	wg.Wait()
}

// CheckByPrivateWs get cancel and expire, delete order in orders and set cancels
func (p *Orders) CheckByPrivateWs(childorders []jsonrpc.WsResponceForChildEvent) {
	// EventType
	// ORDER, ORDER_FAILED, CANCEL, CANCEL_FAILED, EXECUTION, EXPIRE
	for i := range childorders {
		switch childorders[i].EventType {
		// case "ORDER":
		// case "ORDER_FAILED":
		// 	p.getCanceler(childorders[i])
		case "CANCEL": // 部分約定残でのキャンセルが帰ってくる
			p.getCanceler(false, childorders[i])
		// case "CANCEL_FAILED":
		case "EXECUTION":
			// SFD徴収の検出
			if !math.IsNaN(childorders[i].SFD) && childorders[i].SFD != 0 {
				go func() { p.Result <- fmt.Sprintf("execution SFD: %f, %s", childorders[i].SFD, childorders[i].Reason) }()
			}
		case "EXPIRE":
			p.getCanceler(true, childorders[i])
		default:
			continue
		}
	}

}

func (p *Orders) getCanceler(isExpire bool, e jsonrpc.WsResponceForChildEvent) {
	if isExpire {
		sid := e.ChildOrderAcceptanceID
		p.Orders.Delete(sid)
	}

	// キャンセルされたことを保存
	p.Cancels.Store(e.ChildOrderAcceptanceID, e)
}

// IsCancel is order cancel done
func (p *Orders) IsCancel(o *Order) bool {
	_, isThere := p.Cancels.Load(o.OrderID)
	if !isThere {
		return false
	}
	// キャンセルされていればdeleteしてreturn
	p.Cancels.Delete(o.OrderID)
	return true
}

// Size 建玉枚数を正負枚数で返す
func (p *Orders) Size() (size float64) {
	p.Positions.Range(func(k, v interface{}) bool {
		pos, ok := v.(Order)
		if !ok {
			return false
		}

		if strings.HasPrefix(pos.Side, v1.BUY) {
			size += pos.Size
		} else {
			size -= pos.Size
		}

		return true
	})

	if size == 0 { // 建玉が0ならばすべての保持を捨てる
		p.Positions.Range(func(k, v interface{}) bool {
			p.Positions.Delete(k)
			return true
		})
	}

	return size
}

func (p *Client) ClosePositions() (*Order, error) {
	side := 1
	price := p.E.BestBid
	size := p.O.Size()
	if 0 < size {
		side = -1
		price = p.E.BestAsk
	}

	o, err := p.OrderBySimple(false, side, price, math.Abs(size), nil)
	if err != nil {
		return nil, err
	}
	return o, nil
}
