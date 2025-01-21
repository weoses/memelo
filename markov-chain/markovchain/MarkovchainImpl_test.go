package markovchain

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"testing"

	"github.com/chigopher/pathlib"
	"github.com/stretchr/testify/assert"
)

func TestVerifierAccuary(t *testing.T) {

	alphabet, err := FileToAlphabetByLetter("../datasets/eng/alphabet.txt")
	assert.NoError(t, err, "Failed to read alphabet")

	chain := NewMarkovChain(alphabet)

	gpDataset := pathlib.NewPath("../datasets/eng/J. K. Rowling - Harry Potter 1 - Sorcerer's Stone.txt")
	gpDatasetReader, err := gpDataset.Open()
	assert.NoError(t, err, "Failed to read train alphabet")

	err = chain.Train(bufio.NewReader(gpDatasetReader))
	assert.NoError(t, err, "Failed to train chain")

	log.Println("==GOOD FILES==")
	minGood := float64(999999999999999999999999999999999999999)
	maxBad := float64(0)
	for i := 1; i <= 4; i++ {
		testFile := pathlib.NewPath(fmt.Sprintf("../datasets/test/eng-ok-%d.txt", i))
		testFileReader, err := testFile.Open()
		assert.NoError(t, err, "Failed to open test file")
		conf, _ := chain.Confidence(bufio.NewReader(testFileReader))
		log.Printf("- %f", conf)
		minGood = math.Min(conf, minGood)
	}

	log.Println("==BAD FILES==")
	for i := 1; i <= 4; i++ {
		testFile := pathlib.NewPath(fmt.Sprintf("../datasets/test/eng-bad-%d.txt", i))
		testFileReader, err := testFile.Open()
		assert.NoError(t, err, "Failed to open test file")
		conf, _ := chain.Confidence(bufio.NewReader(testFileReader))
		log.Printf("- %f", conf)
		maxBad = math.Max(conf, maxBad)
	}

	log.Printf("Min good: %f", minGood)
	log.Printf("Max bad:  %f", maxBad)

	log.Printf("Recommended treshold: %f", (minGood-maxBad)/2+maxBad)
	assert.Less(t, maxBad, minGood)
}
