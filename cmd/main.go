/*
Homework
Параллельный загрузчик файлов
Загрузчик файлов,который может загружать файлы с нескольких URL-адресов.
Есть таймауты, лимиты на воркеров (5 sec)

Шаблоны: workerpool, fan-in, timeout
*/

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Site struct {
	URL string
}

type Result struct {
	URL             string
	workerIdMessage string
	File            []byte
}

func main() {

	jobs := make(chan Site, 3)
	results := make(chan Result, 3)
	urls := []string{
		"https://jsonplaceholder.typicode.com/posts/1",
		"https://jsonplaceholder.typicode.com/posts/2",
		"https://jsonplaceholder.typicode.com/posts/3",
		"https://jsonplaceholder.typicode.com/posts/4",
		"https://jsonplaceholder.typicode.com/posts/5",
		"https://jsonplaceholder.typicode.com/posts/6",
		"https://jsonplaceholder.typicode.com/posts/7",
		"https://google.com",
	}
	// читаем через пул с таймаутами сайты по списку урлов

	for w := 1; w < 4; w++ {

		go getFileWithTimer(3, w, jobs, results)
	}

	for _, url := range urls {
		jobs <- Site{URL: url}
	}
	close(jobs)

	// данные отдаем в каналы, которые собираем в один

	finalByteStream := merge(results)

	//организуем запись в куда-нибудь, лучше через интерфейс

	finalFile := FileWriting{}

	WriteByte(finalFile, finalByteStream)

}

func getFile(ctx context.Context, wId int, job <-chan Site, result chan<- Result) {

	for site := range job {
		resp, err := http.Get(site.URL)
		if err != nil {
			log.Println(err.Error())
		}

		defer resp.Body.Close()

		file, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err.Error())
		}

		result <- Result{
			URL:             site.URL,
			workerIdMessage: fmt.Sprintf("\nThe worker id is %d, and status_code %d", wId, resp.StatusCode),
			File:            file,
		}

	}

}

func getFileWithTimer(timeForRequestInSec int, wId int, job <-chan Site, result chan<- Result) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeForRequestInSec))
	defer cancel()
	getFile(ctx, wId, job, result)
}

// merging multiple channels and returning single channel
func merge(cs ...<-chan Result) <-chan []byte {
	var wg sync.WaitGroup
	out := make(chan []byte)

	send := func(c <-chan Result) {
		for n := range c {
			out <- n.File
		}
		wg.Done()
	}

	wg.Add(len(cs))

	for _, c := range cs {
		go send(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

type Writer interface {
	Write(<-chan []byte) error
}

func WriteByte(w Writer, bytes <-chan []byte) {
	err := w.Write(bytes)
	if err != nil {
		log.Fatalln("Failed writing")
	}
	fmt.Println("Well done")

}

type FileWriting struct {
}

func (f FileWriting) Write(bytes <-chan []byte) error {
	// open output file
	fo, err := os.Create("output.txt")
	if err != nil {
		return err
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// make a buffer to keep chunks that are read
	//buf := make([]byte, 1024)
	for buf := range bytes {
		if _, err := fo.Write(buf); err != nil {
			panic(err)
		}

	}
	return nil
}
