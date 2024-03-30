package main

import (
	"MetalFistBot6000/dca"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"io"
	"log"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes      = (frameSize * 2) * 2 // max size of opus data
)

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

func playTrack(s *discordgo.Session, i *discordgo.InteractionCreate) {
	playAudio(connection, queue[0], make(chan bool))
	//log.Println("out: ", err)
	/*if err != nil {
		log.Println("Error thrown from playTrack: ", err)
		return
	} else {*/
	queue = queue[1:]
	if len(queue) > 0 {
		playTrack(s, i)
	}
	//}
}

func playAudio(v *discordgo.VoiceConnection, track ytTrack, stop <-chan bool) {
	log.Println("play audio")
	options := dca.StdEncodeOptions
	options.BufferedFrames = 100
	options.FrameDuration = 20
	options.CompressionLevel = 10
	options.Bitrate = track.bitrate
	options.Channels = 2
	options.Application = dca.AudioApplicationAudio
	options.PacketLoss = 0
	log.Println(options.Bitrate)

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
		stream.Close()
	}()

	done := make(chan error)
	dca.NewStream(encodingSession, v, done)
	err = <-done
	if err != nil && err != io.EOF {
		log.Println(err)
	}
	log.Println(err)
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
