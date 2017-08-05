// Package analytics provides a thin layer of Segment's client for tracking CLI metrics.
// Events are buffered on disk to improve user experience, only periodically flushing to
// the Segemnt API.
//
// You should call Flush() when desired in order to flush to Segment. You may choose
// to do this after a certain number of events (see Size()) have been buffered,
// or after a given duration (see LastFlush()).
//
// You should prompt the user at some point, allowing them to opt-in, otherwise
// invoke Disable() to flag for future invocations. Once disabled tracking is
// automatically a no-op.
//
// Functions of this package are not thread-safe.
package analytics

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/hashicorp/go-uuid"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	segment "github.com/segmentio/analytics-go"
)

// Event used for storage on disk.
type Event struct {
	Event      string                 `json:"event"`
	Properties map[string]interface{} `json:"properties"`
}

// Config for analytics tracker.
type Config struct {
	WriteKey string        // WriteKey from Segment
	Dir      string        // Dir relative to ~ for storing state
	Log      log.Interface // Log (optional)
}

// defaults applies the default values.
func (c *Config) defaults() {
	if c.Log == nil {
		c.Log = log.Log
	}
}

// New returns a new analytics tracker with `config`.
func New(config *Config) *Analytics {
	config.defaults()

	a := &Analytics{
		Config: config,
	}

	a.init()
	return a
}

// Analytics todo...
type Analytics struct {
	*Config
	root       string
	userID     string
	eventsFile *os.File
	events     *json.Encoder
}

// Initialize:
//
// - ~/<dir>
// - ~/<dir>/id
// - ~/<dir>/events
// - ~/<dir>/last_flush
//
func (a *Analytics) init() {
	a.initRoot()

	enabled, err := a.Enabled()
	if err != nil || !enabled {
		a.Log.Debug("disabled")
		return
	}

	a.initDir()
	a.initID()
	a.initEvents()
}

// init root directory.
func (a *Analytics) initRoot() {
	home, err := homedir.Dir()
	if err != nil {
		a.Log.WithError(err).Debug("error finding home dir")
		return
	}
	a.root = filepath.Join(home, a.Dir)
}

// init ~/<dir>.
func (a *Analytics) initDir() {
	os.Mkdir(a.root, 0755)
}

// init ~/<dir>/id.
func (a *Analytics) initID() {
	path := filepath.Join(a.root, "id")

	b, err := ioutil.ReadFile(path)
	if err == nil {
		a.userID = string(b)
		a.Log.Debug("id already created")
		return
	}

	a.Log.Debug("creating id")
	id, err := uuid.GenerateUUID()
	if err != nil {
		return
	}
	a.userID = string(id)

	err = ioutil.WriteFile(path, []byte(id), 0666)
	if err != nil {
		a.Log.WithError(err).Debug("error saving id")
		return
	}

	a.Touch()
}

// init ~/<dir>/events.
func (a *Analytics) initEvents() {
	path := filepath.Join(a.root, "events")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.WithError(err).Debug("error opening events")
		return
	}
	a.eventsFile = f

	a.events = json.NewEncoder(f)
}

// Enabled returns true if the user hasn't opted out.
func (a *Analytics) Enabled() (bool, error) {
	_, err := os.Stat(filepath.Join(a.root, "disable"))

	if os.IsNotExist(err) {
		return true, nil
	}

	return false, err
}

// Disable tracking. This method creates ~/<dir>/disable.
func (a *Analytics) Disable() error {
	a.Log.Debug("disable")
	_, err := os.Create(filepath.Join(a.root, "disable"))
	return err
}

// Enable tracking. This method removes ~/<dir>/disable.
func (a *Analytics) Enable() error {
	a.Log.Debug("enable")
	return os.Remove(filepath.Join(a.root, "disable"))
}

// Events reads the events from disk.
func (a *Analytics) Events() (v []*Event, err error) {
	f, err := os.Open(filepath.Join(a.root, "events"))
	if err != nil {
		return nil, errors.Wrap(err, "opening")
	}

	dec := json.NewDecoder(f)

	for {
		var e Event
		err := dec.Decode(&e)

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, errors.Wrap(err, "decoding")
		}

		v = append(v, &e)
	}

	return v, nil
}

// Size returns the number of events.
func (a *Analytics) Size() (int, error) {
	events, err := a.Events()
	if err != nil {
		return 0, errors.Wrap(err, "reading events")
	}

	return len(events), nil
}

// Touch ~/<dir>/last_flush.
func (a *Analytics) Touch() error {
	path := filepath.Join(a.root, "last_flush")
	return ioutil.WriteFile(path, []byte(":)"), 0755)
}

// LastFlush returns the last flush time.
func (a *Analytics) LastFlush() (time.Time, error) {
	info, err := os.Stat(filepath.Join(a.root, "last_flush"))
	if err != nil {
		return time.Unix(0, 0), err
	}

	return info.ModTime(), nil
}

// LastFlushDuration returns the last flush time delta.
func (a *Analytics) LastFlushDuration() (time.Duration, error) {
	lastFlush, err := a.LastFlush()
	if err != nil {
		return 0, nil
	}

	return time.Now().Sub(lastFlush), nil
}

// Track event `name` with optional `props`.
func (a *Analytics) Track(name string, props map[string]interface{}) error {
	if a.events == nil {
		return nil
	}

	return a.events.Encode(&Event{
		Event:      name,
		Properties: props,
	})
}

// ConditionalFlush flushes if event count is above `aboveSize`, or age is `aboveDuration`,
// otherwise Close() is called and the underlying file(s) are closed.
func (a *Analytics) ConditionalFlush(aboveSize int, aboveDuration time.Duration) error {
	age, err := a.LastFlushDuration()
	if err != nil {
		return err
	}

	size, err := a.Size()
	if err != nil {
		return err
	}

	ctx := a.Log.WithFields(log.Fields{
		"age":            age,
		"size":           size,
		"above_size":     aboveSize,
		"above_duration": aboveDuration,
	})

	switch {
	case size >= aboveSize:
		ctx.Debug("flush size")
		return a.Flush()
	case age >= aboveDuration:
		ctx.Debug("flush age")
		return a.Flush()
	default:
		return a.Close()
	}
}

// Flush the events to Segment, removing them from disk.
func (a *Analytics) Flush() error {
	if err := a.Close(); err != nil {
		return errors.Wrap(err, "closing")
	}

	events, err := a.Events()
	if err != nil {
		return errors.Wrap(err, "reading events")
	}

	client := segment.New(a.WriteKey)

	for _, event := range events {
		client.Track(&segment.Track{
			Event:      event.Event,
			UserId:     a.userID,
			Properties: event.Properties,
		})
	}

	if err := client.Close(); err != nil {
		return errors.Wrap(err, "closing client")
	}

	if err := a.Touch(); err != nil {
		return errors.Wrap(err, "touching")
	}

	return os.Remove(filepath.Join(a.root, "events"))
}

// Close the underlying file descriptor(s).
func (a *Analytics) Close() error {
	return a.eventsFile.Close()
}
