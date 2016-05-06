package main

import (
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/tj/go-cli-analytics"
)

func main() {
	log.SetHandler(text.Default)

	// use DebugLevel to view logs
	// log.SetLevel(log.InfoLevel)
	log.SetLevel(log.DebugLevel)

	a := analytics.New(&analytics.Config{
		WriteKey: "7Oow2zHAd5HNnRs0DDT1KRMHSMjD9bf7",
		Dir:      ".myapp",
	})

	a.Track("Something", nil)

	a.Track("More Something", map[string]interface{}{
		"other":    "stuff",
		"whatever": "else here",
	})

	a.ConditionalFlush(15, time.Minute)
}
