package main

import (
	"flag"
	"log"
	"unsafe"

	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"github.com/xlab/portaudio-go/portaudio"
)

const (
	samplesPerChannel = 512
	channels          = 1
	sampleFormat      = portaudio.PaInt16
)

var (
	sampleRate float64
	hmm        string
	dict       string
	lm         string
	jsgf       string
	keyPhrase  string
	debugLevel int
	logFile    string
	nfft       int
)

/*
./morty -hmm ~/zero_ru_cont_8k_v3/zero_ru.cd_cont_4000 -dict ./dict/vocabular.dict -jsgf ./dict/grammar.jsgf -lf /dev/stdout -nfft 2048
pocketsphinx_continuous -hmm ~/zero_ru_cont_8k_v3/zero_ru.cd_cont_4000 -jsgf ./dict/grammar.jsgf -dict ./dict/vocabular.dict -inmic yes -adcdev plughw:1

*/

func init() {
	flag.Float64Var(&sampleRate, "sr", 48000, "sample rate")
	flag.StringVar(&hmm, "hmm", "", "directory containing acoustic model files")
	flag.StringVar(&dict, "dict", "", "main pronunciation dictionary (lexicon) input file")
	flag.StringVar(&lm, "lm", "", "word trigram language model input file")
	flag.StringVar(&jsgf, "jsgf", "", "grammar file")
	flag.StringVar(&keyPhrase, "keyphrase", "", "keyphrase")
	flag.StringVar(&logFile, "lf", "/dev/null", "log file")
	flag.IntVar(&debugLevel, "dl", 0, "debug level")
	flag.IntVar(&nfft, "nfft", 0, "nfft")
	flag.Parse()

	if hmm == "" || dict == "" || (lm == "" && jsgf == "") {
		log.Fatalln("hmm, dict and lm or jsgf must be specified")
	}
}

func main() {
	appRun()
}

func appRun() {
	defer closer.Close()
	closer.Bind(func() {
		log.Println("Bye!")
	})
	if err := portaudio.Initialize(); paError(err) {
		log.Fatalln("PortAudio init error:", paErrorText(err))
	}
	closer.Bind(func() {
		if err := portaudio.Terminate(); paError(err) {
			log.Println("PortAudio term error:", paErrorText(err))
		}
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

	var stream *portaudio.Stream
	if err := portaudio.OpenDefaultStream(&stream, channels, 0, sampleFormat, sampleRate,
		samplesPerChannel, l.paCallback, nil); paError(err) {
		log.Fatalln("PortAudio error:", paErrorText(err))
	}
	closer.Bind(func() {
		if err := portaudio.CloseStream(stream); paError(err) {
			log.Println("[WARN] PortAudio error:", paErrorText(err))
		}
	})

	if err := portaudio.StartStream(stream); paError(err) {
		log.Fatalln("PortAudio error:", paErrorText(err))
	}
	closer.Bind(func() {
		if err := portaudio.StopStream(stream); paError(err) {
			log.Fatalln("[WARN] PortAudio error:", paErrorText(err))
		}
	})

	if !dec.StartUtt() {
		closer.Fatalln("[ERR] Sphinx failed to start utterance")
	}
	log.Println("Ready..")
	closer.Hold()
}

type Listener struct {
	inSpeech   bool
	uttStarted bool
	dec        *sphinx.Decoder
}

// paCallback: for simplicity reasons we process raw audio with sphinx in the this stream callback,
// never do that for any serious applications, use a buffered channel instead.
func (l *Listener) paCallback(input unsafe.Pointer, _ unsafe.Pointer, sampleCount uint,
	_ *portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags, _ unsafe.Pointer) int32 {

	const (
		statusContinue = int32(portaudio.PaContinue)
		statusAbort    = int32(portaudio.PaAbort)
	)

	in := (*(*[1 << 24]int16)(input))[:int(sampleCount)*channels]
	// ProcessRaw with disabled search because callback needs to be relatime
	_, ok := l.dec.ProcessRaw(in, true, false)
	// log.Printf("processed: %d frames, ok: %v", frames, ok)
	if !ok {
		return statusAbort
	}
	if l.dec.IsInSpeech() {
		l.inSpeech = true
		if !l.uttStarted {
			l.uttStarted = true
			log.Println("Listening..")
		}
	} else if l.uttStarted {
		// speech -> silence transition, time to start new utterance
		l.dec.EndUtt()
		l.uttStarted = false
		l.report() // report results
		if !l.dec.StartUtt() {
			closer.Fatalln("[ERR] Sphinx failed to start utterance")
		}
	}
	return statusContinue
}

func (l *Listener) report() {
	hyp, _ := l.dec.Hypothesis()
	if len(hyp) > 0 {
		log.Printf("    > hypothesis: %s", hyp)
		return
	}
	log.Println("ah, nothing")
}

func paError(err portaudio.Error) bool {
	return portaudio.ErrorCode(err) != portaudio.PaNoError
}

func paErrorText(err portaudio.Error) string {
	return portaudio.GetErrorText(err)
}
