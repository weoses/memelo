package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log"
	"strings"

	"github.com/chigopher/pathlib"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/image-collector/conf"
)

type FilesystemImageStorageService struct {
	folder string
}

// Delete implements ImageStorageService.
func (f FilesystemImageStorageService) Delete(ctx context.Context, id string) {
	panic("unimplemented")
}

// Get implements ImageStorageService.
func (f FilesystemImageStorageService) Get(ctx context.Context, id string) (*string, error) {
	log.Printf("Read image from filesystem: id=%s", id)
	return readImage(f.folder, id)
}

// Save implements ImageStorageService.
func (f FilesystemImageStorageService) Save(ctx context.Context, storageData *string) (string, error) {
	imageId, _ := uuid.NewRandom()
	log.Printf("Save image to filesystem: id=%s", imageId)
	err := saveImage(f.folder, imageId.String(), storageData)
	return imageId.String(), err
}

func saveImage(folder string, imageId string, data *string) error {
	imagePath := pathlib.NewPath(folder).Join(imageId)
	imageFile, err := imagePath.Create()

	if err != nil {
		log.Printf("Save image to filesystem error: id=%s, path=%s error=%s",
			imageId, imagePath.String(), err.Error())
		return err
	}
	defer imageFile.Close()

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(*data))
	_, err = io.Copy(imageFile, decoder)
	return err
}

func readImage(folder string, imageId string) (*string, error) {
	file := pathlib.NewPath(folder).Join(imageId)
	data, err := file.ReadFile()
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBufferString("")
	encoder := base64.NewEncoder(base64.StdEncoding, buffer)
	defer encoder.Close()
	encoder.Write(data)
	encoder.Close()

	imageBase64Data := buffer.String()
	return &imageBase64Data, nil
}

func NewFilesystemImageStorageService(config *conf.ImageStorageConfig) ImageStorageService {
	return FilesystemImageStorageService{
		folder: config.Folder,
	}
}
