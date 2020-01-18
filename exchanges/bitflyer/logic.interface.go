package bitflyer

// Logicer ロジックごとの型判別で処理を分ける
type Logicer interface {
	Order(p *Client)
}
