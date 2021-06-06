package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	config Config
)

type ConfigFile struct {
	WaitTime           string `json:"WaitTime"`
	ZeroPercentTimeout string `json:"ZeroPercentTimeout"`
	RadarrURL          string `json:"RadarrURL"`
	RadarrAPIKey       string `json:"RadarrAPIKey"`
	Blacklist          bool   `json:"Blacklist"`
}
type Config struct {
	WaitTime           time.Duration `json:"WaitTime"`
	ZeroPercentTimeout time.Duration `json:"ZeroPercentTimeout"`
	RadarrURL          string        `json:"RadarrURL"`
	RadarrAPIKey       string        `json:"RadarrAPIKey"`
	Blacklist          bool          `json:"Blacklist"`
}

func NewConfig() Config {
	WaitTime, _ := time.ParseDuration("4h00m")
	ZeroPercentTimeout, _ := time.ParseDuration("1h00m")
	RadarrURL := "http://localhost"
	RadarrAPIKey := ""
	Blacklist := true
	return Config{WaitTime, ZeroPercentTimeout, RadarrURL, RadarrAPIKey, Blacklist}
}
func NewConfigFromFile(file string) Config {
	config := NewConfig()
	var configFileStruct ConfigFile

	configFile, err := ioutil.ReadFile(file)
	if os.IsNotExist(err) {
		fmt.Println("No config found, using defaults.")
		return config
	} else {
		err = json.Unmarshal(configFile, &configFileStruct)
		if err != nil {
			log.Fatal(err)
		}

		if configFileStruct.WaitTime != "" {
			WaitTime, err := time.ParseDuration(configFileStruct.WaitTime)
			if err == nil {
				config.WaitTime = WaitTime
			} else {
				log.Println(err)
				log.Printf("Waittime is set incorrectly in config, using default - (%s) is incorrect.", configFileStruct.WaitTime)
			}
		} else {
			log.Println("Waittime is not set in config, using default.")
		}
		if configFileStruct.ZeroPercentTimeout != "" {
			ZeroPercentTimeout, err := time.ParseDuration(configFileStruct.ZeroPercentTimeout)
			if err == nil {
				config.ZeroPercentTimeout = ZeroPercentTimeout
			} else {
				log.Println(err)
				log.Printf("ZeroPercentTimeout is set incorrectly in config, using default - (%s) is incorrect.", configFileStruct.ZeroPercentTimeout)
			}
		} else {
			log.Println("ZeroPercentTimeout not set in config, using default.")
		}
		if configFileStruct.RadarrURL != "" {
			config.RadarrURL = configFileStruct.RadarrURL
		} else {
			log.Println("RadarrURL is not set in config, using default.")
		}
		if configFileStruct.RadarrAPIKey != "" {
			config.RadarrAPIKey = configFileStruct.RadarrAPIKey
		} else {
			log.Println("RadarrAPIKey is not set in config, using default.")
		}
		config.Blacklist = configFileStruct.Blacklist
	}

	return config
}

