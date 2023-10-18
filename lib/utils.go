package wget

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
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

	_rateLimit := flag.String("rate-limit", "0B", "Download speed limit in bytes per second")
	_output := flag.String("O", "", "Output file name")
	_downloadPath := flag.String("P", "", "Download file path")
	_mirror := flag.Bool("mirror", false, "Mirror site")
	_logFile := flag.Bool("B", false, "Log file")
	flag.Parse()
	output := *_output
	rateLimit, err := convertFileSizeToBytes(*_rateLimit)
	if err != nil {
		fmt.Println("🚩 Error:", err)
		return "", "", 0, false, "",false,  true
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
	unit := strings.ToLower(fileSize[len(fileSize)-2:])
	value := fileSize[:len(fileSize)-2]

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
	case "kb":
		return int(size * 1024), nil
	case "mb":
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
