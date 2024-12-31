package service

import (
	"fmt"
	"os"

	"gocv.io/x/gocv"
)

type Preprocessor interface {
	Preprocess(id string, file string) (string, error)
	GetName() string
}

type AdaptiveThresholdPreprocessor struct {
}

func (prep *AdaptiveThresholdPreprocessor) GetName() string {
	return "AdaptiveThresholdPreprocessor"
}

func (prep *AdaptiveThresholdPreprocessor) Preprocess(id string, file string) (string, error) {
	original := gocv.IMRead(file, gocv.IMReadGrayScale)

	if original.Empty() {
		return "", fmt.Errorf("failed to load file to opencv mat: file=%s", file)
	}

	threshHolded := gocv.NewMat()
	gocv.AdaptiveThreshold(
		original,
		&threshHolded,
		255,
		gocv.AdaptiveThresholdGaussian,
		gocv.ThresholdBinary,
		35,
		2)
	outputFile, err := os.CreateTemp("", id+".adaptiveThresh.*.jpg")
	if err != nil {
		return "", fmt.Errorf("adaptive threshold preprocessor failed, %w", err)
	}
	outputFileName := outputFile.Name()
	outputFile.Close()
	gocv.IMWrite(outputFileName, threshHolded)

	return outputFileName, nil
}

type EmptyPreprocessor struct{}

func (prep *EmptyPreprocessor) GetName() string {
	return "EmptyPreprocessor"
}

func (prep *EmptyPreprocessor) Preprocess(id string, file string) (string, error) {
	return file, nil
}
