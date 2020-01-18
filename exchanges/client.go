package exchanges

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/go-numb/go-crypto-bot-framework-for-distribution/db"
	"github.com/go-numb/go-crypto-bot-framework-for-distribution/exchanges/bitflyer"
	"github.com/sirupsen/logrus"
)

var f *os.File

type Client struct {
	DB        *db.Client
	Discord   *discordgo.Session
	Exchanges map[string]Exchanger
}

func New() *Client {
	var s map[string]interface{}
	toml.DecodeFile("./config.toml", &s)

	l := logrus.New()
	dir, err := os.Getwd()
	if err != nil {
		l.Fatal(err)
	}
	ldb := db.New(filepath.Join(dir, "leveldb"))

	// Discord connnect interactive command
	d, err := discordgo.New(fmt.Sprintf("Bot %v", s["discord_bot_token"]))
	if err != nil {
		d = nil
	}

	// ログ出力設定
	projectDir, err := os.Getwd()
	if err != nil {
		l.Fatal(err)
	}
	f, err = os.OpenFile(
		filepath.Join(projectDir, "logs", fmt.Sprintf("%s_error.log", time.Now().Format("02-01-2006"))),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0666)
	if err != nil {
		l.Fatal(err)
	}

	l.SetLevel(logrus.ErrorLevel)
	l.SetOutput(f)
	l.SetFormatter(&logrus.JSONFormatter{})

	// 各取引所をinterfaceとして扱い
	// 同様の処理はmain()で簡易switchしていく
	eachExchanges := make(map[string]Exchanger)
	eachExchanges[bitflyer.EXCHANGE] = bitflyer.New(ldb, l, d, s)

	return &Client{
		DB:        ldb,
		Discord:   d,
		Exchanges: eachExchanges,
	}
}

// Close パッケージ変数（ファイル）のクローズ
func (p *Client) Close() error {
	if err := f.Close(); err != nil {
		return err
	}
	if err := p.DB.Close(); err != nil {
		return err
	}
	if err := p.Discord.Close(); err != nil {
		return err
	}
	return nil
}
