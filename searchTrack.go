package main

import (
	"context"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"os"
)

func searchYouTube(query string) (*youtube.SearchResult, error) {
	ctx := context.Background()
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	// Create a new YouTube service
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// Call the search.list method to search for videos
	call := service.Search.List([]string{"snippet"}).Q(query).MaxResults(1)
	response, err := call.Do()
	if err != nil {
		return nil, err
	}

	return response.Items[0], nil
}
