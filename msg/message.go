package msg

// Default message struct
type Message struct {
	From string
	Data interface{}
}

// Node representation
type NodeInfoResp struct {
	NodeName        string
	BlockNum        uint64
	LastBlockHash   string
	TotalDifficulty uint64
}

// Request for needed blocks
type BlocksRequest struct {
	From         string
	To           string
	BlockNumFrom uint64
	BlockNumTo   uint64
}

type BlockMessage struct {
	Block
	TotalDifficulty uint64
}
