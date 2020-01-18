package bitflyer

import (
	porders "github.com/go-numb/go-bitflyer/v1/private/cancels/positions"
	"github.com/go-numb/go-bitflyer/v1/types"
	"github.com/pkg/errors"
)

// CancelByID is cancel by orderID
func (p *Client) CancelByID(id string) (int, error) {
	if err := p.Private.Check(); err != nil { // CancelByIDで減るのかは分からない 2019/12/02
		return 0, err
	}

	_, res, err := p.C.CancelByID(&porders.Request{
		ProductCode:            types.ProductCode(p.Setting.Code),
		ChildOrderAcceptanceId: id,
	})
	if err != nil {
		return 400, err
	}
	if res == nil {
		return 400, errors.New("cancel responce has not data")
	}
	defer res.Body.Close()

	// 返り値からAPILIMITを取得
	// Chancelの場合、現行(2020/01/01現在)Headerに有効値がない
	p.Private.FromHeader(res.Header)

	return 200, nil
}
