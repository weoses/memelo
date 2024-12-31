package service

import (
	"log"
	"strings"
	"sync"

	"github.com/otiai10/gosseract/v2"
)

type TesseractConnector interface {
	ProcessImage(file string) (string, error)
}

type TesseractConnectorImpl struct {
	mu     sync.Mutex
	client *gosseract.Client
}

func (tess *TesseractConnectorImpl) ProcessImage(image string) (string, error) {
	tess.mu.Lock()
	defer tess.mu.Unlock()

	log.Printf("Tesseract started: image=%s", image)
	defer log.Printf("Tesseract ended: image=%s", image)

	err := tess.client.SetImage(image)
	if err != nil {
		return "", err
	}

	text, err := tess.client.Text()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(text), nil
}

func NewTesseractConnector() TesseractConnector {
	client := gosseract.NewClient()
	client.SetLanguage("eng", "rus")
	return &TesseractConnectorImpl{
		client: client,
	}
}
