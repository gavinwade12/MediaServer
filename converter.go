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
// This probably won't matter too much right now since this isn't being used much. We could
// just up the worker count for now, and address this problem when we implement batch uploads
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

	// TODO: The DestDir needs to be updated to the same structure mentioned in the TODO in
	// the upload handler. Obviously the file extension is always going to be JPG. I don't know
	// that we have the option to change the name of the file the 3rd party lib creates, but that
	// would be nice as well. It currently appends some weird _extracted+some+other+jibberish to
	// the end of the filename.
	info := &rawparser.RawFileInfo{
		File:    file,
		DestDir: configuration.MediaDirectoryPath,
		Quality: jpegQuality,
	}

	_, err := nefParser.ProcessFile(info)
	if err != nil {
		log.Printf("Error processing file. File: %s, Error: %s\n", file, err.Error())
		return
	}
	log.Printf("Finished converting file. File: %s, Duration: %s\n", file, time.Since(start))
}
