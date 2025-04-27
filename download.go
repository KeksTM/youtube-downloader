package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

var globalMaxTimeout int = 500 // Max Timeout for Download Requests in MS
var CHUNK_SIZE int = 1603480   // Max Download Chunk Size for each part of the video buffer
var globalProgess int = 0
var globalETA string = ""
var globalMax int = 0 // Max possible "steps" able, used to compute eta and progress
var merged int = 0    // Merged Videos

func downloadVideos(urls []string) {
	var wg sync.WaitGroup
	for _, u := range urls {
		// Anti-Ratelimiting
		time.Sleep(time.Duration(clamp(8*len(urls), 0, globalMaxTimeout) * int(time.Millisecond)))

		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			// Call Youtubes API and fetch Video Data
			var JSONData YoutubeResponseData
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

			// Get highest quality audio and video
			audioIndex := JSONData.StreamingData.AdaptiveFormats[sortAudioByVal(audioData)]
			videoIndex := JSONData.StreamingData.AdaptiveFormats[sortVideoByVal(videoData)[0]]

			// Start Downloading
			strArr := [2]string{JSONData.VideoDetails.Title, JSONData.VideoDetails.VideoID}
			chunkedAsyncDownload(strArr, videoIndex.URL, videoIndex.ContentLength, audioIndex.URL, audioIndex.ContentLength)
		}(u)
	}
	wg.Wait()
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

func createChunkRanges(contentlen string) []string {
	chunks := []string{}
	for i := 0; i < strToInt(contentlen)/CHUNK_SIZE+1; i++ {
		chunks = append(chunks, fmt.Sprint(i*CHUNK_SIZE+i)+"-"+fmt.Sprint((1+i)*CHUNK_SIZE+i))
	}
	return chunks
}
