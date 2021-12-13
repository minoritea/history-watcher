package main

import (
	"github.com/hpcloud/tail"
	"github.com/wangjia184/sortedset"
	"go.etcd.io/bbolt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type watcher struct {
	token *string
	db    *bbolt.DB
	sync.RWMutex
	*sortedset.SortedSet
	lastIndex int
	errch     chan error
}

var bucketName = []byte("HISTORY_WATCHER")

func (watcher *watcher) load() error {
	watcher.Lock()
	watcher.Unlock()
	if watcher.db != nil {
		err := watcher.db.Update(func(tx *bbolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists(bucketName)
			return err
		})
		if err != nil {
			return err
		}
		err = watcher.db.View(func(tx *bbolt.Tx) error {
			c := tx.Bucket(bucketName).Cursor()
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				watcher.lastIndex++
				watcher.AddOrUpdate(string(k), sortedset.SCORE(watcher.lastIndex), struct{}{})
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (watcher *watcher) watch() error {
	err := watcher.load()
	if err != nil {
		return err
	}

	t, err := tail.TailFile(filepath.Join(os.Getenv("HOME"), ".bash_history"), tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}

	for {
		select {
		case <-t.Dead():
			return nil

		case err := <-watcher.errch:
			return err

		case line := <-t.Lines:
			if line == nil {
				return nil
			}

			if err := line.Err; err != nil {
				return err
			}

			if strings.HasPrefix(line.Text, "#") {
				continue
			}

			if watcher.db != nil {
				err := watcher.db.Update(func(tx *bbolt.Tx) error {
					return tx.Bucket(bucketName).Put([]byte(line.Text), []byte{0x0})
				})
				if err != nil {
					return err
				}
			}

			watcher.Lock()
			watcher.lastIndex++
			watcher.AddOrUpdate(line.Text, sortedset.SCORE(watcher.lastIndex), struct{}{})
			watcher.Unlock()
		}
	}
}

func newWatcher() *watcher {
	var (
		db    *bbolt.DB
		token *string
	)
	dbpath := os.Getenv("HW_DBFILE")
	if dbpath != "" {
		var err error
		db, err = bbolt.Open(dbpath, 0600, nil)
		if err != nil {
			log.Println(err)
			db = nil
		}
	}

	tokenStr := os.Getenv("HW_TOKEN")
	if tokenStr != "" {
		token = &tokenStr
	}
	return &watcher{
		db:        db,
		token:     token,
		SortedSet: sortedset.New(),
		errch:     make(chan error),
	}
}

type streamWriter interface {
	http.CloseNotifier
	http.Flusher
	Write([]byte) (int, error)
}

func (watcher *watcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if watcher.token != nil {
		authHeader := r.Header.Get("authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if *watcher.token != token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	sw, ok := w.(streamWriter)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	watcher.RLock()
	li := watcher.lastIndex
	watcher.RUnlock()

	for i := li; i >= 0; i-- {
		select {
		case <-sw.CloseNotify():
			return
		default:
			watcher.RLock()
			node := watcher.GetByRank(i, false)
			if node != nil {
				sw.Write([]byte(node.Key() + "\n"))
			}
			watcher.RUnlock()
			sw.Flush()
		}
	}
}

func main() {
	watcher := newWatcher()

	ip := os.Getenv("HW_IP")
	if ip == "" {
		ip = "127.0.0.1"
	}

	port := os.Getenv("HW_PORT")
	if port == "" {
		port = "14444"
	}

	go func() { watcher.errch <- http.ListenAndServe(ip+":"+port, watcher) }()
	if err := watcher.watch(); err != nil {
		log.Fatal(err)
	}
}
