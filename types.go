package main

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
