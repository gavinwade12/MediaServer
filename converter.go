package main

import (
	"log"
	"time"

	"github.com/jeremytorres/rawparser"
)

var (
	workQueue    chan string
	nefParser    rawparser.RawParser
	nefParserKey string
)

const jpegQuality = 100

func init() {
	workQueue = make(chan string)
	nefParser, nefParserKey = rawparser.NewNefParser(true)
}

type worker struct {
	quit chan bool
}

func newWorker() *worker {
	return &worker{quit: make(chan bool)}
}

//TODO: use another mechanism to prevent a blocking send on the channel (buffered channel?)
func (w *worker) start() {
	log.Println("Worker started.")
	go func() {
		for {
			select {
			case file := <-workQueue:
				w.convert(file)
			case <-w.quit:
				log.Println("Worker quitting.")
				return
			}
		}
	}()
}

func (w *worker) stop() {
	w.quit <- true
}

func (w *worker) convert(file string) {
	log.Printf("Converting file: %s\n", file)
	start := time.Now()
	info := &rawparser.RawFileInfo{
		File:    file,
		DestDir: mediaDirectoryPath,
		Quality: jpegQuality,
	}

	_, err := nefParser.ProcessFile(info)
	if err != nil {
		log.Printf("Error processing file. File: %s, Error: %s\n", file, err.Error())
		return
	}
	log.Printf("Finished converting file. File: %s, Duration: %s\n", file, time.Since(start))
}
