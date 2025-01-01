package entity

type StorageMetaData struct {
	Id   string
	Name string
	Hash string
}

type OcrResult struct {
	ProcessorKey string
	Text         string
}

type OcrResultBulk struct {
	Texts *[]OcrResult
}

type ElasticImageMetaData struct {
	Storage *StorageMetaData
	Result  *OcrResultBulk
	Id      string
}

type StorageData struct {
	ImageBase64 *string
	FileName    *StorageMetaData
}
