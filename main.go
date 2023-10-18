package main

import (
	wget "wget/lib"
)

func main() {
	url, output, rateLimit, logFile, downloadPath, mirror, shouldReturn := wget.GetArgs()
	if shouldReturn {
		return
	}

	if !mirror {
		wget.Domain = wget.GetDomain(url)
		fileName, outputDir := wget.GetFilenameAndDirFromURL(url)
		if output != "" {
			outputDir = output
		}
		wget.DownloadAndSaveResource(url, fileName, outputDir, logFile, rateLimit)
	} else {
		wget.MirrorWebsite(url, downloadPath, logFile, rateLimit)
	}
}
