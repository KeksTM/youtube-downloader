package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var globalMaxTimeout int = 500 // Max Timeout for Download Requests in MS
var CHUNK_SIZE int = 1603480   // Max Download Chunk Size for each part of the video buffer [Dont change unless YT API is fucky wucky]
var globalProgess int = 0
var globalETA string = ""
var globalMax int = 0 // Max possible "steps" able, used to compute eta and progress
var merged int = 0    // Merged Videos

var magic_space string = "                                                            "
var magic_backspace string = "\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b"

type YoutubeResponseData struct {
	StreamingData struct {
		AdaptiveFormats []struct {
			Bitrate       int64
			ContentLength string
			MimeType      string
			Quality       string
			QualityLabel  string
			URL           string
		}
		ExpiresInSeconds string
	}
	VideoDetails struct {
		Title   string
		VideoID string
	}
}

type PlaylistResponseData struct {
	ResponseContext struct {
		VisitorData string
	}
	Contents struct {
		TwoColumnWatchNextResults struct {
			Playlist struct {
				Playlist struct {
					Contents []struct {
						PlaylistPanelVideoRenderer struct {
							VideoID string
						}
					}
					TotalVideosText struct {
						Runs []struct {
							Text string
						}
					}
				}
			}
		}
	}
}

func main() {
	first := "https://www.youtube.com/watch?v=AZ6RJXA4o_o" //"https://www.youtube.com/watch?v=ajTMOR7Ke_I"
	urls := []string{}

	// fmt.Print("[Seperator -> (;)]\nEnter YT Links: ")
	// fmt.Scanln(&first)

	for _, url := range strings.Split(first, ";") {

		// Check if it is a playlist, returns "false" if false else playlistID
		playListID := checkForPlaylist(strings.Split(strings.Split(url, "/")[strings.Count(url, "/")], "&"))
		if playListID == "false" {
			urls = append(urls, url)
		} else {
			// Add all videos from playlist
			for _, videoID := range getPlaylistVideoIDS(strings.Split(playListID, "=")[1]) {
				urls = append(urls, "https://www.youtube.com/watch?v="+videoID)
			}
		}
	}

	// Start threads for ui handling and downloading
	var wg sync.WaitGroup
	for _, u := range [3]int{0, 1, 2} {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i {
			case 0:
				fmt.Println(colorString("[+] Starting Download on "+fmt.Sprint(len(urls))+" Videos", "Yellow", "None"))
				globalMax = len(urls) * 3
				downloadVideos(urls)
			case 1:
				displayProgressBar(len(urls))
			case 2:
				calcETA()
			}
		}(u)
	}
	wg.Wait()
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
					// To date string [Im sure there is a good lib for this but oh well]
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

func getPlaylistVideoIDS(playlistId string) []string {
	var JSONStruct PlaylistResponseData = PlaylistResponseData{}
	videoIDs := make(map[string]struct{})

	playlistIndex := 0
	videoId := "null"
	visitorData := "null"
	old := make(map[string]struct{})

	for {
		body := fmt.Sprintf(`{
            "context": {
                "client": {
                    "clientName": "WEB",
                    "clientVersion": "2.20210408.08.00",
                    "gl": "US",
                    "hl": "en",
                    "utcOffsetMinutes": 0,
                    "visitorData": %s
                }
            },
            "playlistId": "%s",
            "playlistIndex": %d,
            "videoId": %s
        }`, visitorData, playlistId, playlistIndex, videoId)

		derr := json.Unmarshal(youtubeAPIWrapper("https://www.youtube.com/youtubei/v1/next?key=AIzaSyA8eiZmM1FaDVjRy-df2KTyQ_vz_yYM39w&hl=en", body), &JSONStruct)
		if derr != nil {
			panic(derr)
		}

		last := ""
		for _, v := range JSONStruct.Contents.TwoColumnWatchNextResults.Playlist.Playlist.Contents {
			if v.PlaylistPanelVideoRenderer.VideoID != "" {
				videoIDs[v.PlaylistPanelVideoRenderer.VideoID] = struct{}{}
			}
			last = v.PlaylistPanelVideoRenderer.VideoID
		}

		if !reflect.DeepEqual(old, videoIDs) || len(videoIDs) < strToInt(strings.ReplaceAll(JSONStruct.Contents.TwoColumnWatchNextResults.Playlist.Playlist.TotalVideosText.Runs[0].Text, ",", "")) {
			old = videoIDs
			playlistIndex = len(videoIDs)
			videoId = `"` + last + `"`
			if visitorData == "null" {
				visitorData = `"` + JSONStruct.ResponseContext.VisitorData + `"`
			}
			continue
		}
		break
	}

	out := []string{}
	for i := range videoIDs {
		out = append(out, i)
	}

	return out
}

