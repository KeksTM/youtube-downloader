package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
)

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
