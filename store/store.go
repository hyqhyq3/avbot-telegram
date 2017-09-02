package store

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

var filePath string = "store.json"
var defaultStore *Store

type Store struct {
	MessageIDIndex uint64
}

func GetStore() *Store {
	if defaultStore == nil {
		p, _ := ioutil.ReadFile(filePath)
		json.Unmarshal(p, &defaultStore)
		if defaultStore == nil {
			defaultStore = &Store{}
		}
	}

	return defaultStore
}

func (s *Store) Save() {
	p, e := json.Marshal(s)
	if e != nil {
		log.Println("save error", e)
	}

	ioutil.WriteFile(filePath, p, 0755)
}