func getAPIRequestAsJSON(url string) YoutubeResponseData {
	VideoID := ""
	if strings.Contains(url, "youtu.be") {
		VideoID = strings.Split(url, "/")[3]
	} else if strings.Contains(url, "youtube.com") {
		VideoID = strings.Split(url, "=")[1]
	} else {
		panic("Not a valid Youtube ID")
	}

	// JSON body
	body := fmt.Sprintf(`{
		"context": {
			"client": {
				"hl": "en",
				"gl": "LU",
				"visitorData": "Cgt2Q2tnZkZ6Smd3QSjkiISoBjIGCgJOTBIA",
				"userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/117.0,gzip(gfe)",
				"clientName": "WEB",
				"clientVersion": "2.20230912.00.00",
				"osName": "Windows",
				"osVersion": "10.0",
				"originalUrl": "https://www.youtube.com/watch?v=%s",
				"platform": "DESKTOP",
				"clientFormFactor": "UNKNOWN_FORM_FACTOR",
				"userInterfaceTheme": "USER_INTERFACE_THEME_DARK",
				"timeZone": "Europe/Berlin",
				"browserName": "Firefox",
				"browserVersion": "117.0",
				"acceptHeader": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
				"deviceExperimentId": "ChxOekkzT0RFd016TXdNVFEwTnprMU5qa3dPQT09EOSIhKgGGOSIhKgG",
				"screenWidthPoints": 1920,
				"screenHeightPoints": 392,
				"screenPixelDensity": 1,
				"screenDensityFloat": 1,
				"utcOffsetMinutes": 120,
				"clientScreen": "WATCH",
				"mainAppWebInfo": {
					"graftUrl": "/watch?v=%s",
					"pwaInstallabilityStatus": "PWA_INSTALLABILITY_STATUS_UNKNOWN",
					"webDisplayMode": "WEB_DISPLAY_MODE_BROWSER",
					"isWebNativeShareAvailable": false
				}
			},
			"user": {
				"lockedSafetyMode": false
			},
			"request": {
				"useSsl": true,
				"internalExperimentFlags": [
					{
						"key": "force_enter_once_in_webview",
						"value": "true"
					}
				],
				"consistencyTokenJars": []
			},
			"adSignalsInfo": {
				"params": [
					{
						"key": "dt",
						"value": "1694565474885"
					},
					{
						"key": "flash",
						"value": "0"
					},
					{
						"key": "frm",
						"value": "0"
					},
					{
						"key": "u_tz",
						"value": "120"
					},
					{
						"key": "u_his",
						"value": "10"
					},
					{
						"key": "u_h",
						"value": "1080"
					},
					{
						"key": "u_w",
						"value": "1920"
					},
					{
						"key": "u_ah",
						"value": "1040"
					},
					{
						"key": "u_aw",
						"value": "1920"
					},
					{
						"key": "u_cd",
						"value": "24"
					},
					{
						"key": "bc",
						"value": "31"
					},
					{
						"key": "bih",
						"value": "392"
					},
					{
						"key": "biw",
						"value": "1903"
					},
					{
						"key": "brdim",
						"value": "-8,-8,-8,-8,1920,0,1936,1056,1920,392"
					},
					{
						"key": "vis",
						"value": "1"
					},
					{
						"key": "wgl",
						"value": "true"
					},
					{
						"key": "ca_type",
						"value": "image"
					}
				]
			}
		},
		"videoId": "%s",
		"startTimeSecs": 0,
		"playbackContext": {
			"contentPlaybackContext": {
				"currentUrl": "/watch?v=%s",
				"vis": 0,
				"splay": false,
				"autoCaptionsDefaultOn": false,
				"autonavState": "STATE_NONE",
				"html5Preference": "HTML5_PREF_WANTS",
				"signatureTimestamp": 19605,
				"referer": "https://www.youtube.com/@qdance/videos",
				"lactMilliseconds": "-1",
				"watchAmbientModeContext": {
					"hasShownAmbientMode": true,
					"watchAmbientModeEnabled": true
				}
			}
		},
		"racyCheckOk": false,
		"contentCheckOk": false
	}`, VideoID, VideoID, VideoID, VideoID)

	JSONStruct := YoutubeResponseData{}
	derr := json.Unmarshal(youtubeAPIWrapper("https://www.youtube.com/youtubei/v1/player?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8&prettyPrint=false", body), &JSONStruct)
	if derr != nil {
		panic(derr)
	}

	return JSONStruct
}

