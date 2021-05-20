package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"strconv"
	"sync"
	"strings"
	"fmt"
	"math/rand"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

//initial difficulty
const difficulty = 1

type Block struct {
	Index		int
	Timestamp	string
	Treats		int
	Hash		string
	PrevHash	string
	Difficulty 	int
	Nonce		string
}

var Blockchain []Block

type Message struct {
	Treats int
}

var mutex = &sync.Mutex{}

func calculateHash(block Block) string{
	record:=strconv.Itoa(block.Index) + block.Timestamp + strconv.Itoa(block.Treats)+block.PrevHash+block.Nonce
	h:= sha256.New()
	h.Write([]byte(record))
	hashed:=h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func isHashValid(hash string, difficulty int) bool {
        prefix := strings.Repeat("0", difficulty)
        return strings.HasPrefix(hash, prefix)
}

func generateBlock(oldBlock Block, Treats int) (Block, error){
	var newBlock Block

	t:=time.Now()

	//calc difficulty increase, if any
	var increase int = 0
	if rand.Intn(100) > 90{
		increase =1
	}

	newBlock.Index=oldBlock.Index+1
	newBlock.Timestamp=t.String()
	newBlock.Treats=Treats
	newBlock.PrevHash=oldBlock.Hash
	newBlock.Hash=calculateHash(newBlock)
	newBlock.Difficulty = oldBlock.Difficulty+increase

	for i := 0; ; i++ {
			hex := fmt.Sprintf("%x", i)
			newBlock.Nonce = hex
			if !isHashValid(calculateHash(newBlock), newBlock.Difficulty) {
					fmt.Println(calculateHash(newBlock), " do more work!")
					//time.Sleep(time.Second) //simulate time
					continue
			} else {
					fmt.Println(calculateHash(newBlock), " work done!")
					newBlock.Hash = calculateHash(newBlock)
					break
			}

	}

	return newBlock, nil
}

func isBlockValid(newBlock,oldBlock Block) bool{
	if oldBlock.Index+1!=newBlock.Index{
		return false
	}
	
	if oldBlock.Hash != newBlock.PrevHash{
		return false
	}

	if calculateHash(newBlock)!= newBlock.Hash{
		return false
	}

	return true
}

func run() error{
	mux:=makeMuxRouter()
	httpAddr:= os.Getenv("PORT")
	log.Println("Listening on ", os.Getenv("PORT"))
	s:=&http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	//ensure atomicity when creating new block
    mutex.Lock()
	newBlock,err := generateBlock(Blockchain[len(Blockchain)-1], m.Treats)
	mutex.Unlock()
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}

	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
    w.Header().Set("Content-Type", "application/json")
	response, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
			t := time.Now()
			genesisBlock := Block{}
			genesisBlock = Block{0, t.String(), 0, calculateHash(genesisBlock), "", difficulty, ""} 
			spew.Dump(genesisBlock)

			mutex.Lock()
			Blockchain = append(Blockchain, genesisBlock)
			mutex.Unlock()
	}()
	log.Fatal(run())

}