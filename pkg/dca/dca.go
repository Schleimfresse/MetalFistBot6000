package dca

import (
	"log"
	"time"
)

const (
	// The current version of the DCA format
	FormatVersion int8 = 1

	// The current version of the DCA program
	LibraryVersion string = "0.0.5"

	// The URL to the GitHub repository of DCA
	GitHubRepositoryURL string = "https://github.com/jonas747/dca"
)

type OpusReader interface {
	OpusFrame() (frame []byte, err error)
	FrameDuration() time.Duration
}

var Logger *log.Logger

// logln logs to assigned logger or standard logger
func logln(s ...interface{}) {
	if Logger != nil {
		Logger.Println(s...)
		return
	}

	log.Println(s...)
}
