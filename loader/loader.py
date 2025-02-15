import time
import requests
import sys
import pathlib
import base64

if len(sys.argv) < 3:
    print("2 args required")
    exit(1)

folder = sys.argv[1]
url = sys.argv[2]


path = pathlib.Path(folder)
for i in path.glob("*"):
    print(f"Load image {i}")

    mimeType = "image/jpeg"
    if i.name.lower().endswith("png"):
        mimeType = "image/png"
   
    payload = {
        "MimeType" : mimeType,
        "ImageBase64": base64.b64encode(i.read_bytes()).decode("utf-8")
    }
    response = requests.post(f"{url}/api/v1/meme", json=payload).json()
    print(response)
    time.sleep(1)