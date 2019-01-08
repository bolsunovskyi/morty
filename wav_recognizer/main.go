package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/cryptix/wav"
	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
)

var (
	sampleRate          float64
	hmm                 string
	dict                string
	lm                  string
	jsgf                string
	keyPhrase           string
	debugLevel          int
	logFile             string
	nfft                int
	assistantConfigPath string
	wavPath             string
)

type Listener struct {
	inSpeech   bool
	uttStarted bool
	dec        *sphinx.Decoder
}

func init() {
	flag.Float64Var(&sampleRate, "sr", 16000, "sample rate")
	flag.StringVar(&hmm, "hmm", "", "directory containing acoustic model files")
	flag.StringVar(&dict, "dict", "", "main pronunciation dictionary (lexicon) input file")
	flag.StringVar(&lm, "lm", "", "word trigram language model input file")
	flag.StringVar(&jsgf, "jsgf", "", "grammar file")
	flag.StringVar(&keyPhrase, "keyphrase", "", "keyphrase")
	flag.StringVar(&logFile, "lf", "/dev/null", "log file")
	flag.IntVar(&debugLevel, "dl", 0, "debug level")
	flag.IntVar(&nfft, "nfft", 0, "nfft")
	flag.StringVar(&assistantConfigPath, "ac", "../config/config.yaml", "assistant config path")
	flag.StringVar(&wavPath, "wp", "../assets/kitchen.wav", "wav file path")
	flag.Parse()

	if hmm == "" || dict == "" || (lm == "" && jsgf == "") {
		log.Fatalln("hmm, dict and lm or jsgf must be specified")
	}
}

func main() {
	defer closer.Close()
	closer.Bind(func() {
		log.Println("Bye!")
	})

	cfg := sphinx.NewConfig(
		sphinx.HMMDirOption(hmm),
		sphinx.DictFileOption(dict),
		sphinx.SampleRateOption(float32(sampleRate)),
		sphinx.DebugOption(debugLevel),
		sphinx.LogFileOption(logFile),
	)

	if lm != "" {
		sphinx.LMFileOption(lm)(cfg)
	}

	if jsgf != "" {
		sphinx.UserOption("-jsgf", sphinx.String(jsgf))(cfg)
	}

	if keyPhrase != "" {
		sphinx.KeyphraseOption(keyPhrase)
	}

	if nfft != 0 {
		sphinx.UserOption("-nfft", nfft)(cfg)
	}

	log.Println("Loading CMU PhocketSphinx.")
	log.Println("This may take a while depending on the size of your model.")
	dec, err := sphinx.NewDecoder(cfg)
	if err != nil {
		closer.Fatalln(err)
	}
	closer.Bind(func() {
		dec.Destroy()
	})

	l := &Listener{
		dec: dec,
	}

	if !dec.StartUtt() {
		closer.Fatalln("[ERR] Sphinx failed to start utterance")
	}
	log.Println("Ready..")
	go l.readWav()
	closer.Hold()
}

func (l *Listener) readWav() {
	stat, err := os.Stat(wavPath)
	if err != nil {
		log.Fatalln(err)
	}

	fp, err := os.Open(wavPath)
	if err != nil {
		log.Fatalln(err)
	}

	reader, err := wav.NewReader(fp, stat.Size())
	if err != nil {
		log.Fatalln(err)
	}

	var input []int16
	for {
		rawSample, err := reader.ReadSample()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalln(err)
		}
		input = append(input, int16(rawSample))
	}

	_, ok := l.dec.ProcessRaw(input, false, false)
	if !ok {
		log.Println("status abort")
	}
	l.report()
	/*
		if l.dec.IsInSpeech() {
			l.inSpeech = true
			if !l.uttStarted {
				l.uttStarted = true
				log.Println("Listening..")
			}
		} else if l.uttStarted {
			l.dec.EndUtt()
			l.uttStarted = false
			l.report() // report results
			if !l.dec.StartUtt() {
				closer.Fatalln("[ERR] Sphinx failed to start utterance")
			}
		}*/
}

func (l *Listener) report() {
	hyp, _ := l.dec.Hypothesis()
	if len(hyp) > 0 {
		log.Printf("    > hypothesis: %s", hyp)
		return
	}

	log.Println("ah, nothing")
}
