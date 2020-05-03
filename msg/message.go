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
	TotalDifficulty uint64
}

// Request for needed blocks
type BlocksRequest struct {
	To            string
	LastBlockHash string
}

// Response with error
type BlocksResponse struct {
	To        string
	BlockHash string
	Error     error
}
