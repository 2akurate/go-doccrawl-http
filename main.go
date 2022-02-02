package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	HOST        = "https://www.ordomedic.be/nl/zoek-een-arts/"
	BOUNDARY    = "----WebKitFormBoundary7MA4YWxkTrZu0gW"
	SOURCE_FILE = "C:\\Users\\pkgk02\\Documents\\1. My\\Projects\\concurrent-httprequest\\docnames.txt"
	BATCH_SIZE  = 15
)

func main() {
	lines := scanLines(SOURCE_FILE)
	names := getNames(lines)

	fmt.Printf("\nGetting addresses of %d doctors.\n", len(names))
	fmt.Printf("The requests will be made in batches of %d concurrent requests\n", BATCH_SIZE)

	executeInBatches(names, BATCH_SIZE)

	fmt.Println("Done")
}

func executeInBatches(names []string, batchSize int) {

	currentBatch := 1
	c := make(chan string, batchSize)

	for i, _ := range names {

		go func(name string) {
			getAddress(name, c)
		}(names[i])

		processBatch := (i+1)%batchSize == 0
		processRest := len(names)-(i) < batchSize

		if processBatch || processRest {

			startTime := time.Now()

			if processBatch {
				fmt.Printf("\nBatch %d waiting for responses\n", currentBatch)
				currentBatch++

				waitForBatch(batchSize, c)

			} else if processRest {
				batchSize = batchSize - 1
			}

			fmt.Printf("%v\n", time.Now().Sub(startTime))
		}
	}
}

func waitForBatch(batchSize int, c chan string) {
	for j := 0; j < batchSize; j++ {
		fmt.Println("response", j+1, <-c)
	}
}

func getAddress(fullname string, c chan string) {

	isSuccess := "System failed"
	defer func() { c <- isSuccess }()

	body := createGetAddressRequest(fullname)

	response, err := http.Post(HOST, "multipart/form-data; boundary="+BOUNDARY, body)
	if err != nil {
		log.Fatalf("Error posting multipart form")
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error reading response body")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))

	var address string

	if len(doc.Find(".address").Nodes) > 0 {
		isSuccess = "true"

		address = getAddressFromHTML(doc)

	} else {

		isSuccess = fmt.Sprintf("No address found for %s", fullname)
	}

	// Using address var so as not to get warning
	fmt.Printf(address)

	//writeToFile(data, fullname)
}

func getAddressFromHTML(doc *goquery.Document) string {

	var address string

	doc.Find(".address").Find("dd").Each(func(index int, item *goquery.Selection) {
		item.Each(func(i int, selection *goquery.Selection) {

			if selection.Text() != "" && selection.Text() != " " {
				if index == 0 {
					address += fmt.Sprintf("%s", selection.Text())
				} else {
					address += fmt.Sprintf(", %s", selection.Text())
				}
			}
		})
	})

	return address
}

func getNames(lines []string) []string {
	var names []string
	for _, l := range lines {
		index := bytes.IndexRune([]byte(l), '\t')

		if index <= 0 {
			break
		}
		col := l[:index]
		col = strings.ReplaceAll(col, `"`, "")
		col = strings.ReplaceAll(col, `,`, ``)
		names = append(names, col)
	}
	return names
}

func scanLines(filePath string) []string {
	file, err := os.Open(filePath)
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Return without headers
	return lines[1:]
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func createGetAddressRequest(fullname string) *bytes.Buffer {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.SetBoundary(BOUNDARY)
	writer.WriteField("search_name", fullname)
	writer.Close()
	//fmt.Println(body)
	return body
}

func writeToFile(data []byte, fullname string) {

	fname := strings.Replace(fullname, ` `, `-`, 1)
	fname = "output/" + fname + ".html"

	f, err := os.Create(fname)
	if err != nil {
		log.Fatalf("Problem creating file %v", err)
	}
	defer f.Close()
	f.Write(data)
}
