package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Morty struct {
		Dict struct {
			Greeting []string          `yaml:"greeting"`
			Action   map[string]string `yaml:"action"`
			Target   map[string]string `yaml:"target"`
			Location map[string]string `yaml:"location"`
		} `yaml:"dict"`
	}
}

var (
	configPath        string
	dictGenPath       string
	grammarPath       string
	dictOutPath       string
	perlGeneratorPath string
	perlPath          string
)

const (
	grammarHeader = `#JSGF V1.0;

grammar morty;

`
	grammarFooter = `<expression> = <action> <target> [<location>];
public <query> = <greeting> | <expression>;

`
)

func init() {
	flag.StringVar(&configPath, "c", "config.yaml", "config path")
	flag.StringVar(&dictGenPath, "v", "../dict/vocabular.txt", "dict gen path")
	flag.StringVar(&grammarPath, "g", "../dict/grammar.jsgf", "grammar path")
	flag.StringVar(&dictOutPath, "o", "../dict/vocabular.dict", "dict out")
	flag.StringVar(&perlPath, "p", "perl", "perl path")
	flag.StringVar(&perlGeneratorPath, "pg", "", "perl generator path")

	flag.Parse()
}

func mapKeys(in map[string]string) (res []string) {
	for k := range in {
		res = append(res, k)
	}

	return
}

func main() {
	fp, err := os.Open(configPath)
	if err != nil {
		log.Fatalln(err)
	}

	var conf Config
	if err := yaml.NewDecoder(fp).Decode(&conf); err != nil {
		log.Fatalln(err)
	}

	voc, err := os.Create(dictGenPath)
	if err != nil {
		log.Fatalln(err)
	}

	voc.WriteString(strings.Join(conf.Morty.Dict.Greeting, "\n") + "\n")
	voc.WriteString(strings.Join(mapKeys(conf.Morty.Dict.Location), "\n") + "\n")
	voc.WriteString(strings.Join(mapKeys(conf.Morty.Dict.Target), "\n") + "\n")
	voc.WriteString(strings.Join(mapKeys(conf.Morty.Dict.Action), "\n"))
	if err := voc.Close(); err != nil {
		log.Fatalln(err)
	}

	gram, err := os.Create(grammarPath)
	if err != nil {
		log.Fatalln(err)
	}

	gram.WriteString(grammarHeader)
	gram.WriteString(fmt.Sprintf("<greeting> = (%s)\n", strings.Join(conf.Morty.Dict.Greeting, "|")))
	gram.WriteString(fmt.Sprintf("<action> = (%s)\n", strings.Join(mapKeys(conf.Morty.Dict.Action), "|")))
	gram.WriteString(fmt.Sprintf("<target> = (%s)\n", strings.Join(mapKeys(conf.Morty.Dict.Target), "|")))
	gram.WriteString(fmt.Sprintf("<location> = (%s)\n", strings.Join(mapKeys(conf.Morty.Dict.Location), "|")))
	gram.WriteString(grammarFooter)

	if err := gram.Close(); err != nil {
		log.Fatalln(err)
	}

	cmd := exec.Command("perl", perlGeneratorPath, dictGenPath, dictOutPath)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}

	log.Println("Done.")
}
