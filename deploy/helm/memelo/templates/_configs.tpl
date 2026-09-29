{{/*
Each of these named templates renders one service's config.yaml body, ported
1:1 from deploy/roles/deploy/templates/<service>-config.yaml.j2. They're
included from configsecret.yaml with the root context (`.`), so they can
reach both plain values (.Values.xxx) and secrets (.Values.secrets.xxx)
merged in from the decrypted SOPS file at install time.
*/}}

{{- define "memelo.config.auth-service" -}}
log:
  Level: info

server:
  ListenAddress: :{{ (index .Values.services "auth-service").port }}

postgres:
  Dsn: "postgres://{{ .Values.secrets.postgresUser }}:{{ .Values.secrets.postgresPassword }}@{{ .Values.secrets.postgresHost }}:5432/{{ .Values.postgres.authDb }}?sslmode=disable"
{{- end }}

{{- define "memelo.config.ffmpeg-service" -}}
log:
  Level: info

server:
  ListenAddress: :{{ (index .Values.services "ffmpeg-service").port }}

temp-storage:
  Endpoint: {{ .Values.s3.endpoint }}
  AccessKey: {{ .Values.secrets.s3AccessKey }}
  SecretKey: {{ .Values.secrets.s3SecretKey }}
  Bucket: {{ .Values.s3.tempBucket }}
  Secure: {{ .Values.s3.secure }}

ffmpeg:
  FfmpegBinary: ffmpeg
  FfprobeBinary: ffprobe
  CpuLimit: {{ .Values.ffmpeg.cpuLimit }}
  ThreadsLimit: {{ .Values.ffmpeg.threadsLimit }}

job:
  MaxConcurrentJobs: 4
  JobTtlSeconds: 3600
{{- end }}

{{- define "memelo.config.storage-service" -}}
log:
  Level: info

server:
  ListenAddress: :{{ (index .Values.services "storage-service").port }}

extracting:
  embedding-dimensions: 1408
  video-slice-interval-sec: {{ .Values.extracting.videoSliceIntervalSec }}
  separate-audio: {{ .Values.extracting.separateAudio }}

search:
  SemanticDuplicateThreshold: 0.955
  PercentageDuplicatePartsThreshold: 0.7
  SemanticTextSearchThreshold: 0.5
  Fuzziness: "AUTO:4,8"

media-storage:
  Endpoint: {{ .Values.s3.endpoint }}
  AccessKey: {{ .Values.secrets.s3AccessKey }}
  SecretKey: {{ .Values.secrets.s3SecretKey }}
  Bucket: {{ .Values.s3.mediaBucket }}
  Secure: {{ .Values.s3.secure }}

temp-storage:
  Endpoint: {{ .Values.s3.endpoint }}
  AccessKey: {{ .Values.secrets.s3AccessKey }}
  SecretKey: {{ .Values.secrets.s3SecretKey }}
  Bucket: {{ .Values.s3.tempBucket }}
  Secure: {{ .Values.s3.secure }}

metadata-db:
  Elastic:
    Addresses:
    Username:
    Password:
    CloudID: {{ .Values.secrets.elasticCloudId }}
    ApiKey: {{ .Values.secrets.elasticApiKey }}
  Index: {{ .Values.elastic.metadataIndex }}

tag-db:
  Elastic:
    Addresses:
    Username:
    Password:
    CloudID: {{ .Values.secrets.elasticCloudId }}
    ApiKey: {{ .Values.secrets.elasticApiKey }}
  Index: {{ .Values.elastic.tagIndex }}

image-converter:
  ThumbSize: 360

ffmpeg:
  FfmpegBinary: ffmpeg
  FfprobeBinary: ffprobe
  CpuLimit: {{ .Values.ffmpeg.cpuLimit }}
  ThreadsLimit: {{ .Values.ffmpeg.threadsLimit }}

ffmpeg-service:
  Uri: "http://ffmpeg-service:{{ (index .Values.services "ffmpeg-service").port }}"
  PollIntervalMs: 500
  PollMaxWaitSec: 300

extractor-provider: {{ .Values.providers.extractor }}
embedder-provider: {{ .Values.providers.embedder }}

