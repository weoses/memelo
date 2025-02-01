package entity

type Image struct {
	MimeType string
	Data     *[]byte
}

type ImageSizes struct {
	Width  int
	Height int
}
