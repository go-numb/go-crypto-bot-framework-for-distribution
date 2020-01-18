package bitflyer

import (
	"math"
	"strings"
	"sync"
	"time"

	v1 "github.com/go-numb/go-bitflyer/v1"
	"github.com/go-numb/go-bitflyer/v1/public/executions"
)

// Executes is 一般約定情報
type Executes struct {
	sync.RWMutex

	// BestAsk/Bid
	IsBuy     bool
	Length    int
	Price     float64
	LastPrice float64
	BestAsk   float64
	BestBid   float64

	// 期間内高値安値
	IsRise    int
	High, Low float64

	// 出来高
	Volume float64
	// // ロスカット検出
	// Losscut *Losscut

	// n秒足乖離(加速度)
	Prices      []float64
	Volumes     []float64
	PricesPast  []float64
	VolumesPast []float64

	Delay     time.Duration
	DelayMean time.Duration
}

// NewExecutes is new Executes
func NewExecutes() *Executes {
	return &Executes{}
}

// Set is set executions to struct
func (p *Executes) Set(e []executions.Execution) {
	p.Lock()
	defer p.Unlock()
	if len(e) <= 0 {
		return
	}

	// 最終約定を取得
	p.getLTP(e)

	// Ex配信遅延を取得
	// 約定詰まりで配信延長される
	p.Delay = delay(e[0].ExecDate.Time)
}

func (p *Executes) getLTP(e []executions.Execution) {
	var wg sync.WaitGroup

	// 値幅/出来高影響力を算出するために直近価格を保存
	var lastPrice = p.Price
	if lastPrice == 0 {
		lastPrice = e[0].Price
	}

	wg.Add(1)
	go func() { // 約定量を取得
		var (
			size   float64
			length int
		)
		for i := range e {
			length++
			if strings.HasPrefix(e[i].Side, v1.BUY) {
				size += e[i].Size
			} else {
				size -= e[i].Size
			}
		}
		// ws配信による約定枚数
		p.Volume += size
		// ws配信による約定回数
		p.Length = length

		wg.Done()
	}()

	wg.Add(1)
	go func() { // 約定量を取得
		prices := make([]float64, len(e))
		sizes := make([]float64, len(e))
		for i := range e { // EMAをつくる
			prices[i] = e[i].Price
			sizes[i] = e[i].Size
		}

		p.Prices = append(p.Prices, prices...)
		p.Volumes = append(p.Volumes, sizes...)

		wg.Done()
	}()

	wg.Add(1)
	go func() { // 約定ベースのBest値をとっていく
		// 一配信前の価格を退避
		p.LastPrice = p.Price
		if strings.HasPrefix(e[0].Side, v1.BUY) {
			p.set(false, e[0].Price)
			if len(e) > 1 { // 反対売買を探す
				for i := range e {
					if e[i].Side == v1.SELL {
						p.BestBid = e[i].Price
						break
					}
				}
			}
		} else {
			p.set(true, e[0].Price)
			if len(e) > 1 { // 反対売買を探す
				for i := range e {
					if e[i].Side == v1.BUY {
						p.BestAsk = e[i].Price
						break
					}
				}
			}
		}

		wg.Done()
	}()
	wg.Wait()
}

// Reset is create mean
func (p *Executes) Reset() { // 乖離加速度Logic@taro
	p.Lock()
	defer p.Unlock()

	// 現在を過去n秒の配列に追加
	l := len(p.PricesPast)
	p.PricesPast = append(p.PricesPast, p.Prices...)    // 過去に現在を追加
	p.VolumesPast = append(p.VolumesPast, p.Volumes...) // 過去に現在を追加

	if 0 < l { // 不要部分を削除
		p.PricesPast = p.PricesPast[l:]
		p.VolumesPast = p.VolumesPast[l:]
		p.Prices = []float64{}
		p.Volumes = []float64{}
	}

	if len(p.PricesPast) != len(p.VolumesPast) {
		p.PricesPast = []float64{}
		p.VolumesPast = []float64{}
		return
	}

	p.IsRise, p.High, p.Low = highAndLow(p.Price, p.PricesPast)
}

func highAndLow(price float64, fx []float64) (isBuy int, high, low float64) {
	l := len(fx)
	if l < 2 {
		return 0, high, low
	}

	var (
		nH, nL int
	)
	low = price
	for i := range fx {
		if high < fx[i] {
			high = fx[i]
			nH = i
		} else if fx[i] < low {
			low = fx[i]
			nL = i
		}
	}

	if nH < nL {
		isBuy = -1
	} else if nL < nH {
		isBuy = 1
	}

	return isBuy, high, low
}

// ChangeInTerm 期間内変化量
func (p *Executes) ChangeInTerm() float64 {
	p.RLock()
	defer p.RUnlock()
	return p.High - p.Low
}

func (p *Executes) set(isAsk bool, price float64) {
	if !isAsk {
		p.Price = price
		p.BestAsk = price
		p.IsBuy = true
		return
	}

	p.Price = price
	p.BestBid = price
	p.IsBuy = false
}

func delay(exTime time.Time) time.Duration {
	return time.Now().Sub(exTime)
}

// Spread is culc spread
func (p *Executes) Spread() float64 {
	p.RLock()
	defer p.RUnlock()
	return math.Max(0, p.BestAsk-p.BestBid)
}

// ChangePrice is changed price 1tick ws executions
func (p *Executes) ChangePrice() float64 {
	p.RLock()
	defer p.RUnlock()
	return p.Price - p.LastPrice
}
