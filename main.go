package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

func main() {
	urls := processInputURLs()
	startWorkers(urls)
}

func processInputURLs() []string {
	first := "https://www.youtube.com/watch?v=AZ6RJXA4o_o"
	urls := []string{}

	// fmt.Print("[Seperator -> (;)]\nEnter YT Links: ")
	// fmt.Scanln(&first)

	for _, url := range strings.Split(first, ";") {
		if playListID := checkForPlaylist(strings.Split(strings.Split(url, "/")[strings.Count(url, "/")], "&")); playListID == "false" {
			urls = append(urls, url)
		} else {
			for _, videoID := range getPlaylistVideoIDS(strings.Split(playListID, "=")[1]) {
				urls = append(urls, "https://www.youtube.com/watch?v="+videoID)
			}
		}
	}
	return urls
}

func startWorkers(urls []string) {
	var wg sync.WaitGroup
	for _, u := range [3]int{0, 1, 2} {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			processWorker(i, urls)
		}(u)
	}
	wg.Wait()
}

func processWorker(workerID int, urls []string) {
	switch workerID {
	case 0:
		fmt.Println(colorString("[+] Starting Download on "+fmt.Sprint(len(urls))+" Videos", "Yellow", "None"))
		globalMax = len(urls) * 3
		downloadVideos(urls)
	case 1:
		displayProgressBar(len(urls))
	case 2:
		calcETA()
	}
}

func displayProgressBar(urlLen int) {
	t0 := time.Now()
	time.Sleep(time.Millisecond * 500)
	for {
		percent := 100 * (float64(globalProgess+merged) / float64(globalMax))
		outof := fmt.Sprint(merged) + "/" + fmt.Sprint(urlLen)
		fmt.Print(colorString(fmt.Sprintf("%s========== %s ETA: %s [%s%%] ==========", magic_backspace, outof, globalETA, fmt.Sprintf("%.2f", percent)), "Light yellow", "Cyan"))
		if percent == 100.00 {
			fmt.Println("\n\nDuration", time.Now().Sub(t0))
			break
		}
	}
}

func calcETA() {
	time.Sleep(time.Second)
	history := []int{}
	ticker := time.NewTicker(time.Second)
	done := make(chan bool)
	old := globalProgess + merged

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				currentProgress := globalProgess + merged

				// Difference in completed steps since last call
				diff := currentProgress - old

				// Calc average since over time
				history = append(history, diff)
				average := 0
				for _, v := range history {
					average += v
				}
				average = average / len(history)

				if average != 0 {
					// To date string
					d := (time.Duration((globalMax-(currentProgress))/average) * time.Second)
					h := d / time.Hour
					d -= h * time.Hour
					m := d / time.Minute
					d -= m * time.Minute
					s := d / time.Second
					globalETA = fmt.Sprintf("%02d:%02d:%02d", h, m, s)

					// Update old value if it has changed
					if currentProgress != old {
						old = currentProgress
					}
				}
			}
		}
	}()

	if globalProgess+merged == globalMax {
		ticker.Stop()
		done <- true
	}
}
