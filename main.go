package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	wget "wget/lib"
)

func main() {
	url, output, rateLimit, logFile, downloadPath, mirror, shouldReturn, UrlFile, reject, exclude := wget.GetArgs()
	fmt.Println(reject)
	fmt.Println(exclude)
	var lines []string
	changeDisplay := false
	if UrlFile != "" {
		changeDisplay = true
		readFile, err := os.Open(UrlFile)
		if err != nil {
			log.Fatal(err)
		}
		fileScanner := bufio.NewScanner(readFile)
		fileScanner.Split(bufio.ScanLines)

		for fileScanner.Scan() {
			lines = append(lines, fileScanner.Text())
		}
		readFile.Close()
	} else {
		lines = append(lines, url)
	}

	if shouldReturn {
		return
	}

	var res []int
	var finish string
	var tabUrl []string
	if !mirror {
		for i := 0; i < len(lines); i++ {
			url = lines[i]
			wget.Domain = wget.GetDomain(url)
			fileName, _ := wget.GetFilenameAndDirFromURL(url)
			if output != "" {
				fileName = output
			}
			resp, err, a, b, c := wget.DownloadAndSaveResource(url, fileName, downloadPath, logFile, rateLimit, changeDisplay)
			if err != nil {
				fmt.Printf("Error downloading %s: %v, %v\n", url, err, resp)
			}
			res, finish, tabUrl = a, b, c
		}
		if changeDisplay {
			fmt.Print("content size: ")
			fmt.Println(res)
			fmt.Println(finish)
			fmt.Println(tabUrl)
		}

	} else {
		wget.MirrorWebsite(url, downloadPath, logFile, rateLimit)
	}
}
