package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// These are our in-memory database collections.
var channels []*Chnl
var items []*Itm
var fetchErrors []*fetchError

var protector sync.Mutex

type fetchError struct {
	ChannelUrl string
	Error      string
}

func insertChannel(c *Chnl) {
	defer protector.Unlock()
	protector.Lock()
	channels = append(channels, c)
}

func insertItem(i *Itm) {
	defer protector.Unlock()
	protector.Lock()
	items = append(items, i)
}

func insertFetchError(e *fetchError) {
	defer protector.Unlock()
	protector.Lock()
	fetchErrors = append(fetchErrors, e)
}

type bookmark struct {
	ChannelUrl string
	Time       time.Time
}

func getBookmarksPath() string {
	return os.ExpandEnv(filepath.Join(viper.GetString("data_dir"), "bookmarks.json"))
}

func getBookmarks() ([]*bookmark, error) {
	path := getBookmarksPath()
	file, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var j []*bookmark
	err = json.Unmarshal(file, &j)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func saveBookmarks(bk []*bookmark) error {
	path := getBookmarksPath()
	b, err := json.MarshalIndent(bk, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, b, 0644)
	return err
}

func pruneSentItems(bookmarks []*bookmark) {
	fmt.Printf("Starting prune with %v items\n", len(items))
	for i := len(items) - 1; i >= 0; i-- {
		it := items[i]

		c := findChannelByKey(it.ChannelKey)
		if c == nil {
			continue
		}
		b := findBookmarkByChannel(bookmarks, c)
		if b == nil {
			continue
		}

		if it.Date.Before(b.Time) || it.Date.Equal(b.Time) {
			items = append(items[:i], items[i+1:]...)
		}
	}
	fmt.Printf("Ending prune with %v items\n", len(items))
}

func updateBookmarks(bookmarks []*bookmark) []*bookmark {
	for _, c := range channels {
		var newest time.Time
		for _, i := range getItemsByChannel(c) {
			if i.Date.After(newest) {
				newest = i.Date
			}
		}

		b := findBookmarkByChannel(bookmarks, c)
		if b != nil {
			b.Time = newest
		} else {
			bookmarks = append(bookmarks, &bookmark{
				ChannelUrl: c.Url,
				Time:       newest,
			})
		}
	}
	return bookmarks
}

func findChannelByKey(key string) *Chnl {
	for _, c := range channels {
		if c.Key == key {
			return c
		}
	}
	return nil
}

func findBookmarkByChannel(bookmarks []*bookmark, c *Chnl) *bookmark {
	for _, b := range bookmarks {
		if b.ChannelUrl == c.Url {
			return b
		}
	}

	return nil
}

func pruneEmptyChannels() {
	for i := len(channels) - 1; i >= 0; i-- {
		c := channels[i]

		if len(getItemsByChannel(c)) == 0 {
			// Remove the channel
			channels = append(channels[:i], channels[i+1:]...)
		}
	}
}

func getItemsByChannel(c *Chnl) []*Itm {
	var r []*Itm
	for _, i := range items {
		if i.ChannelKey == c.Key {
			r = append(r, i)
		}
	}
	return r
}
