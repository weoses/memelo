output "service_url" {
  description = "URL of the deployed Cloud Run service"
  value       = google_cloud_run_v2_service.ffmpeg_service.uri
}

output "service_name" {
  description = "Name of the deployed Cloud Run service"
  value       = google_cloud_run_v2_service.ffmpeg_service.name
}
