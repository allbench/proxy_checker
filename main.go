package main

/*
	Написано по мотивам https://habr.com/post/128477/
	по скорости работы разница не велика, но памяти потребляет несравнимо меньше.
*/

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"crysdd/checker"
)

func main() {
	flag.Parse()
	countThreads := countThreadsOrExit(flag.Arg(0))
	file := proxyFileOrExit(flag.Arg(1))
	defer file.Close()

	checker.Checker(countThreads, file)
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

func printError() {
	fmt.Println("Usage: " + os.Args[0] + " <count_threads>" + " <file_with_proxy>")
}
