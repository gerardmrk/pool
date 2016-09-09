## connection pooling

If you are looking at this repo for whatever reason: please use the official `sync.Pool`, not this. Made this for academic reasons and though I will use it in my own projects (so I have finegrain control over optimizing it later), I have no tests for it. So seriously please don't use it.

#### Implementation example:

```
package main

import (
	"io"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gerardmrk/pool"
)

const (
	maxGoroutines   = 25
	pooledResources = 2
)

// dbConnection simulates a resource to share.
type dbConnection struct {
	ID int32
}

// Close implements the io.Closer interface so dbConnection can be managed by the pool.
// Close performs any resource release management.
func (dbConn *dbConnection) Close() error {
	log.Println("Close: Connection", dbConn.ID)
	return nil
}

var idCounter int32

// createConnection is a factory method that will be called by the pool when a new connection is needed.
func createConnection() (io.Closer, error) {
	id := atomic.AddInt32(&idCounter, 1)
	log.Println("Create: New Connection", id)

	return &dbConnection{id}, nil
}

func main() {
	var wg sync.WaitGroup

	wg.Add(maxGoroutines)

	// Create the pool to manage our connections.
	p, err := pool.New(createConnection, pooledResources)
	if err != nil {
		log.Println(err)
	}

	// Perform queies using connections from the pool.
	for query := 0; query < maxGoroutines; query++ {
		go func(q int) {
			performQueries(q, p)
			wg.Done()
		}(query)
	}

	wg.Wait()

	log.Println("Shutting down program..")
	p.Close()
}

func performQueries(query int, p *pool.Pool) {
	conn, err := p.Acquire()
	if err != nil {
		log.Println(err)
		return
	}

	defer p.Release(conn)

	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	log.Printf("QID[%d] CID[%d]\n", query, conn.(*dbConnection).ID)
}
```