func youtubeAPIWrapper(apiURL string, body string) []byte {
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/117.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Youtube-Bootstrap-Logged-In", "true")
	req.Header.Set("X-Youtube-Client-Name", "1")
	req.Header.Set("X-Youtube-Client-Version", "2.20230912.00.00")
	req.Header.Set("X-Origin", "https://www.youtube.com")
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "same-origin")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	x, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return x
}

func currentDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Unable to get the current filename")
	}
	return filepath.Dir(filename)
}

func downloadVideos(urls []string) {
	var wg sync.WaitGroup
	for _, u := range urls {
		// Anti-Ratelimiting
		time.Sleep(time.Duration(clamp(8*len(urls), 0, globalMaxTimeout) * int(time.Millisecond)))

		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			// Call Youtubes API and fetch Video Data
			JSONData := YoutubeResponseData{}
		panic:
			for {
				JSONData = getAPIRequestAsJSON(url)
				if fmt.Sprint(JSONData.StreamingData.AdaptiveFormats) != "[]" {
					break
				}

				time.Sleep(time.Second)
			}

			if strToInt(JSONData.StreamingData.ExpiresInSeconds) < 0 {
				return
			}

			audioData := make(map[int]string)
			videoData := make(map[int]string)

			// Get all the Download data for Audio/Video
			for index, value := range JSONData.StreamingData.AdaptiveFormats {
				if strings.Contains(value.MimeType, "video/mp4") {
					videoData[index] = value.Quality
				} else if strings.Contains(value.MimeType, "audio/mp4") {
					audioData[index] = fmt.Sprint(value.Bitrate)
				}
			}

			if reflect.DeepEqual(audioData, make(map[int]string)) || reflect.DeepEqual(videoData, make(map[int]string)) {
				goto panic
			}

			// Carve out Download Urls
			audioIndex := JSONData.StreamingData.AdaptiveFormats[sortAudioByVal(audioData)]
			for _, i := range sortVideoByVal(videoData) {
				fmt.Print(JSONData.StreamingData.AdaptiveFormats[i])
				// var c bool
				// if c = true; JSONData.StreamingData.AdaptiveFormats[i].URL == "" {
				// 	c = false
				// }
				fmt.Println(JSONData.StreamingData.AdaptiveFormats[i].URL)
			}
			os.Exit(1)
			videoIndex := JSONData.StreamingData.AdaptiveFormats[0]
			// Start Downloading
			strArr := [2]string{JSONData.VideoDetails.Title, JSONData.VideoDetails.VideoID}

			chunkedAsyncDownload(strArr, videoIndex.URL, videoIndex.ContentLength, audioIndex.URL, audioIndex.ContentLength)
		}(u)
	}
	wg.Wait()
}

