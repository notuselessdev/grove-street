import os
import re
import time
import urllib.parse
import requests

BASE_URL = "https://jayuzumi.com/carl-johnson-soundboard"
GS_TO_HTTPS = "https://storage.googleapis.com/"
OUTPUT_DIR = "cj_sounds"
os.makedirs(OUTPUT_DIR, exist_ok=True)

def safe_filename(text: str) -> str:
    text = " ".join(text.strip().split())
    text = text.replace("/", "-")
    text = re.sub(r'[^A-Za-z0-9 _\-]', "", text)
    return text[:80] or "sound"

def fetch_page(url: str) -> str:
    r = requests.get(url, timeout=20)
    r.raise_for_status()
    return r.text

def extract_sounds(html: str):
    # Find the gsAudioUrls array in the page source
    m = re.search(r'gsAudioUrls\s*=\s*\[(.*?)\]', html, re.S)
    if not m:
        print("Could not find gsAudioUrls in page source.")
        return []

    # Extract individual quoted strings (can't use json.loads due to
    # single quotes inside double-quoted strings like AIN'T)
    urls = re.findall(r'"(gs://[^"]+)"', m.group(1))

    sounds = []
    for gs_url in urls:
        # Convert gs:// to https://storage.googleapis.com/
        if gs_url.startswith("gs://"):
            http_url = GS_TO_HTTPS + gs_url[len("gs://"):]
        else:
            http_url = gs_url

        # Extract quote from filename: "QUOTE - AUDIO FROM JAYUZUMI.COM.mp3"
        filename = gs_url.rsplit("/", 1)[-1]
        quote = re.sub(r'\s*-\s*AUDIO FROM JAYUZUMI\.COM\.mp3$', '', filename, flags=re.I)

        if quote:
            sounds.append((quote, http_url))

    return sounds

def download(quote: str, url: str, index: int):
    fname = f"{index:03d}_{safe_filename(quote)}.mp3"
    out_path = os.path.join(OUTPUT_DIR, fname)

    # URL-encode the path portion (spaces etc.)
    parsed = urllib.parse.urlparse(url)
    encoded_path = urllib.parse.quote(parsed.path)
    encoded_url = parsed._replace(path=encoded_path).geturl()

    print(f"Downloading: {quote} -> {fname}")
    try:
        r = requests.get(encoded_url, stream=True, timeout=30)
        if r.status_code != 200:
            print(f"  Skipped ({r.status_code})")
            return
        with open(out_path, "wb") as f:
            for chunk in r.iter_content(8192):
                if chunk:
                    f.write(chunk)
        time.sleep(0.2)
    except Exception as e:
        print(f"  Error for {quote}: {e}")

def main():
    html = fetch_page(BASE_URL)
    sounds = extract_sounds(html)
    print(f"Found {len(sounds)} sounds")
    for i, (quote, url) in enumerate(sounds, start=1):
        download(quote, url, i)

if __name__ == "__main__":
    main()
