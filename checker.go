package main

/*
Написано по мотивам https://habr.com/post/128477/
по скорости работы разница не велика, но памяти потребляет несравнимо меньше.
*/

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
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

func main() {

	// Установка переменных
	var (
		proxies   []string
		lastProxy uint64
	)
	flag.Parse()
	ipChecker := "http://myip.ru/index_small.php"
	numThreads, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		fmt.Println("Usage: " + os.Args[0] + " <count_threads>" + " <file_with_proxy>")
		os.Exit(1)
	}
	file, err := os.Open(flag.Arg(1))
	if err != nil {
		fmt.Println("Usage: " + os.Args[0] + " <count_threads>" + " <file_with_proxy>")
		os.Exit(1)
	}
	defer file.Close()
	os.Create("good_proxy.txt")
	// Узнаем свой текущий внешний ip
	res, _ := http.Get(ipChecker)
	data, _ := ioutil.ReadAll(res.Body)
	r, _ := regexp.Compile("([[:digit:]]+.[[:digit:]]+.[[:digit:]]+.[[:digit:]]+)")
	pr, _ := regexp.Compile("([[:digit:]]+.[[:digit:]]+.[[:digit:]]+.[[:digit:]]+:[[:digit:]]+)")

	myip := r.FindString(string(data))
	fmt.Println(myip)

	wg := sync.WaitGroup{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if pr.MatchString(scanner.Text()) {
			proxy := pr.FindString(scanner.Text())
			proxies = append(proxies, proxy)
		}
	}

	fmt.Println("Proxies found: ", len(proxies))
	// Создаём нужное количество потоков
	for i := 1; i < numThreads; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			// Бесконечный цикл
			for {
				// Берём следующий номер в списке
				seq := atomic.AddUint64(&lastProxy, 1)
				// Если список кончился, заканчиваем
				if int(seq) >= len(proxies) {
					fmt.Println("- Thread ", i, " done.")
					return
				}
				// Получаем следующую проксю из списка
				proxy := proxies[seq]
				// Стартует качалка
				proxyUrl, err := url.Parse("http://" + proxy)
				timeout := time.Duration(10 * time.Second)
				httpClient := &http.Client{
					Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
					Timeout:   timeout,
				}
				response, err := httpClient.Get("http://myip.ru/index_small.php")
				if err != nil { // отчёт
					fmt.Printf("Thread %02d; Seq. %03d; Proxy %20s; Status: ", i, seq, proxy)
					fmt.Println("Unable to connect.")
				} else {
					defer response.Body.Close()
					content, err := ioutil.ReadAll(response.Body)
					if err != nil {
						log.Fatal(err)
					} // отчёт
					fmt.Printf("Thread %02d; Seq. %03d; Proxy %20s; Status: ", i, seq, proxy)
					newip := r.FindString(string(content))
					if newip == myip {
						fmt.Println("Open.")
					} else if r.FindString(string(content)) != "" {
						fmt.Println("Anonimous:" + newip)
						f, err := os.OpenFile("good_proxy.txt", os.O_APPEND|os.O_WRONLY, 0644)
						if err != nil {
							log.Fatal(err)
						}
						defer f.Close()

						if _, err = f.WriteString(proxy + "\n"); err != nil {
							log.Fatal(err)
						}
						f.Sync()
					} else {
						fmt.Println("Not Working...")
					}
				}

			}
		}(i)
	}
	wg.Wait()
}
