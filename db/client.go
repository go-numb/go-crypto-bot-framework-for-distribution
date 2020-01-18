package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/syndtr/goleveldb/leveldb"
)

type Client struct {
	LevelDB *leveldb.DB
}

func New(filename string) *Client {
	ldb, err := leveldb.OpenFile(filename, nil)
	if err != nil {
		fmt.Println(err)
		return &Client{}
	}

	return &Client{
		LevelDB: ldb,
	}
}

// IsLevelDB check for client
func (p *Client) IsLevelDB() error {
	if p.LevelDB == nil {
		return errors.New("not has LevelDB")
	}
	return nil
}

func (p *Client) Close() error {
	if err := p.LevelDB.Close(); err != nil {
		return err
	}
	return nil
}

// Set prefix:unix into data
func (p *Client) Set(key string, in interface{}) error {
	if err := p.IsLevelDB(); err != nil {
		return err
	}

	b, err := json.Marshal(in)
	if err != nil {
		return err
	}

	if err := p.LevelDB.Put([]byte(fmt.Sprintf("%s:%d", key, time.Now().UnixNano())), b, nil); err != nil {
		return err
	}

	return nil
}

// Clear is table name:prefix all clear
func (p *Client) Clear(prefix string) error {
	if err := p.IsLevelDB(); err != nil {
		return err
	}

	var r *util.Range
	if prefix != "" {
		r = util.BytesPrefix([]byte(prefix))
	}
	rows := p.LevelDB.NewIterator(r, nil)

	for rows.Next() {
		key := rows.Key()
		if err := p.LevelDB.Delete(key, nil); err != nil {
			continue
		}
	}
	rows.Release()
	err := rows.Error()
	if err != nil {
		return err
	}

	return nil
}