func main() {
	f, err := os.OpenFile("RadarrTorrentCleaner.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v.", err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("Starting Radarr Torrent Cleaner....")
	log.Printf("Time Is: %v.", time.Now())
	config = NewConfigFromFile("rtcleaner_config.json")

	file, err := ioutil.ReadFile("rtcleaner_queue.json")
	if os.IsNotExist(err) {
		log.Println("No previous Radarr queue file found.")
		queue, err := GetCurrentQueue()
		if err != nil {
			log.Fatalln("Error getting Radarr queue.")
			log.Fatalln(err.Error())
			os.Exit(1)
		} else {
			log.Println("Pulled the queue from Radarr, updating local file...")
			queueJSON, _ := json.Marshal(queue)
			err = ioutil.WriteFile("rtcleaner_queue.json", queueJSON, 0644)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
	} else {
		log.Printf("Got old queue, looking for torrents over %v old...", config.WaitTime)
		currentQueue, err := GetCurrentQueue()
		if err != nil {
			log.Fatal(err)
		}

		var oldQueue RadarrQueue

		err = json.Unmarshal(file, &oldQueue)
		if err != nil {
			log.Fatal(err)
		}

		currentTime := time.Now()

		for i, queueItem := range currentQueue.QueueContainers {
			if queueItem.Queue.Protocol == "torrent" {
				err, oldQueueObject := containsID(oldQueue.QueueContainers, queueItem.Queue.ID)
				if err == nil {
					if queueItem.Queue.Status == "Downloading" {
						timeSinceLastSeen := currentTime.Sub(oldQueueObject.LastSeen)
						if timeSinceLastSeen > config.WaitTime {
							if oldQueueObject.Queue.Sizeleft == queueItem.Queue.Sizeleft {
								log.Printf("Removing - %s, for lack of activity.", oldQueueObject.Queue.Movie.Title)
								removeFromRadarr(oldQueue, currentQueue, oldQueueObject)
								if err != nil {
									log.Fatalf(err.Error())
								}
							} else {
								//Torrent has progressed bump it's last time
								log.Printf("Resetting timers - %s, Progress made.", oldQueueObject.Queue.Movie.Title)
								oldQueue.QueueContainers[i].LastSeen = currentTime
							}
						} else if timeSinceLastSeen > config.ZeroPercentTimeout && queueItem.Queue.Size == queueItem.Queue.Sizeleft {
							log.Printf("Removing - %s, 0%% progress made in the last %s.", oldQueueObject.Queue.Movie.Title,config.ZeroPercentTimeout)
							err = removeFromRadarr(oldQueue, currentQueue, oldQueueObject)
							if err != nil {
								log.Fatalf(err.Error())
							}
						} else {
							log.Printf("Skipping - %s, timers not yet reached.", oldQueueObject.Queue.Movie.Title)
						}
					}
				} else {
					oldQueue.QueueContainers = append(oldQueue.QueueContainers, queueItem)
				}
			}
		}
		log.Println("Queue file updated, saving...")
		queueJSON, _ := json.Marshal(oldQueue)
		err = ioutil.WriteFile("rtcleaner_queue.json", queueJSON, 0644)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func removeFromRadarr(oldQueue RadarrQueue, currentQueue RadarrQueue, oldQueueObject QueueObjectContainer) error {
	url := fmt.Sprintf("%s/api/queue/%d?apikey=%s&blacklist=%t", config.RadarrURL, oldQueueObject.Queue.ID, config.RadarrAPIKey, config.Blacklist)
	req, err := http.NewRequest("DELETE", url, nil)
	// handle err
	if err != nil {
		log.Fatal(err)
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	// handle err
	if err != nil {
		log.Fatal(err)
		return err
	}
	if resp.StatusCode < 300 {
		log.Printf("Removed %s from Radarr.", oldQueueObject.Queue.Movie.Title)
		err, oldQueue.QueueContainers = removeByID(oldQueue.QueueContainers, oldQueueObject.Queue.ID)
		err, currentQueue.QueueContainers = removeByID(currentQueue.QueueContainers, oldQueueObject.Queue.ID)
	} else {
		log.Fatalf("Error removing %s from Radarr! %+v", oldQueueObject.Queue.Movie.Title, resp)
		return errors.New("Error removing movie from Radarr!")
	}
	return nil
}

func removeByID(list []QueueObjectContainer, ID int) (error, []QueueObjectContainer) {
	for i, a := range list {
		if a.Queue.ID == ID {
			return nil, append(list[:i], list[i+1:]...)
		}
	}
	return errors.New("Not Found."), list
}

func containsID(list []QueueObjectContainer, ID int) (error, QueueObjectContainer) {
	for _, a := range list {
		if a.Queue.ID == ID {
			return nil, a
		}
	}
	return errors.New("Not Found."), QueueObjectContainer{}
}

func GetCurrentQueue() (RadarrQueue, error) {
	url := fmt.Sprintf("%s/api/queue?apikey=%s", config.RadarrURL, config.RadarrAPIKey)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	var queue []QueueObject

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&queue)
	if err != nil {
		log.Fatal(err)
		return RadarrQueue{}, err
	}

	var queueContainers []QueueObjectContainer

	currentTime := time.Now()

	for _, queueItem := range queue {
		queueContainers = append(queueContainers, QueueObjectContainer{queueItem, currentTime})
	}
	return RadarrQueue{queueContainers, currentTime}, nil
}

type RadarrQueue struct {
	QueueContainers []QueueObjectContainer `json:"QueueContainers"`
	Time            time.Time              `json:"Time"`
}

type QueueObjectContainer struct {
	Queue    QueueObject `json:"Queue"`
	LastSeen time.Time   `json:"LastSeen"`
}

//An object in the activity queue in Radarr
type QueueObject struct {
	LastCheckedTime time.Time `json:"LastCheckedTime"`
	Movie          struct {
		Title       string `json:"title"`
		SortTitle   string `json:"sortTitle"`
		Status      string `json:"status"`
		Overview    string `json:"overview"`
		Network     string `json:"network"`
		Images      []struct {
			CoverType string `json:"coverType"`
			URL       string `json:"url"`
		} `json:"images"`
		Year              int           `json:"year"`
		Path              string        `json:"path"`
		ProfileID         int           `json:"profileId"`
		LanguageProfileID int           `json:"languageProfileId"`
		Monitored         bool          `json:"monitored"`
		Runtime           int           `json:"runtime"`
		FirstAired        time.Time     `json:"firstAired"`
		LastInfoSync      time.Time     `json:"lastInfoSync"`
		CleanTitle        string        `json:"cleanTitle"`
		ImdbID            string        `json:"imdbId"`
		TitleSlug         string        `json:"titleSlug"`
		Certification     string        `json:"certification"`
		Genres            []string      `json:"genres"`
		Tags              []interface{} `json:"tags"`
		Added             time.Time     `json:"added"`
		Ratings           struct {
			Votes int     `json:"votes"`
			Value float64 `json:"value"`
		} `json:"ratings"`
		QualityProfileID int `json:"qualityProfileId"`
		ID               int `json:"id"`
	} `json:"movie"`
	Quality struct {
		Quality struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Source     string `json:"source"`
			Resolution int    `json:"resolution"`
		} `json:"quality"`
		Revision struct {
			Version int `json:"version"`
			Real    int `json:"real"`
		} `json:"revision"`
	} `json:"quality"`
	Size                    float64       `json:"size"`
	Title                   string        `json:"title"`
	Sizeleft                float64       `json:"sizeleft"`
	Timeleft                string        `json:"timeleft"`
	EstimatedCompletionTime time.Time     `json:"estimatedCompletionTime"`
	Status                  string        `json:"status"`
	TrackedDownloadStatus   string        `json:"trackedDownloadStatus"`
	StatusMessages          []interface{} `json:"statusMessages"`
	DownloadID              string        `json:"downloadId"`
	Protocol                string        `json:"protocol"`
	ID                      int           `json:"id"`
}
