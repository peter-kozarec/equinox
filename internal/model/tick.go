package model

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       float64
	Bid       float64
	AskVolume float64
	BidVolume float64
}
