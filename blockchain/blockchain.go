package blockchain

type Block struct {
	Index		int
	Timestamp	string
	Treats		int
	Hash		string
	PrevHash	string
	Difficult 	int
	Nonce		string
}

var Blockchain []Block
