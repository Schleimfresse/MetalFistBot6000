package streamlink

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

var (
	_ Client = (*client)(nil)

	// ErrStreamLinkNotFound .
	ErrStreamLinkNotFound = errors.New("streamlink not found in PATH. Check https://streamlink.github.io/install.html")
)

type Client interface {
	// Run gets the given streamer's stream URL
	Run(streamer string) ([]byte, error)
}

// client implements Client interface
type client struct {
	Options []string
}

// New create a Client instance
func New(opts ...string) (Client, error) {
	streamlink := new(client)
	streamlink.Options = append([]string{
		"streamlink",
		"--quiet",
		"--twitch-low-latency",
		"--stream-url",
		"--twitch-disable-ads",
		"$url",
		"best",
	}, opts...) // append user-defined options

	// Search in path
	v, err := exec.LookPath(streamlink.Options[0])
	if err != nil {
		return nil, ErrStreamLinkNotFound
	}

	log.Println("found [streamlink] at [%s]", v)
	streamlink.Options[0] = v
	return streamlink, nil
}

func (c *client) Run(streamer string) ([]byte, error) {
	var tmpCommand = make([]string, len(c.Options))
	copy(tmpCommand, c.Options)
	for i := range tmpCommand {
		tmpCommand[i] = strings.ReplaceAll(tmpCommand[i], "$url", fmt.Sprintf("https://www.twitch.tv/%s", streamer))

	}

	// run cmd with a timeout of 10 seconds
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Options[0], tmpCommand[1:]...)
	log.Println("running command [%s]", cmd.String())
	return cmd.Output()
}
