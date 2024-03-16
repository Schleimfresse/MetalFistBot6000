package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
)

func playAudio(v *discordgo.VoiceConnection, videoId string) error {
	client := youtube.Client{}

	video, err := client.GetVideo(videoId)
	if err != nil {
		return fmt.Errorf("error getting video info: %v", err)
	}

	format := findBestAudioFormat(video.Formats)
	if format == nil {
		log.Fatal("No audio format found")
	}

	//reader, _, err := client.GetStream(video, format)
	url, err := client.GetStreamURL(video, format)
	log.Println(video.Formats[0], url)
	response, err := http.Get(url)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}(response.Body)
	run := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "-b:a", "128k", "pipe:1")
	run.Stdin = response.Body
	stdout, err := run.StdoutPipe()
	if err != nil {
		OnError("StdoutPipe Error", err)
		return err
	}

	streamBuffer := bufio.NewReaderSize(stdout, 16384)

	err = run.Start()
	if err != nil {
		OnError("RunStart Error", err)
		return err
	}

	// prevent memory leak from residual ffmpeg streams
	defer run.Process.Kill()

	// Send "speaking" packet over the voice websocket
	err = v.Speaking(true)
	if err != nil {
		OnError("Couldn't set speaking", err)
	}

	// Send not "speaking" packet over the websocket when we finish
	defer func() {
		err := v.Speaking(false)
		if err != nil {
			OnError("Couldn't stop speaking", err)
		}
	}()

	send := make(chan []int16, 2)
	defer close(send)

	close := make(chan bool)
	go func() {
		sendPCM(v, send)
		close <- true
	}()

	for {
		// read data from ffmpeg stdout
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(streamBuffer, binary.LittleEndian, &audiobuf)
		if err == io.EOF {
			return err
		}
		if err != nil {
			OnError("error reading from ffmpeg stdout", err)
			return err
		}

		// Send received PCM to the sendPCM channel
		select {
		case send <- audiobuf:
		case <-close:
			return err
		}
	}
	return nil
}
