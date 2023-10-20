package main

/*
	Написано по мотивам https://habr.com/post/128477/
	по скорости работы разница не велика, но памяти потребляет несравнимо меньше.
*/

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	proxies      []string
	lastProxy    uint64
	ipCheckerUrl = "http://myip.ru/index_small.php"
	myIp         string
)

func main() {
	flag.Parse()
	countThreads := countThreadsOrExit(flag.Arg(0))
	file := proxyFileOrExit(flag.Arg(1))
	defer file.Close()

	os.Create("good_proxy.txt")
	myIpRegexp := regexpForMyIp()
	myIp = getMyIp(ipCheckerUrl, myIpRegexp)
	fmt.Println("My IP is: " + myIp)

	loadProxyFromFile(file)
	fmt.Println("Proxies found: ", len(proxies))

	wg := sync.WaitGroup{}
	for i := 1; i < countThreads; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			for {
				sequence := atomic.AddUint64(&lastProxy, 1)
				if int(sequence) >= len(proxies) {
					fmt.Println("- Thread ", i, " done.")
					return
				}
				proxy := proxies[sequence]
				proxyUrl, err := url.Parse("http://" + proxy)
				if err != nil {
					fmt.Println("Broken...")
				} else {
					httpClient := &http.Client{
						Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
						Timeout:   time.Duration(10 * time.Second),
					}
					response, err := httpClient.Get(ipCheckerUrl)
					if err != nil {
						fmt.Printf("Thread %02d; Sequence %03d; Proxy %20s; Status: Unable to connect.\n", i, sequence, proxy)
					} else {
						defer response.Body.Close()
						content, err := io.ReadAll(response.Body)
						if err != nil {
							log.Fatal(err)
						}
						fmt.Printf("Thread %02d; Seq. %03d; Proxy %20s; Status: ", i, sequence, proxy)
						newIp := myIpRegexp.FindString(string(content))
						if newIp == myIp {
							fmt.Println("Open.")
						} else if myIpRegexp.FindString(string(content)) != "" {
							fmt.Println("Anonymous:" + newIp)
							saveGoodProxy(proxy)
						} else {
							fmt.Println("Not Working...")
						}
					}
				}

			}
		}(i)
	}
	wg.Wait()
}

func saveGoodProxy(proxy string) {
	f, err := os.OpenFile("good_proxy.txt", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err = f.WriteString(proxy + "\n"); err != nil {
		log.Fatal(err)
	}
	f.Sync()
}

func countThreadsOrExit(count string) int {
	countThreads, err := strconv.Atoi(count)
	if err != nil {
		printError()
		os.Exit(1)
	}
	return countThreads
}

func proxyFileOrExit(fileName string) *os.File {
	file, err := os.Open(fileName)
	if err != nil {
		printError()
		os.Exit(1)
	}
	return file
}

func regexpForMyIp() *regexp.Regexp {
	r, err := regexp.Compile("([[:digit:]]+.[[:digit:]]+.[[:digit:]]+.[[:digit:]]+)")
	if err != nil {
		log.Fatal(err)
	}
	return r
}

func regexpForProxy() *regexp.Regexp {
	r, err := regexp.Compile("([[:digit:]]+.[[:digit:]]+.[[:digit:]]+.[[:digit:]]+:[[:digit:]]+)")
	if err != nil {
		log.Fatal(err)
	}
	return r
}

func getMyIp(ipCheckerUrl string, myIpRegexp *regexp.Regexp) string {
	res, err := http.Get(ipCheckerUrl)
	if err != nil {
		fmt.Println("Unable to connect.\nПроверьте сетевое подключение.")
		os.Exit(1)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	return myIpRegexp.FindString(string(data))
}

func loadProxyFromFile(file *os.File) {
	scanner := bufio.NewScanner(file)
	proxyIpRegexp := regexpForProxy()
	for scanner.Scan() {
		if proxyIpRegexp.MatchString(scanner.Text()) {
			proxy := proxyIpRegexp.FindString(scanner.Text())
			proxies = append(proxies, proxy)
		}
	}
}

func printError() {
	fmt.Println("Usage: " + os.Args[0] + " <count_threads>" + " <file_with_proxy>")
}
