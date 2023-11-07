/*
    history-watcher
    Copyright (C) 2018 Minori Tokuda

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package main

import (
	"github.com/hpcloud/tail"
	"github.com/kelseyhightower/envconfig"
	"github.com/wangjia184/sortedset"
	"go.etcd.io/bbolt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	Host   string `default:"127.0.0.1"`
	Port   string `default:"14444"`
	Token  string
	DbFile string
	Poll   bool
}

type watcher struct {
	db *bbolt.DB
	sync.RWMutex
	*sortedset.SortedSet
	lastIndex int
	errch     chan error
	Config
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
	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = watcher.load()
	if err != nil {
		return err
	}

	t, err := tail.TailFile(
		filepath.Join(homedir, ".bash_history"),
		tail.Config{
			ReOpen: true,
			Follow: true,
			Poll:   watcher.Poll,
		},
	)
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

			if strings.HasPrefix(line.Text, "#") || line.Text == "" {
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

func newWatcher(conf Config) *watcher {
	var (
		db *bbolt.DB
	)
	if conf.DbFile != "" {
		var err error
		db, err = bbolt.Open(conf.DbFile, 0600, nil)
		if err != nil {
			log.Println(err)
			db = nil
		}
	}

	return &watcher{
		Config:    conf,
		db:        db,
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
	if watcher.Token != "" {
		authHeader := r.Header.Get("authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if watcher.Token != token {
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

func run() error {
	var conf Config
	err := envconfig.Process("hw", &conf)
	if err != nil {
		return err
	}
	log.Printf(
		"bind_address=%s:%s, db_file=%s, token_authentication_enabled=%t, polling=%t",
		conf.Host, conf.Port,
		conf.DbFile,
		conf.Token != "",
		conf.Poll,
	)
	watcher := newWatcher(conf)
	go func() { watcher.errch <- http.ListenAndServe(net.JoinHostPort(conf.Host, conf.Port), watcher) }()
	if err := watcher.watch(); err != nil {
		return err
	}
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
