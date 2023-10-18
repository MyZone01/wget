package main

import (
	"fmt"
	wget "wget/lib"
)

func main() {
	url, output, rateLimit, logFile, downloadPath, mirror, shouldReturn := wget.GetArgs()
	if shouldReturn {
		return
	}

	if !mirror {
		wget.Domain = wget.GetDomain(url)
		fileName, _ := wget.GetFilenameAndDirFromURL(url)
		if output != "" {
			fileName = output
		}
		resp, err := wget.DownloadAndSaveResource(url, fileName, downloadPath, logFile, rateLimit)
		if err != nil {
			fmt.Printf("Error downloading %s: %v, %v\n", url, err, resp)
		}
	} else {
		wget.MirrorWebsite(url, downloadPath, logFile, rateLimit)
	}
}
