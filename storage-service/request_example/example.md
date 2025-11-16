Account id for all queries = `50c7503d-8301-4697-8f61-1cc31f1e9cac`

Search all images
```bash
curl http://localhost:7001/api/v1/accounts/50c7503d-8301-4697-8f61-1cc31f1e9cac/meme?Query=
```

Search images by query
```bash
curl http://localhost:7001/api/v1/accounts/50c7503d-8301-4697-8f61-1cc31f1e9cac/meme?Query=lorem
```

Search images by id
```bash
curl http://localhost:7001/api/v1/accounts/50c7503d-8301-4697-8f61-1cc31f1e9cac/meme?Query=a7317219-1462-436d-82be-9a051c26ebb7
```

Create image
```bash
curl http://localhost:7001/api/v1/accounts/50c7503d-8301-4697-8f61-1cc31f1e9cac/meme -X POST -H "Content-Type: application/json" -d @request.storage.create.test-image.json
```

Delete by id 
```bash
curl http://localhost:7001/api/v1/accounts/50c7503d-8301-4697-8f61-1cc31f1e9cac/meme/a7317219-1462-436d-82be-9a051c26ebb7 -X DELETE
```

Update ocr data
```bash
curl http://localhost:7001/api/v1/accounts/50c7503d-8301-4697-8f61-1cc31f1e9cac/update-ocr -X POST
```