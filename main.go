package main // import "github.com/ykzts/tea-crawler"

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var (
	target = flag.String("target", "", "")
)

func search(service *youtube.Service, channelID string, pageToken string) (*youtube.SearchListResponse, error) {
	searchCall := service.Search.List("id,snippet").
		ChannelId(channelID).
		MaxResults(50).
		Order("date").
		PageToken(pageToken).
		SafeSearch("none").
		Type("video")

	searchResponse, err := searchCall.Do()
	if err != nil {
		return nil, err
	}

	return searchResponse, nil
}

func normalize(p string) string {
	s := strings.Replace(p, "/", "／", -1)
	s = strings.Replace(s, "?", "？", -1)
	s = strings.Replace(s, ".", "．", -1)

	return s
}

func download(url string, name string) (string, error) {
	p := normalize(name)

	f, err := os.Create(p + filepath.Ext(url))
	if err != nil {
		return "", err
	}
	defer f.Close()

	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	_, err = io.Copy(f, response.Body)
	if err != nil {
		return "", err
	}

	return name, nil
}

func crawl(service *youtube.Service, channelID string) {
	nextPageToken := ""

	for {
		response, err := search(service, channelID, nextPageToken)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		for _, item := range response.Items {
			t := item.Snippet.Thumbnails.High.Url
			t = strings.TrimSuffix(t, "hqdefault.jpg")
			t += "maxresdefault.jpg"

			p, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}
			f := p.Format("20060102") + "-" + item.Id.VideoId + "-" + item.Snippet.Title

			_, err = download(t, f)
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}

		nextPageToken = response.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
}

func main() {
	flag.Parse()

	ctx := context.Background()

	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		log.Fatal("error: API key is required")
	}

	channelID := *target
	if channelID == "" {
		log.Fatal("error: Channel ID is required")
	}

	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	crawl(service, channelID)
}
