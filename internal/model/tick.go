package model

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       int32
	Bid       int32
	AskVolume int32
	BidVolume int32
}
