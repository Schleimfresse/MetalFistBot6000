package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func botInVoiceChannel(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	guild, _ := s.State.Guild(i.GuildID)
	botID := s.State.User.ID

	for _, vs := range guild.VoiceStates {
		if vs.UserID == botID {
			return true // Bot is in a voice channel
		}
	}

	return false // Bot is not in a voice channel
}

/*func sendPCM(vc *discordgo.VoiceConnection, pcm <-chan []int16) error {
	if pcm == nil {
		return fmt.Errorf("PCM nil")
	}

	var err error
	opusEncoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		return fmt.Errorf("error creating Opus encoder: %v", err)
	}

	for {
		// Read PCM data from the channel
		recv, ok := <-pcm
		if !ok {
			return fmt.Errorf("PCM channel closed")
		}

		// Encode PCM to Opus
		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			return fmt.Errorf("encoding error: %v", err)
		}

		// Check voice connection readiness before sending Opus data
		if !vc.Ready || vc.OpusSend == nil {
			return fmt.Errorf("voice connection not ready")
		}
		// Send Opus data to Discord's voice connection

		vc.OpusSend <- opus

	}
}*/

// FindBestAudioFormat finds the best audio format with the closest bitrate not higher than the target bitrate
func findBestAudioFormat(formats []youtube.Format, targetBitrate int) *youtube.Format {
	var bestAudio *youtube.Format

	availableBitrates := getBestBitrates(formats)
	closestBitrate := FindClosestBitrate(targetBitrate, availableBitrates)

	log.Println(availableBitrates, closestBitrate)

	for _, format := range formats {
		if format.Bitrate == closestBitrate {
			bestAudio = &format
			break
		}
	}

	if bestAudio == nil {
		fmt.Println(formats)
	}

	log.Println(bestAudio)
	return bestAudio
}

func FindClosestBitrate(targetBitrate int, availableBitrates []int) int {
	sort.Ints(availableBitrates) // Sort the available bitrates in ascending order

	log.Println(targetBitrate)

	closestBitrate := 0

	if len(availableBitrates) == 1 {
		return availableBitrates[0]
	} else {
		for _, num := range availableBitrates {
			if num < targetBitrate && num > closestBitrate {
				closestBitrate = num
			}
		}

		if closestBitrate == 0 {
			fmt.Println(availableBitrates, len(availableBitrates), closestBitrate)
			closestBitrate = availableBitrates[0]
		}

		return closestBitrate
	}
}

func getBestBitrates(formats []youtube.Format) []int {
	var availableBitrates []int
	for _, format := range formats {
		//	log.Println("ASR, AC: ", format.AudioSampleRate, format.AudioChannels)
		audioSampleRate, _ := strconv.Atoi(format.AudioSampleRate)
		if format.AudioChannels > 0 && audioSampleRate >= 48000 {
			availableBitrates = append(availableBitrates, format.Bitrate)
		}
	}
	if len(availableBitrates) == 0 {
		for _, format := range formats {
			//		log.Println("ASR, AC: ", format.AudioSampleRate, format.AudioChannels)
			if format.AudioChannels > 0 {
				availableBitrates = append(availableBitrates, format.Bitrate)
			}
		}
	}
	if len(availableBitrates) == 0 {
		fmt.Println(formats)
	}
	// 3rd instance
	return availableBitrates
}

func timestamp(i *discordgo.Interaction) string {
	snowflakeTimestamp := time.Now().UTC()
	log.Println("snowflakeTimestamp: ", snowflakeTimestamp)
	return snowflakeTimestamp.Format("2006-01-02T15:04:05Z")
}

/*func extractAudio(input io.ReadCloser) error {
	// Create ffmpeg command
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-vn", "-f", "opus", "D:/Bots/MetalFistBot6000/media/output.opus")

	// Set input
	cmd.Stdin = input

	// Start the command
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting ffmpeg: %v", err)
	}

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("Command failed: %v", err)
	}

	return nil
}*/

func formatDuration(duration time.Duration) string {
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60

	formattedDuration := fmt.Sprintf("%02d:%02d", minutes, seconds)
	return formattedDuration
}

func extractID(url string) string {
	// Define the regular expression pattern to match both track and playlist IDs
	trackPattern := regexp.MustCompile(`/track/(\w{22})`)
	playlistPattern := regexp.MustCompile(`/playlist/(\w{22})`)

	trackMatch := trackPattern.FindStringSubmatch(url)
	if len(trackMatch) >= 2 {
		return trackMatch[1]
	}

	// Check if the URL contains a playlist ID
	playlistMatch := playlistPattern.FindStringSubmatch(url)
	if len(playlistMatch) >= 2 {
		return playlistMatch[1]
	}

	return ""
}

func getStreamerIDFromURL(url string) (string, error) {
	// Regular expression to match the streamer ID in the Twitch URL
	re := regexp.MustCompile(`https://(?:www\.)?twitch\.tv/([a-zA-Z0-9_]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return "", fmt.Errorf("Unable to extract streamer ID from URL")
	}
	return matches[1], nil
}

func pingingInstance(s *discordgo.Session, channelID string, userToPing string) {
	for autoPingState {
		_, err := s.ChannelMessageSend(channelID, strings.Repeat("<@!"+userToPing+">", 9))
		if err != nil {
			log.Println(err)
			return
		}

		time.Sleep(time.Second)
	}
}

func setSpeakingState(state bool) {
	mu.Lock()
	defer mu.Unlock()
	speaking = state
}

func getSpeakingState() bool {
	mu.Lock()
	defer mu.Unlock()
	return speaking
}

func readLastLines() (string, error) {
	const maxChars = 4096
	const fileName = "logs.log"

	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println(err)
		}
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	size := fileInfo.Size()
	if size == 0 {
		return "", nil // File is empty
	}

	var lastLines []byte
	readSize := 0
	for {
		offset := int64(-readSize - 1)
		if offset < -size {
			offset = -size
		}

		_, err = file.Seek(offset, 2)
		if err != nil {
			return "", err
		}

		buf := make([]byte, 1)
		_, err = file.Read(buf)
		if err != nil {
			return "", err
		}

		lastLines = append(buf, lastLines...)
		readSize++

		if readSize >= maxChars || offset == -size {
			break
		}
	}

	return string(lastLines), nil
}

func totalPages(queue []ytTrack) int {
	totalEntries := len(queue)
	entries := 10
	pages := totalEntries / entries
	if totalEntries%entries != 0 {
		pages++
	}
	return pages
}
