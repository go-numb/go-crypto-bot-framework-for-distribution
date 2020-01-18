# go-crypto-bot-framework-for-distribution
- 開発ベース フレームワーク  
- イベント駆動  
- ロジックごとの構造体を活用した追記が可能  

# What is?
- websocket.execution毎に検討（受信・送信速度優位性が必要）  
- API LimitはOrders/CancelAll:300/5m, Private(CancelByID):500/5mで管理。価格変動により100-300/5m使用。  
    - 0.1以下注文の100/1m以内チェックなし  
- SFD徴収があればプログラム終了  


# config
**bf_key**: BitflyerAPI Key  
**bf_secret**: 出金などの口座権限不要  
**testnet**: 注文したものとして価格による約定検出とデータ収集  
**size[0]**: 基本発注サイズ  
**size[1]**: オプション設定用  
**size[2]**: 最大建玉±(size[0]*10)  


# Options
- [x] discord command  
- [x] insert data into project database, use LevelDB like a SQLite([LevelDB client](https://github.com/syndtr/goleveldb)).  
 

# Recommended environment
- Linux(build可能: Mac, Windows), 1-2CPU, Mem orver 512M  
- Region:Tokyo(response time参考: Azul < Conoha < GCE)  
- Go version: 1.13.x later  


# Usage
``` 
$ cd <任意のディレクトリ>
$ git clone https://github.com/go-numb/go-crypto-bot-framework-for-distribution.git
$ cd ./go-crypto-bot-framework-for-distribution
$ mkdir logs
$ mv config_sample.toml config.toml
// APIKeyなどの書き換え
$ vim config.toml
$ go build
$ nohup ./go-crypto-bot-framework-for-distribution &
```


## Discord command
config key: discord_bot_tokenで認証を得られない場合は当オプションの稼働なし  
Discord webhookではなく、Discord developer bot token  
``` from Discord
help options: <> is veriable.
    .bf is size & API remain
    .o is summary of orders, .o<n> return the range before <n> minutes to the present 
    .sizeup is order min size + 0.01
    .sizedown is order min size - 0.01
    .fix is fix order
    .killall is kill process
```

# Author
[@_numbP](https://twitter.com/_numbp)  