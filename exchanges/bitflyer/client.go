package bitflyer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/go-numb/go-bitflyer/auth"
	v1 "github.com/go-numb/go-bitflyer/v1"
	"github.com/go-numb/go-bitflyer/v1/jsonrpc"
	"github.com/go-numb/go-crypto-bot-framework-for-distribution/config"
	"github.com/go-numb/go-crypto-bot-framework-for-distribution/db"
)

const (
	EXCHANGE        = "bitflyer"
	RECONNECTMINUTE = 12
	DELAYTHRESHHOLD = 2000 * time.Millisecond

	// ORDER
	WAITEXECUTE = 1800 // second

	// LOGIC workers
	WORKERS        = 10
	WORKERMMPROFIT = 1
	WORKERMMBASIC  = 1

	// CHECKRESOUCEPERIOD websocket executionsの集計（期間内最大変動などを取得）
	CHECKRESOUCEPERIOD = 3
	// MMBASICCANCELORDER is basic market make orderの板乗り時間（±Delay, ±PingTime）でCHECKRESOUCEPERIOD*MMBASICCANCELORDER秒を基本とする
	MMBASICCANCELORDER = 3
)

var (
	TANHTENSION = 0.01
)

type Client struct {
	DB      *db.Client
	Discord *discordgo.Session

	// Event駆動用
	Event chan interface{}

	C           *v1.Client
	Controllers *Controllers
	Setting     *config.Setting

	// API Limit
	Public  *v1.Limit
	Private *v1.Limit

	// Executions
	E *Executes
	O *Orders

	// Logic 可否
	Logics *Logics
	// Logic 発注開始
	Logic chan Logicer

	Logger *logrus.Entry
}

type Logics struct {
	MMBasic   bool
	MMSpecial bool
	VPIN      bool
}

func New(ldb *db.Client, l *logrus.Logger, d *discordgo.Session, s map[string]interface{}) *Client {
	c := v1.NewClient(&v1.ClientOpts{
		AuthConfig: &auth.AuthConfig{
			APIKey:    fmt.Sprintf("%v", s["bf_key"]),
			APISecret: fmt.Sprintf("%v", s["bf_secret"]),
		},
	})
	if c == nil {
		return nil
	}

	log := l.WithFields(logrus.Fields{
		"exchange": EXCHANGE,
	})

	mmBasic, ok := s["mm_basic"].(bool)
	if !ok {
		mmBasic = false
	}
	mmSpecial, ok := s["mm_special"].(bool)
	if !ok {
		mmSpecial = false
	}
	vpin, ok := s["vpin"].(bool)
	if !ok {
		vpin = false
	}
	swing, ok := s["swing"].(bool)
	if !ok {
		swing = false
	}

	f, err := strconv.ParseFloat(fmt.Sprintf("%v", s["tension"]), 64)
	if err == nil {
		TANHTENSION = f
	}

	return &Client{
		DB:      ldb,
		Discord: d,

		Event: make(chan interface{}),

		C: c,
		Controllers: &Controllers{
			Profit: &Controller{
				IsDo:  false,
				Count: 0,
				Limit: WORKERMMPROFIT,
			},
			Basic: &Controller{
				IsDo:  mmBasic,
				Count: 0,
				Limit: WORKERMMBASIC,
			},
			Special: &Controller{
				IsDo:  mmSpecial,
				Count: 0,
				Limit: 0,
			},
			VPIN: &Controller{
				IsDo:  vpin,
				Count: 0,
				Limit: 1,
			},
			Swing: &Controller{
				IsDo:  swing,
				Count: 0,
				Limit: 1,
			},
		},
		Setting: config.GetSettingByMap(s),
		Public:  v1.NewLimit(),
		Private: v1.NewLimit(),

		E: NewExecutes(),
		O: NewOrders(),

		Logic: make(chan Logicer, WORKERS),

		Logger: log,
	}
}

func recived(event chan interface{}, in interface{}) {
	event <- in
}

func (p *Client) Connect() {
	// connect websocket for private
	go p.ConnectForPrivate()

Reconnect:
	ch := make(chan jsonrpc.Response)
	channels := []string{
		"lightning_executions_FX_BTC_JPY",
	}
	go jsonrpc.Get(channels, ch)
	time.Sleep(time.Second)
	fmt.Printf("read channel: %v\n", channels)

	for {
		select {
		case v := <-ch:
			switch v.Type {
			case jsonrpc.Executions:
				go recived(p.Event, true)
				// LTP取得
				go p.E.Set(v.Executions)
				// 建玉自炊用 約定確認
				go p.O.Check(v.Executions)

			case jsonrpc.Error:
				p.Logger.Error("websocket reconnect error, ", v.Error.Error())
				goto wsError
			}
		}
	}

wsError:
	close(ch)
	if isMentenance() {
		p.Logger.Infof(
			"now time %s, therefore waiting websocket reconnect %dminutes later",
			time.Now().Format("15:04"),
			RECONNECTMINUTE)
		time.Sleep(time.Duration(RECONNECTMINUTE) * time.Minute)
		p.Logger.Infof("end bitflyer mentenance time, reconnect websocket")
	} else {
		time.Sleep(3 * time.Second)
	}
	goto Reconnect
}

