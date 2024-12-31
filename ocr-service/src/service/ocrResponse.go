package service

type OcrResponse struct {
	Texts []OcrResponseItem
}

type OcrResponseItem struct {
	ProcessorKey string
	Text         string
}
