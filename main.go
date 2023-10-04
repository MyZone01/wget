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
	url, output, rateLimit, logFile, downloadPath, shouldReturn := wget.GetArgs()
	if shouldReturn {
		return
	}

	// Perform the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return
	}
	defer resp.Body.Close()

	// Create the output file
	output, file, shouldReturn := wget.CreateOutputFile(output, err, url, downloadPath)
	if shouldReturn {
		return
	}
	defer file.Close()

	// Get the total size of the file
	totalSize, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return
	}

	// Initialize the progress bar and print 
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
		return
	}
	initString += fmt.Sprintf("Content size: %d\n", totalSize)
	initString += fmt.Sprintf("Saving to: ./%s\n\n", output)

	if !logFile {
		fmt.Print(initString)
	}

	// Start reading and writing the file
	var downloadedSize int
	for {
		// Read a chunk from the response body
		buffer := make([]byte, 1024)
		chunk, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Println("🚩 Error:", err)
			return
		}

		// Write the chunk to the output file
		_, err = file.Write(buffer[:chunk])
		if err != nil {
			fmt.Println("🚩 Error:", err)
			return
		}

		// Update the downloaded size
		downloadedSize += chunk

		// Calculate the progress
		progressLength := int(float64(downloadedSize) / float64(totalSize) * barWidth)
		for i := 0; i < barWidth; i++ {
			if i < progressLength {
				progress[i] = '='
			} else {
				progress[i] = ' '
			}
		}

		// Calculate the elapsed time
		elapsedTime := time.Since(startTime)

		// Calculate the estimated time remaining
		bytesPerSec := int(float64(downloadedSize) / elapsedTime.Seconds())
		remainingTime := time.Duration(float64(elapsedTime) / float64(downloadedSize) * float64(totalSize-downloadedSize))

		// Print the progress bar and stats
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

		// Break the loop when the download is complete
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
					return
				}
				file.WriteString(initString + endString)
			}
			break
		}

		// Check if speed limit is specified
		if rateLimit > 0 && bytesPerSec > int(rateLimit) {
			sleepDuration := time.Duration(float64(chunk)/(float64(rateLimit))*1024) * time.Millisecond
			time.Sleep(sleepDuration)
		}
	}
}
