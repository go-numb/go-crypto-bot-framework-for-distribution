package main

import (
	"fmt"

	"github.com/go-numb/go-crypto-bot-framework-for-distribution/exchanges"
)

func init() {

}

func main() {
	done := make(chan bool)
	client := exchanges.New()
	defer client.Close()

	// 設定した取引所をKeyに指定して共通処理を行う
	for exchange := range client.Exchanges {
		fmt.Println("start: " + exchange)
		// connect websocket
		go client.Exchanges[exchange].Connect()
		// check & culc recive data
		go client.Exchanges[exchange].Check()
		// process orders comming from check & culc
		go client.Exchanges[exchange].OrderGroup()
		// ping&pong interactive command
		go client.Exchanges[exchange].Interactive()
	}

	<-done
}
