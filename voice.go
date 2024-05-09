package main

import (
	"MetalFistBot6000/pkg/dca"
	"MetalFistBot6000/pkg/streamlink"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	channels int = 2 // 1 for mono, 2 for stereo
	//frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes      = (frameSize * 2) * 2 // max size of opus data
)

var bitrate int

/*func playAudio(v *discordgo.VoiceConnection, video *youtube.Video, stop <-chan bool) error {
	/*response, err := http.Get(video)
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
	format := findBestAudioFormat(video.Formats)
	client := youtube.Client{}
	reader, _, err := client.GetStream(video, format)
	if err != nil {
		return fmt.Errorf("error getting audio stream: %v", err)
	}
	// nightcore filter: "atempo=1.06,asetrate=44100*1.25"
	// bassboost filter: "equalizer=f=500:width_type=h:width=200:g=1,equalizer=f=250:width_type=h:width=100:g=4,equalizer=f=125:width_type=h:width=50:g=4.8,equalizer=f=62:width_type=h:width=25:g=7",
	// 3d: "apulsator=hz=0.125"
	run := exec.Command("ffmpeg",
		"-i", "pipe:0", // Input from pipe
		"-vn",
		//"-af", "atempo=1.06,asetrate=44100*1.25",
		"-f", "s16le", // Output format
		"-ar", strconv.Itoa(frameRate), // Audio sample rate
		"-ac", strconv.Itoa(channels), // Number of audio channels
		//"-b:a", "128k", // Audio bitrate
		"pipe:1", // Output to pipe
	)
	run.Stdin = reader
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe Error: %v", err)
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	// Starts the ffmpeg command
	err = run.Start()
	if err != nil {
		return fmt.Errorf("RunStart Error: %v", err)
	}

	// prevent memory leak from residual ffmpeg streams
	defer func() {
		run.Process.Kill()
		reader.Close()
	}()

	//when stop is sent, kill ffmpeg
	go func() {
		<-stop
		err = run.Process.Kill()
	}()

	// Send "speaking" packet over the voice websocket
	setSpeakingState(true)
	err = v.Speaking(true)
	if err != nil {
		fmt.Println("Couldn't set speaking", err)
	}

	// Send not "speaking" packet over the websocket when we finish
	defer func() {
		setSpeakingState(false)
		err := v.Speaking(false)
		if err != nil {
			fmt.Println("Couldn't stop speaking", err)
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
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Errorf("EOF error: %v", err)
			//return fmt.Errorf("EOF error: %v", err)
		}
		if err != nil {
			return fmt.Errorf("error reading from ffmpeg stdout: %v", err)
		}

		// Send received PCM to the sendPCM channel
		select {
		case send <- audiobuf:
		case <-close:
			return nil
		}
	}
}*/

func playQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(queue) == 0 {
		file, err := os.Open("./data/queue_end.mp3")
		if err != nil {
			log.Println(err)
			return
		}
		reader, err := dca.EncodeMem(file, dca.StdEncodeOptions)
		done := make(chan error)
		dca.NewStream(reader, connection, done)

		if err != nil {
			log.Println(err)
			return
		}
		<-done
		setSpeakingState(false)
		reader.Cleanup()

		return
	}

	track := queue[0]

	done := make(chan bool)
	go func() {
		playAudio(connection, track, done)
	}()

	<-done
	defer func() {
		if len(queue) > 0 {
			queue = queue[1:]
			playQueue(s, i)
		}
	}()
}

func playAudio(v *discordgo.VoiceConnection, track ytTrack, done chan bool) {
	setSpeakingState(true)
	err := v.Speaking(true)
	if err != nil {
		fmt.Println("Couldn't set speaking", err)
	}

	// Announcer

	key := os.Getenv("ELEVEN_LABS")
	url := "https://api.elevenlabs.io/v1/text-to-speech/nPczCjzI2devNBz1zQrb?optimize_streaming_latency=0&output_format=mp3_44100_128"

	payload := strings.NewReader("{\n  \"text\": \"Now playing: " + track.title + "\",\n  \"voice_settings\": {\n    \"similarity_boost\": 0.75,\n    \"stability\": 0.5,\n    \"use_speaker_boost\": true  }\n}")

	req, err := http.NewRequest("POST", url, payload)
	req.Header.Add("xi-api-key", key)
	req.Header.Add("Content-Type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	reader, err := dca.EncodeMem(res.Body, dca.StdEncodeOptions)
	end := make(chan error)
	dca.NewStream(reader, connection, end)
	if err != nil {
		log.Println(err)
		return
	}
	<-end

	//////////////////////////////////////////////////////////////////////////

	options := dca.StdEncodeOptions
	options.BufferedFrames = 100
	options.FrameDuration = 20
	options.CompressionLevel = 5
	options.Bitrate = track.bitrate

	client := youtube.Client{}
	stream, i2, err := client.GetStream(track.video, track.format)

	log.Println("i2: ", i2)
	if err != nil {
		log.Println(err)
	}

	encodingSession, err := dca.EncodeMem(stream, options)
	if err != nil {
		log.Println(err)
	}

	defer func() {
		encodingSession.Cleanup()
		err := stream.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	full := make(chan error)
	go func() {
		streamingSession := dca.NewStream(encodingSession, v, full)
		for {
			fmt.Println("FOR LOOP", playbackPositionSignal, pause)
			select {
			case <-playbackPositionSignal:
				playbackPositionData <- streamingSession.PlaybackPosition()
			case <-pause:
				streamingSession.SetPaused(!streamingSession.Paused())
				isPaused <- streamingSession.Paused()
			}
		}
	}()

	select {
	case <-skip:
		log.Println("SKIP")
		done <- true
	case err := <-full:
		log.Println("END", err)
		if err != nil && err != io.EOF && errors.Is(err, errors.New("Voice connection closed")) {
			log.Println(err)
			done <- false
		}

		done <- true
	}
}

func playLiveStream(v *discordgo.VoiceConnection, url string) {
	client, err := streamlink.New()
	if err != nil {
		log.Fatalf("Error creating streamlink client: %v", err)
	}

	streamer, err := getStreamerIDFromURL(url)

	streamUrl, err := client.Run(streamer)

	setSpeakingState(true)
	err = v.Speaking(true)
	if err != nil {
		fmt.Println("Couldn't set speaking", err)
	}

	options := dca.StdEncodeOptions
	options.BufferedFrames = 100
	//options.Bitrate = queue[0].bitrate
	options.FrameDuration = 20
	options.CompressionLevel = 5

	encodingSession, err := dca.EncodeFile(string(streamUrl), options)

	defer func() {
		encodingSession.Cleanup()
		//stdout.Close()
		setSpeakingState(false)
		err := v.Speaking(false)
		if err != nil {
			fmt.Println("Couldn't set speaking", err)
		}
		log.Println("playLiveStream: function ended")
	}()

	log.Println("isSpeaking:", getSpeakingState())

	full := make(chan error)
	go func() {
		dca.NewStream(encodingSession, v, full)
	}()

	select {
	case <-skip:
		log.Println("SKIP")

	case err := <-full:
		log.Println("END", err)
		if err != nil && err != io.EOF && !errors.Is(err, errors.New("Voice connection closed")) {
			log.Println(err)
		}
	}
}
