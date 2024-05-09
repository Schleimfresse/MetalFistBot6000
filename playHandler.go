package main

import (
	"context"
	"fmt"
	"github.com/kkdai/youtube/v2"
	"github.com/zmb3/spotify/v2"
	"log"
	"strconv"
)

func addTrack(url string, playNext bool) string {
	client := youtube.Client{}
	log.Println(url)
	video, err := client.GetVideo(url)

	if err != nil {
		log.Println(err)
		return ""
	}
	format := findBestAudioFormat(video.Formats, bitrate)
	if format == nil {
		log.Fatal("No audio format found")
	}

	numInt, err := strconv.Atoi(format.ApproxDurationMs)
	if err != nil {
		log.Println(err)
		return ""
	}
	log.Println("END STATS: ", format.Bitrate, format.AverageBitrate, format.AudioSampleRate, numInt/1000)

	streamUrl, err := client.GetStreamURL(video, format)

	track := ytTrack{title: video.Title, streamUrl: streamUrl, duration: video.Duration, author: video.Author, publishDate: video.PublishDate, video: video, bitrate: format.Bitrate, audiosamplerate: format.AudioSampleRate, format: format, thumbnail: video.Thumbnails[len(video.Thumbnails)-1].URL}

	if playNext {
		insertIndex := 1
		for i := len(queue) - 1; i > insertIndex; i-- {
			queue[i] = queue[i-1]
		}
		queue[insertIndex] = track
	} else {
		queue = append(queue, track)
	}
	return video.Title
}

func addPlaylist(url string, playNext bool) string {
	client := youtube.Client{}

	playlist, err := client.GetPlaylist(url)
	if err != nil {
		log.Println(err)
		return ""
	}

	for playNextIndex, entry := range playlist.Videos {
		video, err := client.GetVideo(entry.ID)
		if err != nil {
			log.Println(err)
			return ""
		}
		format := findBestAudioFormat(video.Formats, bitrate)
		if format == nil {
			log.Fatal("No audio format found")
		}

		_, err = strconv.Atoi(format.ApproxDurationMs)
		if err != nil {
			log.Println(err)
			return ""
		}
		//log.Println("END STATS: ", format.Bitrate, format.AverageBitrate, format.AudioSampleRate, numInt/1000)

		url, err := client.GetStreamURL(video, format)

		track := ytTrack{Id: video.ID, title: video.Title, streamUrl: url, duration: video.Duration, author: video.Author, publishDate: video.PublishDate, video: video, bitrate: format.Bitrate, audiosamplerate: format.AudioSampleRate, thumbnail: video.Thumbnails[len(video.Thumbnails)-1].URL, format: format}

		if playNext {

			queue = append(queue[:playNextIndex+1], queue[playNextIndex:]...)
			queue[playNextIndex+1] = track
		} else {
			queue = append(queue, track)
		}
	}
	return playlist.Title
}

func spotifyTrackHandler(url string, playNext bool) string {
	ctx := context.Background()
	id := extractID(url)

	spTrack, err := spClient.GetTrack(ctx, spotify.ID(id))
	if err != nil {
		log.Println(err)
		return ""
	}
	result, err := searchYouTube(fmt.Sprint(spTrack.Name, " ", spTrack.Artists))
	if err != nil {
		log.Println(err)
		return ""
	}

	requestedUnit := addTrack(fmt.Sprint("https://music.youtube.com/watch?v=", result.Id.VideoId), playNext)
	return requestedUnit
}

func spotifyPlaylistHandler(url string, playNext bool) string {
	var playlist []string
	ctx := context.Background()
	id := extractID(url)
	log.Println(id)

	spPlaylist, err := spClient.GetPlaylist(ctx, spotify.ID(id))
	if err != nil {
		log.Println(err)
	}

	requestedUnit := spPlaylist.Name
	if err != nil {
		log.Println(err)
		return ""
	}
	for _, track := range spPlaylist.Tracks.Tracks {
		query := fmt.Sprint(track.Track.Name, track.Track.Artists)
		playlist = append(playlist, query)
	}
	for {
		tracks, err := spClient.GetPlaylistItems(context.Background(), spotify.ID(id), spotify.Offset(len(playlist)))
		if err != nil {
			log.Println(err)
			return ""
		}
		for _, track := range tracks.Items {
			query := fmt.Sprint(track.Track.Track.Name, track.Track.Track.Artists)
			playlist = append(playlist, query)
		}
		log.Println(len(playlist), tracks.Next)
		if tracks.Next == "" {
			break
		}
	}

	youtubeTracks := make(chan []string)

	go func() {
		var results []string

		for _, track := range playlist {
			searchResult, err := searchYouTube(track)
			if err != nil {
				log.Println(err)
			} else {
				results = append(results, fmt.Sprint("https://music.youtube.com/watch?v=", searchResult.Id))
			}
		}

		youtubeTracks <- results
		close(youtubeTracks)
	}()

	addSpotifyPlaylist(youtubeTracks, playNext)
	return requestedUnit
}

func addSpotifyPlaylist(playlist <-chan []string, playNext bool) {
	client := youtube.Client{}

	for playNextIndex, url := range <-playlist {
		video, err := client.GetVideo(url)
		if err != nil {
			log.Println(err)
		}
		format := findBestAudioFormat(video.Formats, bitrate)
		if format == nil {
			log.Fatal("No audio format found")
		}

		_, err = strconv.Atoi(format.ApproxDurationMs)
		if err != nil {
			log.Println(err)
		}
		//log.Println("END STATS: ", format.Bitrate, format.AverageBitrate, format.AudioSampleRate, numInt/1000)

		url, err := client.GetStreamURL(video, format)

		track := ytTrack{Id: video.ID, title: video.Title, streamUrl: url, duration: video.Duration, author: video.Author, publishDate: video.PublishDate, video: video, bitrate: format.Bitrate, audiosamplerate: format.AudioSampleRate, thumbnail: video.Thumbnails[len(video.Thumbnails)-1].URL, format: format}

		if playNext {

			queue = append(queue[:playNextIndex+1], queue[playNextIndex:]...)
			queue[playNextIndex+1] = track
		} else {
			queue = append(queue, track)
		}
	}
}

func twitchHandler(url string, playNext bool, chanBitrate int) string {
	//queue = append(queue)
	playLiveStream(connection, url)
	return ""
}
