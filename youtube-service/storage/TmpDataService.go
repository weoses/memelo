package storage

import (
	commonservice "github.com/weoses/memelo/common/service"
	commonstorage "github.com/weoses/memelo/common/storage"
	"github.com/weoses/memelo/youtube-service/conf"
)

type TmpDataServiceS3OperationsAdapter commonstorage.S3OperationsAdapter

func NewTmpDataServiceS3Adapter(cfg *conf.Config) (TmpDataServiceS3OperationsAdapter, error) {
	return commonstorage.NewS3OperationsAdapter(cfg.TempStorage)
}

func NewTmpDataService(adapter TmpDataServiceS3OperationsAdapter) (commonservice.TmpDataService, error) {
	return commonservice.NewTmpDataS3Service(adapter)
}
