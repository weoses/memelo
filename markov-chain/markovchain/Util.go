package markovchain

import "github.com/chigopher/pathlib"

func FileToAlphabetByLetter(file string) ([]string, error) {
	alphabetFile := pathlib.NewPath(file)
	alphabetBytes, err := alphabetFile.ReadFile()
	if err != nil {
		return nil, err
	}

	alphabetSlice := []string{}
	for _, char := range string(alphabetBytes) {
		alphabetSlice = append(alphabetSlice, string(rune(char)))
	}
	return alphabetSlice, nil
}