func DownloadFile(url string, filename string, extension string, i int) {
restart:
	resp, err := http.Get(url)

	if err != nil {
		goto restart
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		goto restart
	}

	if resp.StatusCode == 200 {
		writeFile(currentDir()+"\\Downloads\\"+filename+fmt.Sprint(i)+extension, string(b))
	} else {
		fmt.Println(colorString(fmt.Sprintf("[+] Got \"%s\" while downloading %s", resp.Status, filename), "Light yellow", "None"))
		time.Sleep(time.Millisecond * 500)
		goto restart
	}
}

func downloadLoop(chunks []string, filename [2]string, baseurl string, extension string) {
	globalMax += len(chunks) * 2
	mainFile := currentDir() + "\\Downloads\\" + filename[1]
	writeFile(mainFile+extension, "")
	// Asynchronously downloads each chunk of the Video and stitches it back together
	var wg sync.WaitGroup
	for i, u := range chunks {
		// Anti-Ratelimiting
		time.Sleep(time.Duration(clamp(6*len(chunks), 0, globalMaxTimeout) * int(time.Millisecond)))

		wg.Add(1)
		go func(url string, filename string, i int) {
			defer wg.Done()
			globalProgess += 1
			DownloadFile(url, filename, extension, i)
			globalProgess += 1
		}(baseurl+"&range="+u+"&hl=en", filename[1], i)
	}
	wg.Wait()
	for i := 0; i < len(chunks); i++ {
		curFile := mainFile + fmt.Sprint(i) + extension
		appendFile(mainFile+extension, readFile(curFile))
		os.Remove(curFile)
	}
}

func chunkedAsyncDownload(filename [2]string, videoBaseURL string, videoContentLength string, audioBaseURL string, audioContentLength string) {
	globalProgess += 1
	fmt.Println(colorString("[+] Started Download on: "+filename[0], "Light yellow", "None"))
	// Download Audio/Video
	var wg sync.WaitGroup
	for _, u := range [2]int{0, 1} {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i {
			case 0:
				downloadLoop(createChunkRanges(videoContentLength), filename, videoBaseURL, ".mp4")
			case 1:
				downloadLoop(createChunkRanges(audioContentLength), filename, audioBaseURL, ".m4a")
			}
		}(u)
	}
	wg.Wait()
	fmt.Println(colorString("[+] Finished Download on: "+filename[0], "Cyan", "None"))
	globalProgess += 1
	merge(filename)
}

func merge(filename [2]string) {
	vFilename := currentDir() + "\\Downloads\\" + filename[1] + ".mp4"
	aFilename := currentDir() + "\\Downloads\\" + filename[1] + ".m4a"

	// Starting FFMPEG to stitch Audio/Video together
	execWrapper(currentDir() + "\\ffmpeg.exe -i \"" + vFilename + "\" -i \"" + aFilename + "\" -c:v copy -c:a aac \"" + currentDir() + "\\Downloads\\" + removeInvalidFilenameChars(filename[0]) + ".mp4\"")
	os.Remove(vFilename)
	os.Remove(aFilename)
	fmt.Println(colorString("[+] Merged: "+filename[0], "Light green", "None"))
	merged += 1
}

func execWrapper(commandLine string) {
	// command to execute, may contain quotes, backslashes and spaces
	comSpec := os.Getenv("COMSPEC")
	if comSpec == "" {
		comSpec = os.Getenv("SystemRoot") + "\\System32\\cmd.exe"
	}

	childProcess := exec.Command(comSpec)
	childProcess.SysProcAttr = &syscall.SysProcAttr{CmdLine: "/C \"" + commandLine + "\""}

	// Then execute and read the output
	childProcess.Run()
}

