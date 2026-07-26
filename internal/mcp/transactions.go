package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/rjboer/omron-mcp/internal/sysmac"
)

var transactionStore = struct {
	sync.Mutex
	items map[string]*sysmac.ProjectTransaction
}{items: map[string]*sysmac.ProjectTransaction{}}

func newTransactionID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return hex.EncodeToString(data)
}

func storeTransaction(tx *sysmac.ProjectTransaction) string {
	id := newTransactionID()
	transactionStore.Lock()
	transactionStore.items[id] = tx
	transactionStore.Unlock()
	return id
}

func getTransaction(id string) (*sysmac.ProjectTransaction, error) {
	transactionStore.Lock()
	tx := transactionStore.items[id]
	transactionStore.Unlock()
	if tx == nil {
		return nil, fmt.Errorf("transaction %q not found", id)
	}
	return tx, nil
}

func removeTransaction(id string) {
	transactionStore.Lock()
	delete(transactionStore.items, id)
	transactionStore.Unlock()
}
