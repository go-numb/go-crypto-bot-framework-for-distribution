package bitflyer

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	v1 "github.com/go-numb/go-bitflyer/v1"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gonum.org/v1/gonum/stat"
)

const (
	// AGGPERIOD Aggregation period
	AGGPERIOD = 15
	// DBTABLEORDERS 注文（約定: IsDone）
	DBTABLEORDERS = "orders"
	// DBTABLEORDERSFORTEST 注文（約定: IsDone）
	DBTABLEORDERSFORTEST = "orders_test"
	// DBTABLEORDERSINFO 約定集計（約定率やbid/askの傾き、最大保持枚数）
	DBTABLEORDERSINFO = "orders_info"
)

// GetOrders get orders before start to end time
func (p *Client) GetOrders(prefix string, start, end time.Time) ([]Order, error) {
	if err := p.DB.IsLevelDB(); err != nil {
		return nil, err
	}

	rows := p.DB.LevelDB.NewIterator(&util.Range{
		Start: []byte(fmt.Sprintf("%s:%d", prefix, start.UnixNano())),
		Limit: []byte(fmt.Sprintf("%s:%d", prefix, end.UnixNano())),
	}, nil)

	var data = make([]Order, 0)
	for rows.Next() {
		value := rows.Value()
		var o Order
		if err := json.Unmarshal(value, &o); err != nil {
			return nil, err
		}
		data = append(data, o)
	}
	rows.Release()
	err := rows.Error()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// CulcForward culc orders in this running
func (p *Client) CulcForward() {
	start := time.Now()

	orders, err := p.GetOrders(DBTABLEORDERS, time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		p.SetError(false, err)
		return
	}

	var (
		wg                      sync.WaitGroup
		pricesB                 []float64
		volumesB                []float64
		pricesS                 []float64
		volumesS                []float64
		buys, sells             float64
		buysIsDone, sellsIsDone float64
		sum, sumIsDone          float64
		maxSize                 float64
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range orders {
			sum += orders[i].Size
			if orders[i].Side == v1.BUY {
				buys += orders[i].Size
			} else if orders[i].Side == v1.SELL {
				sells += orders[i].Size
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range orders {
			if !orders[i].IsDone {
				continue
			}
			sumIsDone += orders[i].Size

			if orders[i].Side == v1.BUY {
				pricesB = append(pricesB, orders[i].Price)
				volumesB = append(volumesB, orders[i].Size)
				buysIsDone += orders[i].Size
			} else if orders[i].Side == v1.SELL {
				pricesS = append(pricesS, orders[i].Price)
				volumesS = append(volumesS, orders[i].Size)
				sellsIsDone += orders[i].Size
			}

			diff := math.Abs(buysIsDone - sellsIsDone)
			if maxSize < diff {
				maxSize = diff
			}
		}
	}()

	wg.Wait()

	end := time.Now()

	if err := p.DB.Set(DBTABLEORDERSINFO, &OrdersInfo{
		Length:       len(orders),
		Rate:         sumIsDone / sum,
		RateBid:      buysIsDone / sum,
		RateAsk:      sellsIsDone / sum,
		Max:          maxSize,
		AvgBid:       stat.Mean(pricesB, volumesB),
		AvgAsk:       stat.Mean(pricesS, volumesS),
		VolumeBid:    buysIsDone,
		VolumeAsk:    sellsIsDone,
		OrdersizeBid: buys,
		OrdersizeAsk: sells,
		ExecTime:     end.Sub(start),
		CreatedAt:    time.Now(),
	}); err != nil {
		p.SetError(false, err)
		return
	}
}

type OrdersInfo struct {
	Length                     int
	Rate                       float64
	RateBid, RateAsk           float64
	Max                        float64
	AvgBid, AvgAsk             float64
	VolumeBid, VolumeAsk       float64
	OrdersizeBid, OrdersizeAsk float64

	ExecTime  time.Duration
	CreatedAt time.Time
}

// Columns return string, header columns
func (p *OrdersInfo) Columns() string {
	return fmt.Sprint("length,	rate,	rateAsk,	rateBid,	return,	vol_ask,	vol_bid,	order_ask,	order_bid,	max,	exec_time,	created_at")
}

// String is print for human
func (p *OrdersInfo) String() string {
	return fmt.Sprintf(
		"%d,	%.2f,	%.2f,	%.2f,	%.1f,	%.4f,	%.4f,	%.4f,	%.4f,	%.4f,	%v,	%s",
		p.Length,
		p.Rate*100,
		p.RateAsk*100,
		p.RateBid*100,
		p.AvgAsk-p.AvgBid,
		p.VolumeAsk,
		p.VolumeBid,
		p.OrdersizeAsk,
		p.OrdersizeBid,
		p.Max,
		p.ExecTime.Seconds(),
		p.CreatedAt.Format("2006/01/02 15:04:05"))
}

// GetOrdersInfo aggrigate orders
func (p *Client) GetOrdersInfo(prefix string, start, end time.Time) ([]OrdersInfo, error) {
	if err := p.DB.IsLevelDB(); err != nil {
		return nil, err
	}

	rows := p.DB.LevelDB.NewIterator(&util.Range{
		Start: []byte(fmt.Sprintf("%s:%d", prefix, start.UnixNano())),
		Limit: []byte(fmt.Sprintf("%s:%d", prefix, end.UnixNano())),
	}, nil)

	var data = make([]OrdersInfo, 0)
	for rows.Next() {
		value := rows.Value()
		var o OrdersInfo
		if err := json.Unmarshal(value, &o); err != nil {
			return nil, err
		}
		data = append(data, o)
	}
	rows.Release()
	err := rows.Error()
	if err != nil {
		return nil, err
	}

	return data, nil
}
