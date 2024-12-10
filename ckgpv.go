// Copyright (c) 2024 Egor
// SPDX-License-Identifier: GPL-2.0-or-later

package ckgpv

import (
	"slices"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type Page struct {
	Title string      `json:"title"`
	Rows  [][2]string `json:"rows"`
}
type GPVs struct {
	Seen  map[int]struct{} `json:"seen"`
	Pages map[int]*Page    `json:"pages"`
}

func Update(gpvs *GPVs) {
	newsCollector := colly.NewCollector()
	newsCollector.OnHTML(":root", func(e *colly.HTMLElement) {
		id, err := strconv.Atoi(strings.TrimPrefix(e.Request.URL.Path, "/news/"))
		if err != nil {
			panic(err)
		}
		tds := e.ChildTexts("td")
		rows := make([][2]string, len(tds)/2)
		for i := range len(tds) / 2 {
			rows[i][0] = tds[2*i]
			rows[i][1] = tds[2*i+1]
		}
		gpvs.Pages[id] = &Page{
			Title: e.ChildText("title"),
			Rows:  rows,
		}
	})

	homepageCollector := colly.NewCollector()
	homepageCollector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if strings.HasPrefix(href, "/news/") {
			id, err := strconv.Atoi(strings.TrimPrefix(href, "/news/"))
			if err != nil {
				panic(err)
			}
			_, ok := gpvs.Seen[id]
			gpvs.Seen[id] = struct{}{}
			if !ok && strings.Contains(e.Text, "погодинних відключень") {
				newsCollector.Visit(e.Request.AbsoluteURL(href))
			}
		}
	})
	homepageCollector.Visit("https://cherkasyoblenergo.com/")
}
func Filter(gpvs *GPVs, shard int) {
	shardCh := strconv.Itoa(shard) // assume shard < 10
	for _, page := range gpvs.Pages {
		page.Rows = slices.DeleteFunc(page.Rows, func(row [2]string) bool {
			return !strings.Contains(row[1], shardCh)
		})
	}
}
func Summarize(page *Page) string {
	fromto := make([][2]string, 0, len(page.Rows))
	for _, row := range page.Rows {
		s := strings.Split(row[0], "-")
		if len(fromto) > 0 && fromto[len(fromto)-1][1] == s[0] {
			fromto[len(fromto)-1][1] = s[1]
		} else {
			fromto = append(fromto, [2]string(s))
		}
	}
	summary := page.Title
	for _, s := range fromto {
		summary += "\n" + s[0] + "-" + s[1]
	}
	return summary
}
