import glob
import os
import subprocess
import uuid

from fastapi import FastAPI, Header, HTTPException, Query
from fastapi.responses import FileResponse

app = FastAPI(title="AnvuMusic Fetch API")

API_KEY = os.environ.get("MUSIC_API_KEY", "changeme")
COOKIES_PATH = os.environ.get(
    "COOKIES_PATH", "../internal/cookies/cookies.txt"
)
DOWNLOAD_DIR = os.environ.get("DOWNLOAD_DIR", "downloads")

os.makedirs(DOWNLOAD_DIR, exist_ok=True)


def _check_key(x_api_key: str | None):
    if x_api_key != API_KEY:
        raise HTTPException(401, "invalid api key")


@app.get("/health")
def health():
    return {"status": "ok"}


@app.get("/download")
def download(
    video_id: str = Query(...),
    video: bool = Query(False),
    x_api_key: str | None = Header(default=None),
):
    _check_key(x_api_key)

    if not video_id.isalnum() and "-" not in video_id and "_" not in video_id:
        raise HTTPException(400, "invalid video_id")

    url = f"https://www.youtube.com/watch?v={video_id}"
    job_id = uuid.uuid4().hex[:8]
    out_template = os.path.join(DOWNLOAD_DIR, f"{video_id}_{job_id}.%(ext)s")

    cmd = ["yt-dlp", "--no-playlist", "-o", out_template]
    if os.path.exists(COOKIES_PATH):
        cmd += ["--cookies", COOKIES_PATH]

    if video:
        cmd += [
            "-f",
            "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
            "--merge-output-format",
            "mp4",
        ]
    else:
        cmd += ["-x", "--audio-format", "opus"]

    cmd.append(url)

    result = subprocess.run(cmd, capture_output=True, text=True, timeout=180)
    if result.returncode != 0:
        raise HTTPException(502, f"yt-dlp failed: {result.stderr[-1500:]}")

    matches = glob.glob(os.path.join(DOWNLOAD_DIR, f"{video_id}_{job_id}.*"))
    if not matches:
        raise HTTPException(502, "no output file produced")

    path = matches[0]
    media_type = "video/mp4" if video else "audio/ogg"
    return FileResponse(path, media_type=media_type, filename=os.path.basename(path))
