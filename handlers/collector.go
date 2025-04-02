package handlers

import (
	"log"
	"net/http"

	"github.com/gocolly/colly/v2"
)

var SharedCollector *colly.Collector

func CloneCollectorWithRetry() *colly.Collector {
	c := SharedCollector.Clone()

	c.OnError(func(resp *colly.Response, err error) {
		if resp.StatusCode == http.StatusForbidden && resp.Request.Ctx.Get("retried") != "true" {
			log.Printf("403 received from %s, retrying once...", resp.Request.URL)
			resp.Request.Ctx.Put("retried", "true")
			if err := resp.Request.Retry(); err != nil {
				log.Printf("Retry failed for %s: %v", resp.Request.URL, err)
			}
			return
		}
		log.Printf("Request to %s failed with status %d: %v", resp.Request.URL, resp.StatusCode, err)
	})

	return c
}
