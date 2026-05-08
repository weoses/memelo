---
name: webapp-backend
description: Practices for backend part of webapp-serivce
---

Backend is located: ./webapp-service

When writing backend for webapp-service
- Use 3 (or more) layers in code - api, service, datasource(or connector to next service)
- Every file must be passed to storage-service ONLY using s3
- Any "upload" endpoint must accept only s3-links. There is get-upload-url for it in browser
- Service may be behind reverse-proxy