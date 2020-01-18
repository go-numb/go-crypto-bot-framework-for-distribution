package config

import (
	"fmt"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestReadToml(t *testing.T) {
	var s map[string]interface{}
	toml.DecodeFile("../config.toml", &s)

	fmt.Printf("%v\n", s["discord_bot_token"])
	GetSettingByMap(s)
}
