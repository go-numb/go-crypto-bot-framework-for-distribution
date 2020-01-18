package bitflyer

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"

	v1 "github.com/go-numb/go-bitflyer/v1"
	"github.com/go-numb/go-crypto-bot-framework-for-distribution/db"
	"gonum.org/v1/gonum/stat"
)

// TestGet get and culc back test orders
func TestGet(t *testing.T) {
	var s map[string]interface{}
	toml.DecodeFile("./config.toml", &s)
	d := db.New("../../leveldb")
	l := logrus.New()
	client := New(d, l, nil, s)
	defer d.Close()

	func() {
		start := time.Now()
		defer func() {
			end := time.Now()
			fmt.Println("exec time: ", end.Sub(start))
		}()

		orders, err := client.GetOrders("orders_test", time.Now().Add(-10*time.Hour), time.Now())
		if err != nil {
			t.Fatal(err)
		}

		var (
			pricesB        []float64
			volumesB       []float64
			pricesS        []float64
			volumesS       []float64
			buys, sells    float64
			sum, sumIsDone float64
			maxSize        float64
		)
		for i := range orders {
			sum += orders[i].Size
			if !orders[i].IsDone {
				continue
			}
			sumIsDone += orders[i].Size

			if orders[i].Side == v1.BUY {
				pricesB = append(pricesB, orders[i].Price)
				volumesB = append(volumesB, orders[i].Size)
				buys += orders[i].Size
			} else if orders[i].Side == v1.SELL {
				pricesS = append(pricesS, orders[i].Price)
				volumesS = append(volumesS, orders[i].Size)
				sells += orders[i].Size
			}

			diff := math.Abs(buys - sells)
			if maxSize < diff {
				maxSize = diff
			}
		}

		fmt.Printf("order length: %d,	exec rate: %.3f％\n", len(orders), (sumIsDone/sum)*100.0)
		fmt.Printf("保有最大: %.4f\n", maxSize)
		fmt.Printf("BUY	%.1f	%.4f\n", stat.Mean(pricesB, volumesB), buys)
		fmt.Printf("SELL	%.1f	%.4f\n", stat.Mean(pricesS, volumesS), sells)
	}()
}

// TestClear clear all table in levelDB
func TestClear(t *testing.T) {
	client := db.New("../../leveldb")
	defer client.Close()

	client.Clear("orders_test")
}
