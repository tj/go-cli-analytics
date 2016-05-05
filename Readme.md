
# go-cli-analytics

Disk-buffered analytics for CLI tools powered by [Segment](https://segment.com). Analytics for CLI tools can help determine which aspects of your tool are most valued by users, and where improvements could be made. Track command or flag usage, errors, latencies, or others, just don't be sketchy!

## How

Pass your Segemnt write key, and specify a directory name which will reside in HOME.

```go
a := analytics.New(&analytics.Config{
  WriteKey: "<write key>",
  Dir:      ".myprogram",
})
```

This package will create three files:

- ~/DIR/id – pseudo user id
- ~/DIR/events – buffered events
- ~/DIR/last_flush – state for previous flush

Track events like this:

```go
a.Track("More Something", map[string]interface{}{
  "other":    "stuff",
  "whatever": "else here",
})
```

Flush events at random, based on the previous duration time, or based on size. Note that flushing on every command will introduce ~500ms of latency, so don't do this.

```go
lastFlush, _ := a.LastFlush()
n, _ := a.Size()

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
```

## Notes

Note that there is no file-level locking at the moment for concurrent executions of your program. This may be added in the future if necessary.

## Badges

[![GoDoc](https://godoc.org/github.com/tj/go-cli-analytics?status.svg)](https://godoc.org/github.com/tj/go-cli-analytics)
![](https://img.shields.io/badge/license-MIT-blue.svg)
![](https://img.shields.io/badge/status-stable-green.svg)

---

> [tjholowaychuk.com](http://tjholowaychuk.com) &nbsp;&middot;&nbsp;
> GitHub [@tj](https://github.com/tj) &nbsp;&middot;&nbsp;
> Twitter [@tjholowaychuk](https://twitter.com/tjholowaychuk)