// ConnectForPrivate check order status
func (p *Client) ConnectForPrivate() {
Reconnect:
	var (
		channels = []string{
			"lightning_ticker_BTC_JPY",
			"child_order_events",
			// "parent_order_events",
		}
		ch = make(chan jsonrpc.Response)
	)
	go jsonrpc.GetPrivate(p.C.AuthConfig.APIKey, p.C.AuthConfig.APISecret, channels, ch)

	for {
		select {
		case v := <-ch:
			switch v.Type {
			case jsonrpc.ChildOrders:
				// fmt.Printf("child: %.4f, %+v\n", time.Now().Sub(v.ChildOrders[0].EventDate).Seconds(), v.ChildOrders)
				p.O.CheckByPrivateWs(v.ChildOrders)

			// case jsonrpc.ParentOrders:
			// 	fmt.Printf("parent: %.4f, %+v\n", time.Now().Sub(v.ChildOrders[0].EventDate).Seconds(), v.ParentOrders)
			default:
				p.Logger.Error("websocket reconnect error, ", v.Error.Error())
				goto wsError
			}

		}
	}

wsError:
	close(ch)
	if isMentenance() {
		p.Logger.Infof(
			"now time %s, therefore waiting websocket reconnect %dminutes later",
			time.Now().Format("15:04"),
			RECONNECTMINUTE)
		time.Sleep(time.Duration(RECONNECTMINUTE) * time.Minute)
		p.Logger.Infof("end bitflyer mentenance time, reconnect websocket")
	} else {
		time.Sleep(3 * time.Second)
	}
	goto Reconnect
}

func isMentenance() bool {
	// ServerTimeを考慮し、基本に合わせる
	hour := time.Now().UTC().Hour()
	if hour != 19 {
		return false
	}
	// 午前四時台ならば、分チェックする
	if RECONNECTMINUTE < time.Now().Minute() { // メンテナンス以外
		return false
	}
	return true
}

// Check gets resorse and culc
func (p *Client) Check() {
Reconnect:

	ctx, cancel := context.WithCancel(context.Background())

	var eg errgroup.Group
	eg.Go(func() error {
		ticker := time.NewTicker(time.Duration(CHECKRESOUCEPERIOD) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-p.Event:
				go p.LogicChecker()

			case <-ticker.C:
				p.E.Reset()

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	eg.Go(func() error {
		ticker1m := time.NewTicker(time.Minute)
		defer ticker1m.Stop()

		for {
			select {
			case t := <-ticker1m.C:
				go p.CulcForward()
				if t.Minute()%AGGPERIOD == 0 {
					data, err := p.GetOrdersInfo(DBTABLEORDERSINFO, time.Now().Add(-time.Duration(AGGPERIOD)*time.Minute), time.Now())
					if err != nil {
						cancel()
						return err
					}
					if len(data) < 1 {
						continue
					}
					fmt.Println(data[0].Columns())
					for i := range data {
						fmt.Println(data[i].String())
					}
				}

			case result := <-p.O.Result:
				return errors.New(result)

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	if err := eg.Wait(); err != nil {
		if strings.Contains(err.Error(), "SFD") {
			p.Logger.Fatal("killall by SFD")
		}
		p.Logger.Error(err)
	}

	goto Reconnect
}

// OrderGroup LogicIgnition受信用
func (p *Client) OrderGroup() {
Reconnect:

	for {
		select {
		case logic := <-p.Logic:
			go logic.Order(p)

		}
	}

	p.Logger.Error("undefined error in LogicLoop")

	goto Reconnect
}

// SetError is log & time.Sleep
func (p *Client) SetError(isWait bool, err error) {
	p.Logger.Error(err)
	switch {
	case strings.Contains(err.Error(), "509"):
		if !isWait {
			return
		}
		time.Sleep(5 * time.Second)

	case strings.Contains(err.Error(), "429 Too Many Requests"):
		if !isWait {
			return
		}
		time.Sleep(time.Duration(p.Private.Period) * time.Second)

	case strings.Contains(err.Error(), "400 Bad Request"):
		if !isWait {
			return
		}
		time.Sleep(1 * time.Second)
	}
}
