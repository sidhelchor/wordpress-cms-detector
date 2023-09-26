package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"strconv"
)

// Customize your ASCII banner here
const customBanner = `
 __  __    __   __ +                                           
|  \/  |   \ \ / /        
| \  / |_ __\ V /   
| |\/| | '__|> < ______Language:go 
| |  | | |_ / . \   
|_|  |_|_(_)_/ \_\ Wp Filter V.1.2
 telegram id: @jackleet

                                                    
`

func main() {
	clearScreen()
	fmt.Print(customBanner)
	fmt.Println()

	sitelist := getInput("Enter the name of the sitelist file: ")

	sites, err := readSiteList(sitelist)
	if err != nil {
		fmt.Println("Error reading site list:", err)
		return
	}

	threadCount := getThreadCount() // Get the number of threads from user input
	threadCount = clamp(threadCount, 60, 500) // Ensure thread count is between 60 and 500

	var wg sync.WaitGroup
	var mu sync.Mutex // Mutex to protect the uniqueSites set

	uniqueSites := make(map[string]struct{}) // Set to store unique WordPress sites

	// Create a semaphore to limit the number of Goroutines
	semaphore := make(chan struct{}, threadCount)

	// Progress variables
	totalSites := len(sites)
	sitesScanned := 0

	for _, site := range sites {
		if !strings.HasPrefix(site, "http://") && !strings.HasPrefix(site, "https://") {
			site = "http://" + site
		}

		semaphore <- struct{}{} // Acquire a semaphore before starting a Goroutine
		wg.Add(1)
		go func(site string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the semaphore when done

			scan(site, &mu, uniqueSites)
			incrementProgress(&sitesScanned, totalSites)
		}(site)
	}

	wg.Wait()
}

func clearScreen() {
	fmt.Print("\033[H\033[2J") // ANSI escape codes to clear the screen
}

func getInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

func readSiteList(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var sites []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		site := scanner.Text()
		sites = append(sites, site)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func scan(site string, mu *sync.Mutex, uniqueSites map[string]struct{}) {
	resp, err := http.Get(site)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	bodyScanner := bufio.NewScanner(resp.Body)
	var responseBody string

	for bodyScanner.Scan() {
		responseBody += bodyScanner.Text()
	}

	// Check for common WordPress paths
	wordpressPaths := []string{
		"/wp-content/",
		"/wp-admin/",
		"/wp-includes/",
		"/wp-login.php",
		"/xmlrpc.php",
		"/readme.html",
		// Add more paths here if needed
	}

	isWordPress := false

	for _, path := range wordpressPaths {
		if strings.Contains(responseBody, path) {
			isWordPress = true
			break
		}
	}

	if isWordPress {
		mu.Lock() // Lock to protect the uniqueSites set
		defer mu.Unlock()

		if _, exists := uniqueSites[site]; !exists {
			uniqueSites[site] = struct{}{}
			fmt.Println("\nWordPress Detected --->", site)
			appendToFile("wordpress.txt", site)
		}
	} else {
		fmt.Println("\nNot WordPress --->", site)
	}
}

func appendToFile(filename, data string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(data + "\n"); err != nil {
		fmt.Println("Error:", err)
	}
}

func getThreadCount() int {
	threadCountStr := getInput("Enter the number of threads (default: 60, max: 500): ")
	threadCount, err := strconv.Atoi(threadCountStr)
	if err != nil {
		fmt.Println("Invalid input. Using default thread count (60).")
		return 60
	}
	return threadCount
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

func incrementProgress(count *int, total int) {
	*count++
	progress := float64(*count) / float64(total) * 100
	fmt.Printf("\rScanning Progress: %.2f%%", progress)
}
