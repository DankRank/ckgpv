// Copyright (c) 2024, 2026 Egor
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

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func splitGPVLine(line string) (row [2]string, ok bool) {
	row[0], row[1], ok = strings.Cut(line, " ")
	if ok {
		row[0] = strings.TrimRight(row[0], ":")
		ok = len(row[0]) == 3 && isDigit(row[0][0]) && row[0][1] == '.' && isDigit(row[0][2])
	}
	return
}

func Update(seen map[int]struct{}) map[int]*Page {
	pages := make(map[int]*Page)
	newsCollector := colly.NewCollector()
	newsCollector.OnHTML(":root", func(e *colly.HTMLElement) {
		id, err := strconv.Atoi(strings.TrimPrefix(e.Request.URL.Path, "/news/"))
		if err != nil {
			panic(err)
		}
		// tables (old)
		tds := e.ChildTexts("td")
		rows := make([][2]string, len(tds)/2)
		for i := range len(tds) / 2 {
			rows[i][0] = tds[2*i]
			rows[i][1] = tds[2*i+1]
		}
		// paragraphs
		for _, p := range e.ChildTexts("p") {
			if row, ok := splitGPVLine(p); ok {
				rows = append(rows, row)
			}
		}
		pages[id] = &Page{
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
			_, ok := seen[id]
			if seen != nil {
				seen[id] = struct{}{}
			}
			// "погодинних відключень" / "погодинних вимкнень"
			if !ok && strings.Contains(e.Text, "погодинних в") {
				newsCollector.Visit(e.Request.AbsoluteURL(href))
			}
		}
	})

	homepageCollector.Visit("https://cherkasyoblenergo.com/")
	return pages
}

// format 1: ["14:00-15:00","3, 4 та 5 черги"]
// format 2: ["4.ІІ","08:00-09:30, 16:30-20:00"]
// format 3: ["2.2", "09:00 - 13:00, 15:00 - 18:00, 22:00 - 24:00"]
// (same as 2, but different shard syntax)

func Filter(pages map[int]*Page, shard int) {
	shardCh := strconv.Itoa(shard) // assume shard < 10
	for _, page := range pages {
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

func Filter2(pages map[int]*Page, shard string) {
	shardUk := strings.ReplaceAll(shard, "I", "\u0406")
	for _, page := range pages {
		page.Rows = slices.DeleteFunc(page.Rows, func(row [2]string) bool {
			return row[0] != shard && row[0] != shardUk
		})
	}
}
func Summarize2(page *Page) string {
	summary := page.Title
	for _, s := range page.Rows {
		summary += "\n" + s[0] + ": " + s[1]
	}
	return summary
}
