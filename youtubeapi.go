package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

var API_KEY = os.Getenv("YOUTUBE_API_KEY")

func FillYoutubeLinkMeta(link *Link) error {
	l, err := url.Parse(link.URL)
	if err != nil {
		log.Fatal("Failed to parse link")
	}
	videoID := l.Query().Get("v")
	if videoID == "" {
		return errors.New("Invalid Link")
	}
	return fillVideoDetails(link, videoID)
}

func fillVideoDetails(link *Link, videoID string) error {
	response := struct {
		Items []struct {
			Snippet struct {
				ChannelTitle string `json:"channelTitle"`
				Title        string `json:"title"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
		}
	}{}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://www.googleapis.com/youtube/v3/videos", nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("key", API_KEY)
	q.Add("part", "snippet,contentDetails")
	q.Add("id", videoID)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	respText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(respText, &response)
	if err != nil {
		log.Fatal(err)
	}
	if len(response.Items) == 0 {
		return errors.New("No item returned from YouTube")
	}

	link.Title = response.Items[0].Snippet.Title
	link.ChannelName = response.Items[0].Snippet.ChannelTitle
	link.VideoID = videoID

	var min, sec int64
	fmt.Sscanf(response.Items[0].ContentDetails.Duration, "PT%dM%dS", &min, &sec)
	link.Duration = min*60 + sec

	return nil
}
