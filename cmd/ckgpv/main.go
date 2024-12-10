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

func main() {
	var shard int
	var doFeed, dryRun bool
	var listenAddr string
	flag.IntVar(&shard, "shard", 0, "shard number")
	flag.BoolVar(&doFeed, "feed", false, "run a webserver with an atom feed")
	flag.StringVar(&listenAddr, "listen-addr", ":8091", "(feed) listen address/port")
	flag.BoolVar(&dryRun, "dry-run", false, "(feed) don't keep state")
	flag.Parse()

	if !doFeed {
		gpvs := ckgpv.GPVs{Seen: make(map[int]struct{}), Pages: make(map[int]*ckgpv.Page)}
		ckgpv.Update(&gpvs)
		if shard > 0 {
			ckgpv.Filter(&gpvs, shard)
		}

		bytes, err := json.Marshal(&gpvs)
		if err != nil {
			panic(err)
		}
		os.Stdout.Write(bytes)
	} else {
		dummyTimestamp := time.Now()
		gpvs := ckgpv.GPVs{Seen: make(map[int]struct{}), Pages: make(map[int]*ckgpv.Page)}

		bytes, err := os.ReadFile("ckgpv-state.json")
		if err == nil {
			err = json.Unmarshal(bytes, &gpvs)
			if err != nil {
				panic(err)
			}
		}

		http.HandleFunc("/feed.xml", func(w http.ResponseWriter, _ *http.Request) {
			ckgpv.Update(&gpvs)
			if shard > 0 {
				ckgpv.Filter(&gpvs, shard)
			}

			feed := &feeds.Feed{
				Id:      "https://cherkasyoblenergo.com/",
				Title:   "Cherkasy GPV",
				Updated: dummyTimestamp,
				Link:    &feeds.Link{Href: "https://cherkasyoblenergo.com/"},
			}
			feed.Items = make([]*feeds.Item, 0, len(gpvs.Pages))
			for i, page := range gpvs.Pages {
				id := "https://cherkasyoblenergo.com/news/" + strconv.Itoa(i)
				feed.Items = append(feed.Items,
					&feeds.Item{
						Id:      id,
						Title:   ckgpv.Summarize(page),
						Updated: dummyTimestamp,
						Link:    &feeds.Link{Href: id, Type: "text/html"},
					})
			}
			atom, err := feed.ToAtom()
			if err != nil {
				panic(err)
			}
			io.WriteString(w, atom)

			if len(gpvs.Pages) > 0 && !dryRun {
				// For my uses, I don't need to keep these after they've been retrieved by the client
				clear(gpvs.Pages)

				bytes, err := json.Marshal(&gpvs)
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
