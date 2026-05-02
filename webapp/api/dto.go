package api

import "github.com/weoses/memelo/webapp/service"

type MemeResponse struct {
	Id              string   `json:"id"`
	Caption         string   `json:"caption"`
	Type            string   `json:"type"`
	OcrResult       string   `json:"ocr_result"`
	OnScreenText    string   `json:"on_screen_text"`
	AudioTranscript string   `json:"audio_transcript"`
	AudioTrack      string   `json:"audio_track"`
	Tags            []string `json:"tags"`
	ThumbnailURL    string   `json:"thumbnail_url"`
	ThumbnailW      int32    `json:"thumbnail_w"`
	ThumbnailH      int32    `json:"thumbnail_h"`
	OriginalURL     string   `json:"original_url"`
	OriginalW       int32    `json:"original_w"`
	OriginalH       int32    `json:"original_h"`
	Status          string   `json:"status"`
	Edited          bool     `json:"edited"`
}

type PaginationResponse struct {
	Searcher     string   `json:"searcher"`
	SortingAfter []string `json:"sorting_after"`
}

type SearchResponse struct {
	Memes    []MemeResponse      `json:"memes"`
	NextPage *PaginationResponse `json:"next_page"`
}

type UploadUrlResponse struct {
	UploadUrl  string            `json:"upload_url"`
	FormFields map[string]string `json:"form_fields"`
	Token      string            `json:"token"`
}

func memeToResponse(m service.MemeResult) MemeResponse {
	return MemeResponse{
		Id:              m.Id,
		Caption:         m.Caption,
		Type:            m.Type,
		OcrResult:       m.OcrResult,
		OnScreenText:    m.OnScreenText,
		AudioTranscript: m.AudioTranscript,
		AudioTrack:      m.AudioTrack,
		Tags:            m.Tags,
		ThumbnailURL:    m.ThumbnailURL,
		ThumbnailW:      m.ThumbnailW,
		ThumbnailH:      m.ThumbnailH,
		OriginalURL:     m.OriginalURL,
		OriginalW:       m.OriginalW,
		OriginalH:       m.OriginalH,
		Status:          m.Status,
		Edited:          m.Edited,
	}
}

func searchToResponse(s *service.SearchResult) SearchResponse {
	memes := make([]MemeResponse, len(s.Memes))
	for i, m := range s.Memes {
		memes[i] = memeToResponse(m)
	}
	resp := SearchResponse{Memes: memes}
	if s.NextPage != nil {
		resp.NextPage = &PaginationResponse{
			Searcher:     s.NextPage.Searcher,
			SortingAfter: s.NextPage.SortingAfter,
		}
	}
	return resp
}

func uploadUrlToResponse(u *service.GetUploadUrlResult) UploadUrlResponse {
	return UploadUrlResponse{
		UploadUrl:  u.UploadUrl,
		FormFields: u.FormFields,
		Token:      u.Token,
	}
}
