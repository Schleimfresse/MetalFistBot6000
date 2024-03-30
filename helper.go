package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"layeh.com/gopus"
	"log"
	"sort"
	"strconv"
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

func sendPCM(vc *discordgo.VoiceConnection, pcm <-chan []int16) error {
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
}

// FindBestAudioFormat finds the best audio format with the closest bitrate not higher than the target bitrate
func findBestAudioFormat(formats []youtube.Format, targetBitrate int) *youtube.Format {
	var bestAudio *youtube.Format

	availableBitrates := getBestBitrates(formats)
	closestBitrate := FindClosestBitrate(targetBitrate, availableBitrates)

	log.Println(availableBitrates)

	for _, format := range formats {
		if format.Bitrate == closestBitrate {
			bestAudio = &format
			break
		}
	}
	return bestAudio
}

func FindClosestBitrate(targetBitrate int, availableBitrates []int) int {
	sort.Ints(availableBitrates) // Sort the available bitrates in ascending order

	log.Println(targetBitrate)

	closestBitrate := 0

	for _, num := range availableBitrates {
		if num < targetBitrate && num > closestBitrate {
			closestBitrate = num
		}
	}

	return closestBitrate
}

func getBestBitrates(formats []youtube.Format) []int {
	var availableBitrates []int
	log.Println(availableBitrates)
	for _, format := range formats {
		log.Println("ASR, AC: ", format.AudioSampleRate, format.AudioChannels)
		audioSampleRate, _ := strconv.Atoi(format.AudioSampleRate)
		if format.AudioChannels > 0 && audioSampleRate >= 48000 {
			availableBitrates = append(availableBitrates, format.Bitrate)
		}
	}
	if len(availableBitrates) == 0 {
		for _, format := range formats {
			log.Println("ASR, AC: ", format.AudioSampleRate, format.AudioChannels)
			audioSampleRate, _ := strconv.Atoi(format.AudioSampleRate)
			if format.AudioChannels > 2 && audioSampleRate >= 44100 {
				availableBitrates = append(availableBitrates, format.Bitrate)
			}
		}
	}
	if len(availableBitrates) == 0 {
		for _, format := range formats {
			log.Println("ASR, AC: ", format.AudioSampleRate, format.AudioChannels)
			if format.AudioChannels > 2 {
				availableBitrates = append(availableBitrates, format.Bitrate)
			}
		}
	}
	return availableBitrates
}

func timestamp(i *discordgo.Interaction) (string, error) {
	snowflakeTimestamp, err := discordgo.SnowflakeTimestamp(i.ID)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return snowflakeTimestamp.Format("2006-01-02T15:04:05Z"), nil
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
