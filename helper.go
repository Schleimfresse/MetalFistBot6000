package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"layeh.com/gopus"
	"os"
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

var OnError = func(str string, err error) {
	prefix := "dgVoice: " + str

	if err != nil {
		os.Stderr.WriteString(prefix + ": " + err.Error())
	} else {
		os.Stderr.WriteString(prefix)
	}
}

func sendPCM(vc *discordgo.VoiceConnection, pcm <-chan []int16) error {
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

func findBestAudioFormat(formats []youtube.Format) *youtube.Format {
	var bestAudio *youtube.Format
	for _, format := range formats {
		if format.AudioQuality != "" && format.AudioChannels > 0 {
			if bestAudio == nil || format.AudioQuality == "high" {
				bestAudio = &format
			}
		}
	}
	return bestAudio
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
