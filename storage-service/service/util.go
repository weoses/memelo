package service

import (
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

type SavedArtifactType string

const SavedOriginal SavedArtifactType = "original"
const SavedThumb SavedArtifactType = "thumbnail"

func storageMediaType(metadataType entity.MetadataType, savedItem SavedArtifactType) storage.MediaType {
	if metadataType == entity.VideoMetadataType {
		if savedItem == SavedOriginal {
			return storage.VideoMp4Original
		}
		if savedItem == SavedThumb {
			return storage.VideoMp4ThumbV1
		}
	}

	if metadataType == entity.ImageMetadataType {
		if savedItem == SavedOriginal {
			return storage.ImageJpegOriginal
		}
		if savedItem == SavedThumb {
			return storage.ImageJpegThumbV1
		}
	}

	panic("Unsupported metadata type")
}
