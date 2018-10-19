package main

import (
	"github.com/hpcloud/tail"
	"github.com/wangjia184/sortedset"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type watcher struct {
	sync.RWMutex
	*sortedset.SortedSet
	lastIndex int
	errch     chan error
}

func (watcher *watcher) watch() error {
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

			watcher.Lock()
			watcher.lastIndex++
			watcher.AddOrUpdate(line.Text, sortedset.SCORE(watcher.lastIndex), struct{}{})
			watcher.Unlock()
		}
	}
}

func newWatcher() *watcher {
	return &watcher{
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
	sw, ok := w.(streamWriter)
	if !ok {
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

	go func() { watcher.errch <- http.ListenAndServe("127.0.0.1:14444", watcher) }()
	if err := watcher.watch(); err != nil {
		log.Fatal(err)
	}
}
