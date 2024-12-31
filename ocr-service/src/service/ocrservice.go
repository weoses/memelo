package service

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"gocv.io/x/gocv"
)

type OcrService interface {
	ProcessImage(Name string, img io.Reader) (*OcrResponse, error)
}

type OcrServiceImpl struct {
	processors    []Preprocessor
	tessConnector TesseractConnector
	debug         bool
}

func NewOcrService(
	processors []Preprocessor,
	tessConnector TesseractConnector,
	debug bool) OcrService {
	return &OcrServiceImpl{
		processors:    processors,
		tessConnector: tessConnector,
		debug:         debug,
	}
}

func (ocr *OcrServiceImpl) ProcessImage(Name string, originalReader io.Reader) (*OcrResponse, error) {

	savedOriginalFile, err := saveOriginalFile(Name, originalReader)
	if err != nil {
		return nil, errors.Join(errors.New("failed to save incoming file"), err)
	}

	if !ocr.debug {
		defer os.Remove(savedOriginalFile)
	}

	err = checkImage(savedOriginalFile)

	if err != nil {
		return nil, errors.Join(errors.New("failed to check image format"), err)
	}

	response := OcrResponse{}
	for _, processor := range ocr.processors {
		log.Printf("Run ocr file=%s preprocessor=%s\n", savedOriginalFile, processor.GetName())
		ocrData, err := ocr.runForPreprocessor(processor, Name, savedOriginalFile)
		if err != nil {
			log.Printf("Failed to do ocr with preprocessor: proeprocessor=%s error=%s", processor.GetName(), err.Error())
			continue
		}
		log.Printf("Do-ocr success: preprocessor=%s", processor.GetName())

		response.Texts = append(
			response.Texts,
			*ocrData,
		)
	}

	return &response, nil
}

func (ocr *OcrServiceImpl) runForPreprocessor(processor Preprocessor, Name string, originalFile string) (*OcrResponseItem, error) {
	log.Printf("Invoke preprocessor file=%s preprocessor=%s\n", originalFile, processor.GetName())
	preprocessedFile, err := processor.Preprocess(Name, originalFile)
	if err != nil {
		log.Printf("Preprocessor fails: file=%s preprocessor=%s error=%s",
			preprocessedFile, processor.GetName(), err.Error())
		return nil, err
	}

	log.Printf("Preprocessor compeles: file=%s preprocessor=%s", preprocessedFile, processor.GetName())
	if !ocr.debug {
		log.Printf("Preprocessed defer remove: file=%s ", preprocessedFile)
		defer os.Remove(preprocessedFile)
	}

	log.Printf("Invoke tesseract file=%s", preprocessedFile)
	text, err := ocr.tessConnector.ProcessImage(preprocessedFile)
	if err != nil {
		log.Printf("Tesseract fails: file=%s error=%s", preprocessedFile, err.Error())
		return nil, err
	}
	log.Printf("Tesseract completes: file=%s textlen=%d", preprocessedFile, len(text))

	normalizedText := AlphabetFix(text)

	return &OcrResponseItem{
		ProcessorKey: processor.GetName(),
		Text:         normalizedText,
	}, nil
}

func saveOriginalFile(id string, originalReader io.Reader) (string, error) {
	originalFile, err := os.CreateTemp("", id+".original.*.jpg")
	if err != nil {
		return "", err
	}
	defer originalFile.Close()

	written, err := io.Copy(originalFile, originalReader)
	if err != nil {
		return "", err
	}

	log.Printf("Original file created: Name=%s File=%s written=%d", id, originalFile.Name(), written)

	return originalFile.Name(), nil
}

func checkImage(path string) error {
	original := gocv.IMRead(path, gocv.IMReadGrayScale)

	if original.Empty() {
		return fmt.Errorf("failed to load file to opencv mat: file=%s", path)
	}
	return nil
}
