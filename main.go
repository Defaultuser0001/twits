package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"sync"
	"text/template"
	"time"
)

type FollowedChannel struct {
	Follows []struct {
		CreatedAt time.Time `json:"created_at"`
		Links     struct {
			Self string `json:"self"`
		} `json:"_links"`
		Channel struct {
			Name string `json:"name"`
		} `json:"channel"`
	} `json:"follows"`
	Total int `json:"_total"`
	Links struct {
		Self string `json:"self"`
		Next string `json:"next"`
	} `json:"_links"`
}

type Channel struct {
	Stream                       Stream
	Mature                       bool      `json:"mature"`
	Status                       string    `json:"status"`
	BroadcasterLanguage          string    `json:"broadcaster_language"`
	DisplayName                  string    `json:"display_name"`
	Game                         string    `json:"game"`
	Language                     string    `json:"language"`
	ID                           int       `json:"_id"`
	Name                         string    `json:"name"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
	Delay                        string    `json:"delay"`
	Logo                         string    `json:"logo"`
	Banner                       string    `json:"banner"`
	VideoBanner                  string    `json:"video_banner"`
	Background                   string    `json:"background"`
	ProfileBanner                string    `json:"profile_banner"`
	ProfileBannerBackgroundColor string    `json:"profile_banner_background_color"`
	Partner                      bool      `json:"partner"`
	URL                          string    `json:"url"`
	Views                        int       `json:"views"`
	Followers                    int       `json:"followers"`
	Links                        struct {
		Self          string `json:"self"`
		Follows       string `json:"follows"`
		Commercial    string `json:"commercial"`
		StreamKey     string `json:"stream_key"`
		Chat          string `json:"chat"`
		Features      string `json:"features"`
		Subscriptions string `json:"subscriptions"`
		Editors       string `json:"editors"`
		Teams         string `json:"teams"`
		Videos        string `json:"videos"`
	} `json:"_links"`
}

type Stream struct {
	Game        string    `json:"game"`
	Viewers     int       `json:"viewers"`
	AverageFps  float64   `json:"average_fps"`
	Delay       int       `json:"delay"`
	VideoHeight int       `json:"video_height"`
	IsPlaylist  bool      `json:"is_playlist"`
	CreatedAt   time.Time `json:"created_at"`
	ID          int64     `json:"_id"`
	Channel     struct {
		Mature                       bool        `json:"mature"`
		Status                       string      `json:"status"`
		BroadcasterLanguage          string      `json:"broadcaster_language"`
		DisplayName                  string      `json:"display_name"`
		Game                         string      `json:"game"`
		Delay                        interface{} `json:"delay"`
		Language                     string      `json:"language"`
		ID                           int         `json:"_id"`
		Name                         string      `json:"name"`
		CreatedAt                    time.Time   `json:"created_at"`
		UpdatedAt                    time.Time   `json:"updated_at"`
		Logo                         string      `json:"logo"`
		Banner                       string      `json:"banner"`
		VideoBanner                  string      `json:"video_banner"`
		Background                   interface{} `json:"background"`
		ProfileBanner                string      `json:"profile_banner"`
		ProfileBannerBackgroundColor string      `json:"profile_banner_background_color"`
		Partner                      bool        `json:"partner"`
		URL                          string      `json:"url"`
		Views                        int         `json:"views"`
		Followers                    int         `json:"followers"`
		Links                        struct {
			Self          string `json:"self"`
			Follows       string `json:"follows"`
			Commercial    string `json:"commercial"`
			StreamKey     string `json:"stream_key"`
			Chat          string `json:"chat"`
			Features      string `json:"features"`
			Subscriptions string `json:"subscriptions"`
			Editors       string `json:"editors"`
			Teams         string `json:"teams"`
			Videos        string `json:"videos"`
		} `json:"_links"`
	} `json:"channel"`
	Preview struct {
		Small    string `json:"small"`
		Medium   string `json:"medium"`
		Large    string `json:"large"`
		Template string `json:"template"`
	} `json:"preview"`
	Links struct {
		Self string `json:"self"`
	} `json:"_links"`
}
type Streams struct {
	Streams []Stream
	Total   int `json:"_total"`
	Links   struct {
		Self     string `json:"self"`
		Next     string `json:"next"`
		Featured string `json:"featured"`
		Summary  string `json:"summary"`
		Followed string `json:"followed"`
	} `json:"_links"`
}
type Page struct {
	FollowedChannels []Channel
	CsStreams        []Stream
	LolStreams       []Stream
}
type Channels []Channel
type CahnnelList struct {
	Channels []Channel
}

func (a Channels) Len() int           { return len(a) }
func (a Channels) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Channels) Less(i, j int) bool { return a[i].Stream.Channel.Name < a[j].Stream.Channel.Name }

var vlcUrl string
var twitchUsername string

func main() {
	flag.StringVar(&vlcUrl, "vlc", "", "set path to vlc (for linux just vlc is enough)")
	flag.StringVar(&twitchUsername, "username", "", "set twitch username")
	flag.Parse()
	if vlcUrl == "" || twitchUsername == "" {
		log.Fatal("Flags must be set")
	}
	http.HandleFunc("/", listTwitchStreams)
	http.HandleFunc("/start_stream", startStream)
	http.Handle("/frontend/", http.StripPrefix("/frontend/", http.FileServer(http.Dir("frontend"))))
	if err := http.ListenAndServe(":9797", nil); err != nil {
		log.Fatalf("Server crashed\n%s", err)
	}
}

func listTwitchStreams(w http.ResponseWriter, r *http.Request) {
	getRequest := fmt.Sprintf("https://api.twitch.tv/kraken/users/%s/follows/channels", twitchUsername)
	resp, err := http.Get(getRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Request '%s' gave an the following error: %v", getRequest, err)
	}
	defer resp.Body.Close()

	var followedChannels FollowedChannel
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&followedChannels)
	if err != nil {
		fmt.Errorf("Could not parse response body: %f", err)
	}
	var wg sync.WaitGroup
	channels := &CahnnelList{}
	wg.Add(len(followedChannels.Follows))
	for _, following := range followedChannels.Follows {
		getStreamRequest := fmt.Sprintf("https://api.twitch.tv/kraken/streams/%s", following.Channel.Name)

		go func(getStreamRequest string, channels *CahnnelList) {
			defer wg.Done()
			resp, err := http.Get(getStreamRequest)
			if err != nil {
				fmt.Errorf("Request '%s' gave the following error: %v", getRequest, err)
			}
			defer resp.Body.Close()
			var channel Channel
			decoder := json.NewDecoder(resp.Body)
			if err = decoder.Decode(&channel); err != nil {
				fmt.Errorf("Could not parse response body: %f", err)
			}
			emptyStream := Stream{}
			if channel.Stream != emptyStream {
				channels.Channels = append(channels.Channels, channel)
			}
		}(getStreamRequest, channels)
	}
	wg.Wait()
	csStreams := Streams{}
	getStreamsByGame("Counter-Strike:%20Global%20Offensive", 20, &csStreams)
	lolStreams := Streams{}
	getStreamsByGame("League%20of%20Legends", 20, &lolStreams)
	sort.Sort(Channels(channels.Channels))
	t, err := template.ParseFiles("frontend/templates/homepage.html")
	if err != nil {
		http.Error(w, fmt.Errorf("error occurd loading template: %v", err).Error(), http.StatusInternalServerError)
		return
	}
	page := Page{
		FollowedChannels: channels.Channels,
		CsStreams:        csStreams.Streams,
		LolStreams:       lolStreams.Streams,
	}

	if err = t.Execute(w, page); err != nil {
		http.Error(w, fmt.Errorf("error occurd: %v", err).Error(), http.StatusInternalServerError)
		return
	}
}

func getStreamsByGame(game string, limit int, streams *Streams) error {
	csRequest := fmt.Sprintf("https://api.twitch.tv/kraken/streams?game=%s&limit=%d", game, limit)
	response, err := http.Get(csRequest)
	if err != nil {
		return fmt.Errorf("Request '%s' gave an the following error: %v", csRequest, err)
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&streams)
	if err != nil {
		return fmt.Errorf("Could not parse response body: %f", err)
	}
	return nil
}

func startStream(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		http.Error(w, fmt.Errorf("Could not start stream channel name cannot be empty").Error(), http.StatusBadRequest)
		return
	}
	go startLivestreamer(fmt.Sprintf("http://www.twitch.tv/%s", channel), "best", vlcUrl)
	channelRequest := fmt.Sprintf("https://api.twitch.tv/kraken/channels/%s", channel)
	response, err := http.Get(channelRequest)
	if err != nil {
		http.Error(w, fmt.Errorf("Request '%s' gave an the following error: %v", channelRequest, err).Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	channelResp := Channel{}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&channelResp)
	if err != nil {
		http.Error(w, fmt.Errorf("Could not parse response body: %f", err).Error(), http.StatusInternalServerError)
		return
	}
	t, err := template.ParseFiles("frontend/templates/channel.html")

	if err != nil {
		http.Error(w, fmt.Errorf("error occurd loading template: %v", err).Error(), http.StatusInternalServerError)
		return
	}
	if err = t.Execute(w, channelResp); err != nil {
		http.Error(w, fmt.Errorf("error occurd: %v", err).Error(), http.StatusInternalServerError)
		return
	}

}

func startLivestreamer(url string, quality string, playerPath string) error {
	if err := executeCommand(exec.Command("livestreamer", url, quality, fmt.Sprintf("--player=%s", playerPath))); err != nil {
		return fmt.Errorf("Could not start livestreamer:\n%v", err)
	}
	return nil
}

func executeCommand(cmd *exec.Cmd) error {
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to load stream: %s", err)
	}
	return nil
}