gemini-extractor:
  apikey: {{ .Values.secrets.geminiApiKey }}
  apiendpoint: {{ .Values.gemini.extractorApiEndpoint }}
  model-image: {{ .Values.gemini.extractorModelImage }}
  model-video: {{ .Values.gemini.extractorModelVideo }}
  model-audio: {{ .Values.gemini.extractorModelAudio }}
  image-extract-prompt: "Analyze this media and call the extract_metadata function with your findings. If you found nothing for parameter, leave parameter empty (except caption). Always preserve original language, and use correct alphabet (cyrillic, latin)"
  video-extract-prompt: "Analyze this media and call the extract_metadata function with your findings. If you found nothing for parameter, leave parameter empty (except caption). Always preserve original language, and use correct alphabet (cyrillic, latin)"
  audio-extract-prompt: "Analyze this media and call the extract_metadata function with your findings. If you found nothing for parameter, leave parameter empty (except caption). Always preserve original language, and use correct alphabet (cyrillic, latin)"
  output-tool-description: "Extract structured metadata from the media"
  output-tool-transcription-desc: "Audio transcription or speech-to-text content from the media. Preserve original language. Capture speech and music lyrics."
  output-tool-on-screen-text-desc: "Any text visible on screen (OCR). Preserve original language"
  output-tool-caption-desc: "A small caption summarizing the media content. Must not be more than 2-3 words"
  output-tool-audio-track-desc: "An audio track, song, that is at sound background. If it is not famous, or it to quiet, or you not sure, ignore it"
  combine-prompt: "Multiple extractions of video segments are provided below. Segments may overlap one another. Merge on-screen text and transcripts. Pick the most fitting caption. Synthesize them into a single portion of data and call extract_metadata."
  duplicate-prompt: "Are these two images is same, has same idea, same meaning, same characters, etc? Pass true to tool check_duplicate if images is same, or false otherwise."

gemini-embedding:
  apikey: {{ .Values.secrets.geminiApiKey }}
  apiendpoint: {{ .Values.gemini.embeddingApiEndpoint }}
  model: {{ .Values.gemini.embeddingModel }}

openrouter-extractor:
  apikey: {{ .Values.secrets.openrouterApiKey }}
  model-image: {{ .Values.openrouter.extractorModelImage }}
  model-video: {{ .Values.openrouter.extractorModelVideo }}
  model-audio: {{ .Values.openrouter.extractorModelAudio }}
  image-extract-prompt: "You are combined media/audio to text extractor. Analyze this media and call the extract_metadata function with your findings. If you found nothing for parameter, leave parameter empty (except caption). Always preserve original language, and use correct alphabet (cyrillic, latin). Always return only text/words/lyrics that actually was in media - do not try to write full fragment, return only data that is actually inside media"
  video-extract-prompt: "You are combined media/audio to text extractor. Analyze this media and call the extract_metadata function with your findings. If you found nothing for parameter, leave parameter empty (except caption). Always preserve original language, and use correct alphabet (cyrillic, latin). Always return only text/words/lyrics that actually was in media - do not try to write full fragment, return only data that is actually inside media"
  audio-extract-prompt: "You are audio to text extractor. Analyze this audio and call the extract_metadata function with your findings. If you found nothing for parameter, leave parameter empty (except caption). Always preserve original language, and use correct alphabet (cyrillic, latin). Always return only text/words/lyrics that actually was in media - do not try to write full fragment, return only data that is actually inside media. If something is too quiet, ignore it."
  output-tool-description: "Extract structured metadata from the media"
  output-tool-transcription-desc: "Audio transcription or speech-to-text content from the media. Preserve original language. Capture speech and music lyrics."
  output-tool-on-screen-text-desc: "Any text visible on screen (OCR). Preserve original language"
  output-tool-caption-desc: "A small caption summarizing the media content. Must not be more than 2-3 words"
  output-tool-audio-track-desc: "An audio track, song, that is at sound background. If it is not famous, or it to quiet, or you not sure, ignore it"
  combine-prompt: "Multiple extractions of video segments are provided below. Segments may overlap one another. Merge on-screen text and transcripts. Pick the most fitting caption. Synthesize them into a single portion of data and call extract_metadata."
  duplicate-prompt: "Are these two images is same, has same idea, same meaning, same characters, etc? Pass true to tool check_duplicate if images is same, or false otherwise."

