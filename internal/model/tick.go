package model

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       Price
	Bid       Price
	AskVolume Price
	BidVolume Price
}
