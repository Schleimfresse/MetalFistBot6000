package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

func playAudio(v *discordgo.VoiceConnection, url string) error {
	response, err := http.Get(url)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status code: %d", response.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}(response.Body)
	// nightcore filter: "atempo=1.06,asetrate=44100*1.25"
	// bassboost filter: "equalizer=f=500:width_type=h:width=200:g=1,equalizer=f=250:width_type=h:width=100:g=4,equalizer=f=125:width_type=h:width=50:g=4.8,equalizer=f=62:width_type=h:width=25:g=7",
	// 3d: "apulsator=hz=0.125"
	run := exec.Command("ffmpeg",
		"-i", "pipe:0", // Input from pipe
		//"-af", "atempo=1.06,asetrate=44100*1.25",
		"-f", "s16le", // Output format
		"-ar", strconv.Itoa(frameRate), // Audio sample rate
		"-ac", strconv.Itoa(channels), // Number of audio channels
		"-b:a", "128k", // Audio bitrate
		"pipe:1", // Output to pipe
	)
	run.Stdin = response.Body
	//stdout, err := run.StdoutPipe()
	if err != nil {
		OnError("StdoutPipe Error", err)
		return err
	}

	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr

	//streamBuffer := bufio.NewReaderSize(stdout, 16384)

	err = run.Start()
	if err != nil {
		OnError("RunStart Error", err)
		return err
	}
	log.Println("T")

	if err := run.Wait(); err != nil {
		fmt.Printf("Error waiting for ffmpeg command to finish: %v\n", err)
		return err
	}
	log.Println("T 1")
	// prevent memory leak from residual ffmpeg streams
	defer func() {
		run.Process.Kill()
	}()

	err = v.Speaking(true)
	setSpeakingState(true)
	if err != nil {
		OnError("Couldn't set speaking", err)
	}

	defer func() {
		err := v.Speaking(false)
		setSpeakingState(false)
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

	stdoutReader := bytes.NewReader(stdout.Bytes())

	for {
		// read data from ffmpeg stdout
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(stdoutReader, binary.LittleEndian, &audiobuf)

		if err != nil {
			log.Println(err, len(audiobuf))
			if err == io.EOF {
				break
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Println("TRIGGER")
				if len(audiobuf) > 0 {
					log.Println("TRIGGER 2")

					send <- audiobuf
				}
				break
			}
			OnError("error reading from ffmpeg stdout", err)
			return err
		}

		// Send received PCM to the sendPCM channel
		select {
		case send <- audiobuf:
		case <-close:
			return nil
		}
	}

	return nil
}

func playTrack(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := playAudio(connection, queue[0].streamUrl)
	if err != nil {
		log.Println(err)
		return
	} else {
		queue = queue[1:]
		if len(queue) > 0 {
			playTrack(s, i)
		}
	}
}

func stopPlayback(v *discordgo.VoiceConnection) error {
	// Stop speaking in the voice channel
	err := v.Speaking(false)
	if err != nil {
		return err
	}

	// Close the Opus send channel
	if v.OpusSend != nil {
		close(v.OpusSend)
	}

	return nil
}
