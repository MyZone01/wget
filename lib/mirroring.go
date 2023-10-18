package wget

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var Domain = ""
var Res []int
var Finish = ""
var TabUrl []string

const user_agent = "Golang Mirror v. 2.0"

// MirrorWebsite mirrors a website by recursively downloading all its pages.
//
// It takes a URL and an output directory as parameters and returns an error if the operation fails.
func MirrorWebsite(urlString, downloadPath string, logFile bool, rateLimit int) error {
	Domain = GetDomain(urlString)
	folderName := filepath.Join(".", Domain)

	// Create output directory
	err := os.MkdirAll(folderName, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}
	visited := make(map[string]bool)
	return mirrorPage(urlString, folderName, visited, logFile, rateLimit)
}

// mirrorPage mirrors a web page by downloading its resources and recursively mirroring linked pages.
//
// Parameters:
// - url: the URL of the web page to mirror.
// - outputDir: the directory where the mirrored resources will be saved.
// - visited: a map that keeps track of visited URLs to avoid duplicates.
//
// Returns:
// - error: an error if there was a problem while mirroring the page, otherwise nil.
func mirrorPage(url, outputDir string, visited map[string]bool, logFile bool, rateLimit int) error {
	if GetDomain(url) != Domain || visited[url] {
		return nil
	}

	visited[url] = true

	resp, err := launchRequest(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tokens := html.NewTokenizer(resp.Body)

	stop := false
	for {
		tokenType := tokens.Next()
		switch tokenType {
		case html.ErrorToken:
			stop = true // Finished parsing
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokens.Token()
			if token.Data == "a" || token.Data == "link" || token.Data == "img" || token.Data == "script" {
				for _, attr := range token.Attr {
					if attr.Key == "href" || attr.Key == "src" {
						link := attr.Val
						if !strings.HasPrefix(link, "http") {
							link = resolveRelativeURL(url, link)
						}

						// Download and save the linked resource
						if !strings.HasSuffix(link, ".html") {
							// Extract the file name from the URL
							fileName, outputDir := GetFilenameAndDirFromURL(link)
							_, err, _, _, _ := DownloadAndSaveResource(link, fileName, outputDir, logFile, rateLimit, false)
							if err != nil {
								fmt.Printf("Error downloading %s: %v\n", link, err)
							}
							continue
						}

						// Recursively mirror linked page
						mirrorPage(link, outputDir, visited, logFile, rateLimit)
					}
				}
			}
		default:
			// Other token types can be ignored
		}
		if stop {
			break
		}
	}

	fileName, _ := GetFilenameAndDirFromURL(url)
	_, err, _, _, _ = DownloadAndSaveResource(url, fileName, outputDir, logFile, rateLimit, false)
	if err != nil {
		fmt.Printf("Error downloading %s: %v\n", url, err)
	}
	return nil
}

func GetFilenameAndDirFromURL(link string) (string, string) {
	_subDir, fileName := path.Split(link)
	subDir := strings.Split(_subDir, "/")
	outputDir := strings.Join(subDir[2:len(subDir)-1], "/")
	if fileName == "" {
		fileName = "index.html"
	}
	return fileName, outputDir
}

// resolveRelativeURL resolves a relative URL against a base URL.
//
// It takes two parameters:
// - baseURL: The base URL to resolve against.
// - relativeURL: The relative URL to be resolved.
//
// The function returns a string, which is the resolved URL.
func resolveRelativeURL(baseURL, relativeURL string) string {
	base, _ := url.Parse(baseURL)
	rel, _ := url.Parse(relativeURL)
	return base.ResolveReference(rel).String()
}

// GetDomain returns the domain of a given URL.
//
// It takes a urlString string as a parameter and parses it to get the domain.
// If there is an error parsing the URL, it prints the error message and returns "unknown".
// Otherwise, it returns the domain.
func GetDomain(urlString string) string {
	// Parse URL to get the domain
	u, err := url.Parse(urlString)
	if err != nil {
		fmt.Printf("Error parsing URL: %v\n", err)
		return "unknown"
	}
	return u.Host
}

// DownloadAndSaveResource downloads a resource from a given URL and saves it to the specified output directory.
//
// Parameters:
// - url: the URL of the resource to be downloaded
// - outputDir: the directory where the downloaded resource will be saved
//
// Returns:
// - error: an error if any occurred during the download or saving process
func DownloadAndSaveResource(url, fileName, outputDir string, logFile bool, rateLimit int, changeDisplay bool) (*http.Response, error, []int, string, []string) {
	if GetDomain(url) != Domain {
		return nil, fmt.Errorf("domain mismatch: %s != %s", GetDomain(url), Domain), nil, "", nil
	}
	resp, err := launchRequest(url)
	if err != nil {
		return resp, err, nil, "", nil
	}
	defer resp.Body.Close()

	totalSize, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		buffer := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buffer)
			if err == io.EOF {
				totalSize += n
				break
			}
			if err != nil {
				return resp, err, nil, "", nil
			}
			totalSize += n
		}
		resp, err = launchRequest(url)
		if err != nil {
			return resp, err, nil, "", nil
		}
		defer resp.Body.Close()
		// return resp, fmt.Errorf("error converting total size: %s", err)
	}

	// Create the directory structure if it doesn't exist
	outputDir, err = expandTilde(outputDir)
	if err != nil {
		return resp, err, nil, "", nil
	}
	_, err = os.Stat(outputDir)
	if os.IsNotExist(err) {
		// The folder does not exist.
		err = os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			return resp, err, nil, "", nil
		}
	}

	// Create the local file and copy the resource into it
	filePath := path.Join(outputDir, fileName)
	localFile, err := os.Create(filePath)
	if err != nil {
		return resp, err, nil, "", nil
	}
	defer localFile.Close()

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
		return resp, fmt.Errorf("error %s", err), nil, "", nil
	}
	initString += fmt.Sprintf("Content size: %d\n", totalSize)
	initString += fmt.Sprintf("Saving file to: %s\n", filePath)

	if !logFile && !changeDisplay {
		fmt.Print(initString)
	}
	if changeDisplay {
		Res = append(Res, totalSize)
	}

	var downloadedSize int
	for {
		buffer := make([]byte, 1024)
		chunk, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			return resp, fmt.Errorf("error %s", err), nil, "", nil
		}

		_, err = localFile.Write(buffer[:chunk])
		if err != nil {
			return resp, fmt.Errorf("error %s", err), nil, "", nil
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

		if !logFile && !changeDisplay {
			fmt.Printf(
				"\r %s / %s [%s] %.2f%% - %s/s Time Remaining: %s - Time Elapsed: %s",
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
			if !logFile && !changeDisplay {
				fmt.Print("\n\n" + endString)
			} else if logFile {
				file, err := os.Create("wget-log")
				if err != nil {
					return resp, fmt.Errorf("error %s", err), nil, "", nil
				}
				file.WriteString(initString + endString)
			} else if changeDisplay {
				Finish += "finished " + fileName + "\n"
				TabUrl = append(TabUrl, url)
			}
			break
		}

		if rateLimit > 0 && bytesPerSec > int(rateLimit) {
			sleepDuration := time.Duration(float64(chunk)/(float64(rateLimit))*1024) * time.Millisecond
			time.Sleep(sleepDuration)
		}
	}
	return resp, err, Res, Finish, TabUrl
}

// launchRequest sends a GET request to the specified URL and returns the HTTP response and any error encountered.
//
// url: The URL to send the request to.
//
// Returns:
// - *http.Response: The HTTP response from the server.
// - error: Any error encountered during the request.
func launchRequest(url string) (*http.Response, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", user_agent)

	resp, err := client.Do(req)
	return resp, err
}
