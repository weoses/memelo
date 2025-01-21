package markovchain

import (
	"bufio"
	"io"
	"slices"
	"strings"
)

type Model3Way struct {
	Alphabet        []string
	Data            map[string]map[string]map[string]float64
	EncounteredKeys int64
}

func (m *Model3Way) Train(text *bufio.Reader) error {

	var context []string
	readOneToContext(text, m.Alphabet, &context)
	readOneToContext(text, m.Alphabet, &context)

	counter := 0

	for {
		err := readOneToContext(text, m.Alphabet, &context)

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		value := m.getMatrixValue(context)
		value += 1
		m.setMatrixValue(context, value)

		counter += 1
		context = context[1:]
	}

	if counter == 0 {
		return io.EOF
	}

	m.EncounteredKeys = int64(counter)

	m.calcProbablities()

	return nil
}

func (m *Model3Way) Confidence(text *bufio.Reader) (float64, error) {
	var context []string
	readOneToContext(text, m.Alphabet, &context)
	readOneToContext(text, m.Alphabet, &context)

	counter := 0

	sums := float64(0)
	for {
		err := readOneToContext(text, m.Alphabet, &context)

		if err == io.EOF {
			break
		}

		if err != nil {
			return 0, err
		}

		value := m.getMatrixValue(context)
		sums = sums + value

		counter += 1
		context = context[1:]
	}

	if counter == 0 {
		return 0, io.EOF
	}

	return sums / float64(counter), nil
}

func readOneToContext(text *bufio.Reader, alpabet []string, context *[]string) error {
	for {
		oneRune, _, err := text.ReadRune()
		if err != nil || oneRune == 0 {
			return err
		}
		oneString := strings.ToLower(string(oneRune))
		if !slices.Contains(alpabet, oneString) {
			continue
		}
		*context = append(*context, oneString)
		return nil
	}
}

func (m *Model3Way) getMatrixValue(text []string) float64 {
	_, ok := m.Data[text[0]]
	if !ok {
		m.Data[text[0]] = map[string]map[string]float64{}
	}

	_, ok = m.Data[text[0]][text[1]]
	if !ok {
		m.Data[text[0]][text[1]] = map[string]float64{}
	}

	v1, ok := m.Data[text[0]][text[1]][text[2]]
	if !ok {
		m.Data[text[0]][text[1]][text[2]] = 0
		return 0
	}
	return v1
}

func (m *Model3Way) setMatrixValue(text []string, value float64) {
	_, ok := m.Data[text[0]]
	if !ok {
		m.Data[text[0]] = map[string]map[string]float64{}
	}

	_, ok = m.Data[text[0]][text[1]]
	if !ok {
		m.Data[text[0]][text[1]] = map[string]float64{}
	}

	m.Data[text[0]][text[1]][text[2]] = value
}

func (m *Model3Way) calcProbablities() {
	index := make([]string, 3)
	for _, k1 := range m.Alphabet {
		index[0] = k1
		for _, k2 := range m.Alphabet {
			index[1] = k2
			for _, k3 := range m.Alphabet {
				index[2] = k3

				value := m.getMatrixValue(index)
				value = value / float64(m.EncounteredKeys)
				m.setMatrixValue(index, value)
			}
		}
	}
}
