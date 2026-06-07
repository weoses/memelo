package utils

import (
	"context"
	"errors"
	"fmt"

	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/storage-service/service"
)

func PipelineAfterIDFromProto(p *v1.PipelinePagination) *service.PipelineAfterID {
	if p == nil {
		return nil
	}
	return &service.PipelineAfterID{
		SearcherName: p.Searcher,
		SortKey:      p.SortingAfter,
	}
}

func PipelineAfterIDToProto(a *service.PipelineAfterID) *v1.PipelinePagination {
	if a == nil {
		return nil
	}
	return &v1.PipelinePagination{
		Searcher:     a.SearcherName,
		SortingAfter: a.SortKey,
	}
}

func ToData(ctx context.Context, dataService commonservice.TmpDataService, media *v1.MediaDataDto) (temp.S3BackedData, bool, error) {
	if media.GetS3Path() != "" {
		result, err := dataService.WrapS3Path(ctx, media.GetS3Path())
		if err != nil {
			return nil, false, fmt.Errorf("failed to create backed temp by s3 path: %w", err)
		}
		return result, false, nil
	}
	if media.GetData() != nil {
		data, err := dataService.ByBytes(ctx, "application/octet-stream", media.GetData())
		if err != nil {
			return nil, false, fmt.Errorf("failed to get data from bytes: %w", err)
		}
		return data, true, nil
	}
	return nil, false, errors.New("media temp is empty")
}

func MetadataToMemeDto(urls *service.MetadataWithUrls) *v1.MemeDto {
	dto := &v1.MemeDto{
		Id:        urls.Metadata.ImageId.String(),
		OcrResult: urls.Metadata.Result,
		MediaOriginal: &v1.ImageDto{
			Url: urls.UrlOriginal,
		},
		ImageThumbnail: &v1.ImageDto{
			Url:         urls.UrlThumb,
			ImageWidth:  helper.Addr(int32(urls.Metadata.ThumbSize.Width)),
			ImageHeight: helper.Addr(int32(urls.Metadata.ThumbSize.Height)),
		},
		Tags:   urls.Metadata.Tags,
		Type:   string(urls.Metadata.Type),
		Edited: urls.Metadata.Edited,
	}

	if urls.Metadata.ImageSize != nil {
		dto.MediaOriginal.ImageWidth = helper.Addr(int32(urls.Metadata.ImageSize.Width))
		dto.MediaOriginal.ImageHeight = helper.Addr(int32(urls.Metadata.ImageSize.Height))
	}

	if urls.Metadata.ResultData != nil {
		dto.Caption = urls.Metadata.ResultData.Caption
		dto.OnScreenText = urls.Metadata.ResultData.OnScreenText
		dto.AudioTranscript = urls.Metadata.ResultData.AudioTranscript
		dto.AudioTrack = urls.Metadata.ResultData.AudioTrack
	}

	return dto
}
