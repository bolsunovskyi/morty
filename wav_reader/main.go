package main

import (
	"flag"
	"log"
	"os"

	"github.com/cryptix/wav"
)

var (
	path string
)

func init() {
	flag.StringVar(&path, "p", "../assets/kitchen.wav", "wav file path")
	flag.Parse()
}

func main() {
	stat, err := os.Stat(path)
	if err != nil {
		log.Fatalln(err)
	}

	fp, err := os.Open(path)
	if err != nil {
		log.Fatalln(err)
	}

	reader, err := wav.NewReader(fp, stat.Size())
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Sample rate: %d\tChanneld: %d\n", reader.GetSampleRate(), reader.GetNumChannels())

	sample, err := reader.ReadRawSample()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Sample length: %d\n", len(sample))

	rawSample, err := reader.ReadSample()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Raw Sample: %d\n", rawSample)
}
