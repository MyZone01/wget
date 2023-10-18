package wget

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func GetArgs() (string, string, int, bool, string, bool, bool) {
	nbArgs := len(os.Args)
	if nbArgs < 2 {
		fmt.Println("Usage: go run main.go <url>")
		return "", "", 0, false, "", false, true
	}

	url := os.Args[nbArgs-1]
	if url == "" {
		fmt.Println("Please provide a valid URL.")
		return "", "", 0, false, "", false, true
	}

	_rateLimit := flag.String("rate-limit", "", "Download speed limit in bytes per second")
	_output := flag.String("O", "", "Output file name")
	_downloadPath := flag.String("P", "", "Download file path")
	_mirror := flag.Bool("mirror", false, "Mirror site")
	_logFile := flag.Bool("B", false, "Log file")
	flag.Parse()
	output := *_output
	rateLimit, err := convertFileSizeToBytes(*_rateLimit)
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return "", "", 0, false, "", false, true
	}

	logFile := *_logFile
	downloadPath := *_downloadPath
	mirror := *_mirror
	return url, output, rateLimit, logFile, downloadPath, mirror, false
}

func CreateOutputFile(output string, url string, downloadPath string) (string, *os.File, bool) {
	if output == "" {
		_output, err := getResourceName(url)
		if err != nil {
			fmt.Println("🚩 Error:", err)
			return "", nil, true
		}
		output = _output
	}
	if downloadPath != "" {
		output = downloadPath + "/" + output
	}
	file, err := os.Create(output)
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return "", nil, true
	}

	return output, file, false
}

func FormatFileSize(size int) string {
	const (
		KiB = 1024
		MiB = KiB * 1024
		GiB = MiB * 1024
	)

	switch {
	case size >= GiB:
		return fmt.Sprintf("%.2f GiB", float64(size)/float64(GiB))
	case size >= MiB:
		return fmt.Sprintf("%.2f MiB", float64(size)/float64(MiB))
	case size >= KiB:
		return fmt.Sprintf("%.2f KiB", float64(size)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func getResourceName(urlPath string) (string, error) {
	parsedURL, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}

	resourceName := path.Base(parsedURL.Path)
	return resourceName, nil
}

func convertFileSizeToBytes(fileSize string) (int, error) {
	if fileSize == "" {
		return 0, nil
	}

	unit := strings.ToLower(fileSize[len(fileSize)-1:])
	value := fileSize[:len(fileSize)-1]

	size, err := strconv.ParseFloat(value, 64)
	if err != nil {
		if unit[1] == 'b' {
			value = fileSize[:len(fileSize)-1]
			size, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return 0, err
			}
			return int(size), nil
		}
		return 0, err
	}

	switch unit {
	case "k":
		return int(size * 1024), nil
	case "m":
		return int(size * 1024 * 1024), nil
	case "gb":
		return int(size * 1024 * 1024 * 1024), nil
	default:
		if unit[1] == 'b' {
			value = fileSize[:len(fileSize)-1]
			size, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return 0, err
			}
			return int(size), nil
		}
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}
}

func downloadAndSaveResource(url string, output string, downloadPath string, logFile bool, rateLimit int) bool {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return true
	}
	defer resp.Body.Close()

	output, file, shouldReturn := CreateOutputFile(output, url, downloadPath)
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
				FormatFileSize(downloadedSize),
				FormatFileSize(totalSize),
				string(progress),
				float64(downloadedSize)/float64(totalSize)*100,
				FormatFileSize(bytesPerSec),
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
