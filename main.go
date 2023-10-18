package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
	wget "wget/lib"
)

func main() {
	url, output, rateLimit, logFile, downloadPath, mirror, shouldReturn := wget.GetArgs()
	if shouldReturn {
		return
	}

	if !mirror {
		wget.DownloadAndSaveResource(url, output, logFile, rateLimit)
	} else {
		wget.MirrorWebsite(url, downloadPath, logFile, rateLimit)
	}
}

func downloadAndSaveResource(url string, output string, downloadPath string, logFile bool, rateLimit int) bool {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return true
	}
	defer resp.Body.Close()
	
	output, file, shouldReturn := wget.CreateOutputFile(output,  url, downloadPath)
	if shouldReturn {
		return true
	}
	defer file.Close()

	totalSize, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return true
	}

	const barWidth = 50
	progress := make([]rune, barWidth)
	startTime := time.Now()
	startTimeString := startTime.Format("2006-01-02 15:04:05")
	initString := ""
	endString := ""
	initString += fmt.Sprintf("Start at: %s\n", startTimeString)
	initString += "Sending request, awaiting response... "
	if resp.StatusCode == http.StatusOK {
		initString += "status 200 OK\n"
	} else {
		fmt.Println("🚩 Error:", resp.StatusCode)
		return true
	}
	initString += fmt.Sprintf("Content size: %d\n", totalSize)
	initString += fmt.Sprintf("Saving to: ./%s\n\n", output)

	if !logFile {
		fmt.Print(initString)
	}

	var downloadedSize int
	for {

		buffer := make([]byte, 1024)
		chunk, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Println("🚩 Error:", err)
			return true
		}

		_, err = file.Write(buffer[:chunk])
		if err != nil {
			fmt.Println("🚩 Error:", err)
			return true
		}

		downloadedSize += chunk

		progressLength := int(float64(downloadedSize) / float64(totalSize) * barWidth)
		for i := 0; i < barWidth; i++ {
			if i < progressLength {
				progress[i] = '='
			} else {
				progress[i] = ' '
			}
		}

		elapsedTime := time.Since(startTime)

		bytesPerSec := int(float64(downloadedSize) / elapsedTime.Seconds())
		remainingTime := time.Duration(float64(elapsedTime) / float64(downloadedSize) * float64(totalSize-downloadedSize))

		if !logFile {
			fmt.Printf(
				"\r%s / %s [%s] %.2f%% - %s/s Time Remaining: %s - Time Elapsed: %s",
				wget.FormatFileSize(downloadedSize),
				wget.FormatFileSize(totalSize),
				string(progress),
				float64(downloadedSize)/float64(totalSize)*100,
				wget.FormatFileSize(bytesPerSec),
				remainingTime.Truncate(time.Second).String(),
				elapsedTime.Truncate(time.Second).String(),
			)
		}

		if err == io.EOF || (downloadedSize == totalSize) {
			endTime := time.Now()
			endTimeString := endTime.Format("2006-01-02 15:04:05")
			endString += fmt.Sprintf("Download completed [%s]\n", url)
			endString += fmt.Sprintf("finished at: %s\n", endTimeString)
			if !logFile {
				fmt.Print("\n\n" + endString)
			} else {
				file, err := os.Create("wget-log")
				if err != nil {
					fmt.Println("🚩 Error:", err)
					return true
				}
				file.WriteString(initString + endString)
			}
			break
		}

		if rateLimit > 0 && bytesPerSec > int(rateLimit) {
			sleepDuration := time.Duration(float64(chunk)/(float64(rateLimit))*1024) * time.Millisecond
			time.Sleep(sleepDuration)
		}
	}
	return false
}
