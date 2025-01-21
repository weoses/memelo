package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/chigopher/pathlib"
	"mine.local/markov-chain/markovchain"
)

func main() {

	command := os.Args[1]
	if command == "train" {
		Train()
	} else if command == "verify" {
		Verify()
	}

}

func Verify() {
	if len(os.Args) < 4 {
		panic("Not enought args: need: ModelFile InputFile")
	}

	modelArg := os.Args[2]
	inputArg := os.Args[3]

	modelFile := pathlib.NewPath(modelArg)
	inputFile := pathlib.NewPath(inputArg)

	model := markovchain.Model3Way{}
	modelBytes, err := modelFile.ReadFile()
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(modelBytes, &model)
	if err != nil {
		panic(err)
	}
	inputReader, err := inputFile.Open()
	if err != nil {
		panic(err)
	}
	defer inputReader.Close()

	confidence, err := model.Confidence(bufio.NewReader(inputReader))
	if err != nil {
		panic(err)
	}

	log.Printf("Confidence: %f", confidence)
}

func Train() {
	if len(os.Args) < 5 {
		panic("Not enought args: need: AlphabetFile DatasetFile OutputFile")
	}
	alphabetArg := os.Args[2]
	datasetArg := os.Args[3]
	outputArg := os.Args[4]
	datasetFile := pathlib.NewPath(datasetArg)
	outputFile := pathlib.NewPath(outputArg)

	datasetReader, err := datasetFile.Open()
	if err != nil {
		panic(err)
	}
	defer datasetReader.Close()

	if exists, err := outputFile.Exists(); exists {
		if err != nil {
			panic(err)
		}
		err = outputFile.Remove()
		if err != nil {
			panic(err)
		}
	}

	outputWriter, err := outputFile.Create()
	if err != nil {
		panic(err)
	}
	defer outputWriter.Close()

	alphabetSlice, err := markovchain.FileToAlphabetByLetter(alphabetArg)
	if err != nil {
		panic(err)
	}

	chain := markovchain.NewMarkovChain(alphabetSlice)
	err = chain.Train(bufio.NewReader(datasetReader))
	if err != nil {
		panic(err)
	}

	modelMarshalled, err := json.Marshal(chain)
	if err != nil {
		panic(err)
	}
	outputWriter.Write(modelMarshalled)

	log.Printf("Completed!")
}
