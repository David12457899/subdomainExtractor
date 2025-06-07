package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

func main() {
	domain := flag.String("d", "", "Target domain to search for subdomains (e.g. ford.com)")
	url := flag.String("u", "", "URL to fetch and search")
	urlFile := flag.String("f", "", "File containing newline-separated URLs to fetch")
	output := flag.String("o", "", "File to write found subdomains (default: stdout)")
	maxThreads := flag.Int("t", 10, "Maximum number of concurrent threads")
	maxRPS := flag.Int("rps", 20, "Maximum number of HTTP requests per second")
	insecure := flag.Bool("i", true, "Skip TLS certificate verification (use with caution)")

	flag.Parse()

	if *domain == "" || (*url == "" && *urlFile == "") || (*url != "" && *urlFile != "") {
		fmt.Println("Usage: subdomainExtractor -d <domain> -u <url> / -f <file>")
		flag.Usage()
		os.Exit(1)
	}

	var urls []string
	if *urlFile != "" {
		file, err := os.Open(*urlFile)
		if err != nil {
			fmt.Printf("Error reading URL file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				urls = append(urls, line)
			}
		}
	} else {
		urls = append(urls, *url)
	}

	var writer io.Writer = os.Stdout
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}

	results := make(chan string)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, *maxThreads)
	rateLimiter := time.Tick(time.Second / time.Duration(*maxRPS))

	seen := make(map[string]bool)
	var seenLock sync.Mutex

	bar := progressbar.NewOptions(len(urls),
		progressbar.OptionSetDescription("Processing"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionClearOnFinish(),
	)

	for _, link := range urls {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()

			<-rateLimiter
			semaphore <- struct{}{}

			content, err := fetchContent(link, *insecure)
			<-semaphore

			_ = bar.Add(1)

			if err != nil {
				return
			}

			matches := findSubdomains(content, *domain)
			seenLock.Lock()
			for _, m := range matches {
				if !seen[m] {
					seen[m] = true
					results <- m
				}
			}
			seenLock.Unlock()
		}(link)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for sub := range results {
		fmt.Fprintln(writer, sub)
	}
}

func fetchContent(url string, insecure bool) (string, error) {
	client := &http.Client{}
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received status code %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func findSubdomains(content string, domain string) []string {
	pattern := `([a-zA-Z0-9_-]+\.)+` + regexp.QuoteMeta(domain)
	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllString(content, -1)

	unique := make(map[string]struct{})
	for _, match := range matches {
		unique[match] = struct{}{}
	}

	result := make([]string, 0, len(unique))
	for sub := range unique {
		result = append(result, sub)
	}
	return result
}