openrouter-embedding:
  apikey: {{ .Values.secrets.openrouterApiKey }}
  model: {{ .Values.openrouter.embeddingModel }}
{{- end }}

{{- define "memelo.config.telegram-service" -}}
log:
  Level: info

server:
  ListenAddress: :{{ (index .Values.services "telegram-service").port }}

telegram:
  Token: {{ .Values.secrets.telegramToken }}
  Debug: false

inline:
  PageSize: 20

storage-service:
  Uri: "http://storage-service:{{ (index .Values.services "storage-service").port }}"

auth-service:
  Uri: "http://auth-service:{{ (index .Values.services "auth-service").port }}"

postgres:
  Dsn: "postgres://{{ .Values.secrets.postgresUser }}:{{ .Values.secrets.postgresPassword }}@{{ .Values.secrets.postgresHost }}:5432/{{ .Values.postgres.telegramDb }}?sslmode=disable"

user-account:
  StaticUuid: "00000000-0000-0000-0000-000000000000"

webhook:
  ExternalUrl: "https://{{ .Values.ingress.webhookDomain }}"

temp-storage:
  Endpoint: {{ .Values.s3.endpoint }}
  AccessKey: {{ .Values.secrets.s3AccessKey }}
  SecretKey: {{ .Values.secrets.s3SecretKey }}
  Bucket: {{ .Values.s3.tempBucket }}
  Secure: {{ .Values.s3.secure }}

permissions:
    Create:
      AllowedUserIds:
      {{- range .Values.permissions.createAllowed }}
        - {{ . }}
      {{- end }}

    Delete:
      AllowedUserIds:
      {{- range .Values.permissions.deleteAllowed }}
        - {{ . }}
      {{- end }}

    Recompute:
      AllowedUserIds:
      {{- range .Values.permissions.recomputeAllowed }}
        - {{ . }}
      {{- end }}

    Search:
      AllowedUserIds:
      {{- range .Values.permissions.searchAllowed }}
        - {{ . }}
      {{- end }}

youtube-service:
  Uri: "http://youtube-service:{{ (index .Values.services "youtube-service").port }}"
{{- end }}

{{- define "memelo.config.youtube-service" -}}
log:
  Level: info

server:
  ListenAddress: :{{ (index .Values.services "youtube-service").port }}

temp-storage:
  Endpoint: {{ .Values.s3.endpoint }}
  AccessKey: {{ .Values.secrets.s3AccessKey }}
  SecretKey: {{ .Values.secrets.s3SecretKey }}
  Bucket: {{ .Values.s3.tempBucket }}
  Secure: {{ .Values.s3.secure }}

youtube:
  MaxVideoSizeBytes: 524288000
  TempDir: "/tmp"
  MaxConcurrentDownloads: 4
  JobTtlSeconds: 3600
  ApiKey: {{ .Values.secrets.youtubeProviderApiKey }}
  ApiHost: p.savenow.to
  VideoFormat: "720"
  MaxDuration: 10
{{- end }}

{{- define "memelo.config.webapp-service" -}}
log:
  Level: info

server:
  ListenAddress: :{{ (index .Values.services "webapp-service").port }}

storage-service:
  Uri: "http://storage-service:{{ (index .Values.services "storage-service").port }}"

account:
  Id: "{{ .Values.secrets.webappAccountId }}"

temp-storage:
  Endpoint: {{ .Values.s3.endpoint }}
  AccessKey: {{ .Values.secrets.s3AccessKey }}
  SecretKey: {{ .Values.secrets.s3SecretKey }}
  Bucket: {{ .Values.s3.tempBucket }}
  Secure: {{ .Values.s3.secure }}

jwt:
  Secret: "{{ .Values.secrets.webappJwtSecret }}"

frontend:
  BaseUrl: {{ .Values.webapp.baseUrl }}
{{- end }}
