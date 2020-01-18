package exchanges

// Exchanger 以下exchangeを共通な処理で汎用化する
type Exchanger interface {
	Connect()
	Check()
	OrderGroup()

	// Discord command
	Interactive()
}
