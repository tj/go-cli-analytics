package main

import (
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/tj/go-cli-analytics"
)

func main() {
	log.SetHandler(text.Default)

	// use DebugLevel to view logs
	log.SetLevel(log.InfoLevel)

	a := analytics.New(&analytics.Config{
		WriteKey: "7Oow2zHAd5HNnRs0DDT1KRMHSMjD9bf7",
		Dir:      ".myapp",
	})

	a.Track("Something", nil)

	a.Track("More Something", map[string]interface{}{
		"other":    "stuff",
		"whatever": "else here",
	})

	lastFlush, _ := a.LastFlush()
	fmt.Printf("  last flush: %s\n", lastFlush)

	n, _ := a.Size()
	fmt.Printf("  events: %d\n", n)

	switch {
	case n >= 15:
		fmt.Printf("  flush due to size\n")
		a.Flush()
	case time.Now().Sub(lastFlush) >= time.Minute:
		fmt.Printf("  flush due to duration\n")
		a.Flush()
	default:
		fmt.Printf("  close\n")
		a.Close()
	}
}
