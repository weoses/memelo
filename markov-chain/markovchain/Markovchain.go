package markovchain

import "bufio"

type MarkovChain interface {
	Train(text *bufio.Reader) error
	Confidence(text *bufio.Reader) (float64, error)
}

func NewMarkovChain(alphabet []string) *Model3Way {
	return &Model3Way{
		Alphabet: alphabet,
		Data:     map[string]map[string]map[string]float64{},
	}
}
