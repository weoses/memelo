package main

import (
	"os"

	echo "github.com/labstack/echo/v4"
	api "mine.local/ocr-gallery/ocr-server/api"
	service "mine.local/ocr-gallery/ocr-server/service"
)

func main() {

	e := echo.New()
	argPreprocessor := []service.Preprocessor{
		&service.AdaptiveThresholdPreprocessor{},
		&service.EmptyPreprocessor{},
	}
	argTessConnector := service.NewTesseractConnector()
	ocrService := service.NewOcrService(
		argPreprocessor,
		argTessConnector,
		os.Getenv("OCR_SERVICE_DEBUG") == "true",
	)
	api.RegisterHandlers(e, api.NewServerInterfaceImpl(ocrService))
	e.Logger.Fatal(e.Start(":7002"))
}