func createChunkRanges(contentlen string) []string {
	// Creates a string that defines the section of the Video that should be downloaded
	chunks := []string{}

	for i := 0; i < strToInt(contentlen)/CHUNK_SIZE+1; i++ {
		chunks = append(chunks, fmt.Sprint(i*CHUNK_SIZE+i)+"-"+fmt.Sprint((1+i)*CHUNK_SIZE+i))
	}

	return chunks
}

// Helper Funcs

func colorString(inputString string, textcolor string, backgroundcolor string) string {
	textcolorMap := map[string]string{
		"None":         "30",
		"Red":          "31",
		"Green":        "32",
		"Yellow":       "33",
		"Blue":         "34",
		"Magenta":      "35",
		"Cyan":         "36",
		"Light gray":   "37",
		"Dark gray":    "90",
		"Light red":    "91",
		"Light green":  "92",
		"Light yellow": "93",
		"Light blue":   "94",
		"Light agenta": "95",
		"Light cyan":   "96",
		"White":        "97",
		"Bold":         "1",
		"Underline":    "4",
		"No underline": "24",
		"Negative":     "7",
		"Positive":     "27",
		"Default":      "0",
	}

	backgroundcolorMap := map[string]string{
		"Black":        ";40",
		"Red":          ";41",
		"Green":        ";42",
		"Yellow":       ";43",
		"Blue":         ";44",
		"Magenta":      ";45",
		"Cyan":         ";46",
		"Light gray":   ";47",
		"Dark gray":    ";100",
		"Light red":    ";101",
		"Light green":  ";102",
		"Light yellow": ";103",
		"Light blue":   ";104",
		"Light agenta": ";105",
		"Light cyan":   ";106",
		"White":        ";107",
		"None":         "",
	}

	return magic_backspace + "[" + textcolorMap[textcolor] + backgroundcolorMap[backgroundcolor] + "m" + inputString + "[40;0m" + strings.Replace(magic_space, " ", "", len(inputString)+len(textcolorMap[textcolor])+len(backgroundcolorMap[backgroundcolor])-3)
}

func sortVideoByVal(inputMap map[int]string) []int {
	qualities := []string{"hd1080", "hd720", "large", "medium", "small", "tiny"}

	inVideokeys := make([]int, 0, len(inputMap))
	for key := range inputMap {
		inVideokeys = append(inVideokeys, key)
	}

	outVideokeys := make([]int, 0, len(inputMap))
	for _, i := range inVideokeys {
		for _, q := range qualities {
			if strings.Contains(inputMap[i], q) {
				outVideokeys = append(outVideokeys, i)
			}
		}
	}
	sort.Ints(outVideokeys)

	return outVideokeys
}

func sortAudioByVal(inputMap map[int]string) int {
	Audiokeys := make([]int, 0, len(inputMap))
	for key := range inputMap {
		Audiokeys = append(Audiokeys, key)
	}

	return Audiokeys[0]
}

func removeInvalidFilenameChars(filename string) string {
	invalidChars := "\\/:*?\"<>|"
	for _, v := range invalidChars {
		filename = strings.ReplaceAll(filename, string(v), "")
	}
	return filename
}

func checkForPlaylist(slice []string) string {
	for _, v := range slice {
		if strings.Contains(v, "list") {
			return v
		}
	}
	return "false"
}

func strToInt(str string) int {
	outInt, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}
	return outInt
}

// FS Interaction Wrappers

func readFile(filepath string) string {
	data, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func writeFile(filepath string, data string) {
	err := os.WriteFile(filepath, []byte(data), 0644)
	if err != nil {
		panic(err)
	}
}

func appendFile(filepath string, data string) {
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.WriteString(data); err != nil {
		panic(err)
	}
}

// Math Funcs

func clamp(num int, nummin int, nummax int) int {
	return min(max(num, nummax), nummin)
}

func min(num int, min int) int {
	if num < min {
		return min
	}
	return num
}

func max(num int, max int) int {
	if num > max {
		return max
	}
	return num
}
