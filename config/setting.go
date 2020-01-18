package config

import (
	"sync"

	"strings"
)

// Setting is Bitflyer base setting
type Setting struct {
	sync.Mutex

	IsTestNet  bool
	Leverage   int
	Code       string
	Size       []float64
	ExpireTime int

	DiffRatio float64
}

// NewSetting is useful variables
func NewSetting(isTest bool, leverage, expire int, code string, diffRatio float64, sizes []float64) *Setting {
	return &Setting{
		IsTestNet:  isTest,
		Leverage:   leverage,
		Code:       code,
		Size:       sizes,
		ExpireTime: expire,
		DiffRatio:  diffRatio,
	}
}

// SizeUp is up order size for minsize
func (p *Setting) SizeUp() {
	p.Size[0] += 0.01
}

// SizeDown is up order size for minsize
func (p *Setting) SizeDown() {
	p.Size[0] -= 0.01
	if p.Size[0] <= 0 {
		p.Size[0] = 0.01
	}
}

// DiffUp is 指値距離
func (p *Setting) DiffUp() {
	p.DiffRatio += 0.00001
}

// DiffDown is 指値距離
func (p *Setting) DiffDown() {
	p.DiffRatio -= 0.00001
}

// GetSettingByMap 規定のkeyを持つmap interfaceからSetting structを作る
func GetSettingByMap(in map[string]interface{}) *Setting {
	var s = &Setting{}
	for k, v := range in {
		switch {
		case strings.Contains(strings.ToLower(k), "testnet"):
			isTest, ok := v.(bool)
			if !ok {
				continue
			}
			s.IsTestNet = isTest

		case strings.Contains(strings.ToLower(k), "leverage"):
			leverage, ok := v.(int64)
			if !ok {
				continue
			}
			s.Leverage = int(leverage)

		case strings.Contains(strings.ToLower(k), "code"):
			code, ok := v.(string)
			if !ok {
				continue
			}
			s.Code = code

		case strings.Contains(strings.ToLower(k), "size"):
			sizes, ok := v.([]interface{})
			if !ok {
				continue
			}

			size := make([]float64, len(sizes))
			for i := range sizes {
				f, ok := sizes[i].(float64)
				if !ok {
					continue
				}
				size[i] = f
			}
			s.Size = size

		case strings.Contains(strings.ToLower(k), "expire"):
			expire, ok := v.(int64)
			if !ok {
				continue
			}
			s.ExpireTime = int(expire)

		case strings.Contains(strings.ToLower(k), "mm_basic_diff_ratio"):
			ratio, ok := v.(float64)
			if !ok {
				continue
			}
			s.DiffRatio = ratio

		default:
			continue
		}
	}

	return s
}
