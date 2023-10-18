package wget

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetArgs returns the command line arguments required for the program.
//
// The function does the following:
// - Retrieves the number of command line arguments.
// - Prints the usage message if there are less than 2 arguments.
// - Returns empty strings, 0, false, "", false, true if there are less than 2 arguments.
// - Retrieves the last argument as the URL.
// - Prints an error message if the URL is empty.
// - Returns empty strings, 0, false, "", false, true if the URL is empty.
// - Parses the command line flags.
// - Retrieves the values of the flags and assigns them to variables.
// - Converts the rate limit to bytes per second.
// - Prints an error message if there is an error converting the rate limit.
// - Returns empty strings, 0, false, "", false, true if there is an error converting the rate limit.
// - Returns the URL, output file name, rate limit, log file, download file path, mirror site, and a false value.
//
// Return values:
// - string: URL of the website.
// - string: Output file name.
// - int: Download speed limit in bytes per second.
// - bool: Log file.
// - string: Download file path.
// - bool: Mirror site.
// - bool: Error flag.
func GetArgs() (string, string, int, bool, string, bool, bool) {
	nbArgs := len(os.Args)
	if nbArgs < 2 {
		fmt.Println("Usage: ./wget <url>")
		return "", "", 0, false, "", false, true
	}

	urlString := os.Args[nbArgs-1]
	if urlString == "" {
		fmt.Println("Please provide a valid URL.")
		return "", "", 0, false, "", false, true
	}

	_rateLimit := flag.String("rate-limit", "", "Download speed limit in bytes per second")
	_output := flag.String("O", "", "Output file name")
	_downloadPath := flag.String("P", ".", "Download file path")
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
	return urlString, output, rateLimit, logFile, downloadPath, mirror, false
}

func expandTilde(path string) (string, error) {
	if path[:2] == "~/" {
		currentUser, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(currentUser.HomeDir, path[2:]), nil
	}
	return path, nil
}

// CreateOutputFile creates an output file with the given parameters.
//
// It takes in three parameters:
// - output: the name of the output file (string).
// - url: the URL used to retrieve the resource name (string).
// - downloadPath: the path where the output file will be downloaded (string).
//
// The function returns three values:
// - output: the name of the output file (string).
// - file: a pointer to the created file (os.File).
// - bool: a boolean indicating if there was an error during the creation of the file (bool).
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

// FormatFileSize formats the given file size into a human-readable format.
//
// It takes an integer size as input, representing the size of the file in bytes.
// It returns a string representing the formatted file size.
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

// getResourceName retrieves the name of a resource from a given URL path.
//
// It takes a single parameter:
// - urlPath: a string representing the URL path from which to retrieve the resource name.
//
// It returns two values:
// - resourceName: a string representing the name of the resource.
// - err: an error, if any occurred during the process.
func getResourceName(urlPath string) (string, error) {
	parsedURL, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}

	resourceName := path.Base(parsedURL.Path)
	return resourceName, nil
}

// convertFileSizeToBytes converts a file size string to bytes.
//
// It takes the fileSize string as a parameter, which represents the size of a file.
// The function returns an integer value representing the file size in bytes and an error.
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

// downloadAndSaveResource downloads a resource from the given URL and saves it to the specified output file.
//
// Parameters:
// - url: the URL of the resource to download (string)
// - output: the name of the output file (string)
// - downloadPath: the path where the file should be saved (string)
// - logFile: whether to log the progress to a file (bool)
// - rateLimit: the maximum download rate in bytes per second (int)
//
// Returns:
// - true if there was an error during the download and save process (bool)
// - false otherwise (bool)
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
