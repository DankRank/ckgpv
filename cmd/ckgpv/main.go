// Copyright (c) 2024 Egor
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/DankRank/ckgpv"

	"github.com/gorilla/feeds"
)

type TestOutput struct {
	Pages map[int]*ckgpv.Page `json:"pages"`
}
type FeedState struct {
	Seen map[int]struct{} `json:"seen"`
}

func main() {
	var shard string
	var doFeed, dryRun bool
	var listenAddr string
	flag.StringVar(&shard, "shard", "", "shard number")
	flag.BoolVar(&doFeed, "feed", false, "run a webserver with an atom feed")
	flag.StringVar(&listenAddr, "listen-addr", ":8091", "(feed) listen address/port")
	flag.BoolVar(&dryRun, "dry-run", false, "(feed) don't keep state")
	flag.Parse()

	if !doFeed {
		pages := ckgpv.Update(nil)
		if shard != "" {
			ckgpv.Filter2(pages, shard)
		}

		bytes, err := json.Marshal(&TestOutput{Pages: pages})
		if err != nil {
			panic(err)
		}
		os.Stdout.Write(bytes)
	} else {
		dummyTimestamp := time.Now()
		state := FeedState{Seen: make(map[int]struct{})}

		bytes, err := os.ReadFile("ckgpv-state.json")
		if err == nil {
			err = json.Unmarshal(bytes, &state)
			if err != nil {
				panic(err)
			}
		}

		http.HandleFunc("/feed.xml", func(w http.ResponseWriter, _ *http.Request) {
			pages := ckgpv.Update(state.Seen)
			if shard != "" {
				ckgpv.Filter2(pages, shard)
			}

			feed := &feeds.Feed{
				Id:      "https://cherkasyoblenergo.com/",
				Title:   "Cherkasy GPV",
				Updated: dummyTimestamp,
				Link:    &feeds.Link{Href: "https://cherkasyoblenergo.com/"},
			}
			feed.Items = make([]*feeds.Item, 0, len(pages))
			for i, page := range pages {
				id := "https://cherkasyoblenergo.com/news/" + strconv.Itoa(i)
				feed.Items = append(feed.Items,
					&feeds.Item{
						Id:      id,
						Title:   ckgpv.Summarize2(page),
						Updated: dummyTimestamp,
						Link:    &feeds.Link{Href: id, Type: "text/html"},
					})
			}
			atom, err := feed.ToAtom()
			if err != nil {
				panic(err)
			}
			io.WriteString(w, atom)

			if len(pages) > 0 && !dryRun {
				bytes, err := json.Marshal(&state)
				if err != nil {
					panic(err)
				}
				err = os.WriteFile("ckgpv-state.json", bytes, 0666)
				if err != nil {
					panic(err)
				}
			}
		})
		http.ListenAndServe(listenAddr, nil)
	}
}
