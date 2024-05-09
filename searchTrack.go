package main

import (
	"context"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"log"
)

func searchYouTube(query string) (*youtube.SearchResult, error) {
	ctx := context.Background()

	// Create a new YouTube service
	service, err := youtube.NewService(ctx, option.WithAPIKey("AIzaSyAqEwTxJ1a9ckZZaUGnKPXlifDZ6qgbQbc"))
	if err != nil {
		return nil, err
	}

	// Call the search.list method to search for videos
	call := service.Search.List([]string{"snippet"}).Q(query).MaxResults(1)
	response, err := call.Do()
	if err != nil {
		return nil, err
	}

	log.Println(response.Items[0].Id)

	return response.Items[0], nil
}
