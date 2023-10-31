package reader

import (
	"sync"

	"git.mills.io/prologic/bitcask"
)

var lock = &sync.Mutex{}
var db *bitcask.Bitcask

// `GetInstance`models the vault(KV store) as a
// Singleton.
// The vault stores IP/truth-value mappings for liarslie
func GetInstance() *bitcask.Bitcask {
	if db == nil {
		lock.Lock()
		defer lock.Unlock()
		if db == nil {
			db, _ = bitcask.Open("storage/")
			defer db.Close()
		}
	}
	return db
}
