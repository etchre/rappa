#!/usr/bin/env python3
import os
import re
import sys
from pathlib import Path
from urllib.parse import parse_qs, urlparse


VIDEO_ID_RE = re.compile(r"^[A-Za-z0-9_-]{11}$")


def main() -> int:
    load_env(Path(".env"))

    if len(sys.argv) != 2:
        print(f"usage: {Path(sys.argv[0]).name} <youtube-video-id-or-url>", file=sys.stderr)
        return 2

    original_video_id = video_id_from_identifier(sys.argv[1])
    if not original_video_id:
        print(f"could not find a YouTube video id in {sys.argv[1]!r}", file=sys.stderr)
        return 1

    print(resolve_video_id(original_video_id))
    return 0


def load_env(path: Path) -> None:
    if not path.exists():
        return

    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue

        name, value = line.split("=", 1)
        os.environ.setdefault(name.strip(), value.strip().strip('"').strip("'"))


def resolve_video_id(video_id: str) -> str:
    try:
        from ytmusicapi import YTMusic
    except ModuleNotFoundError:
        return video_id

    try:
        auth_file = os.environ.get("YTMUSIC_AUTH_FILE", "").strip()
        yt = YTMusic(auth_file) if auth_file else YTMusic()

        song = yt.get_song(video_id)
        details = song.get("videoDetails") or {}
        direct_id = details.get("videoId")
        if is_video_id(direct_id) and direct_id != video_id:
            return direct_id

        if is_playable_song(song):
            return video_id

        searched_id = search_replacement_id(yt, video_id, details)
        if searched_id:
            return searched_id

        return direct_id if is_video_id(direct_id) else video_id
    except Exception as exc:
        print(f"ytmusicapi resolution failed: {exc}", file=sys.stderr)
        return video_id


def is_playable_song(song: dict) -> bool:
    details = song.get("videoDetails") or {}
    playability = song.get("playabilityStatus") or {}
    return playability.get("status") == "OK" and details.get("isCrawlable") is not False


def search_replacement_id(yt, video_id: str, details: dict) -> str:
    title = details.get("title", "").strip()
    author = details.get("author", "").strip()
    if not title:
        return ""

    query = " ".join(part for part in [title, author] if part)
    results = yt.search(query, filter="songs", limit=5)

    for result in results:
        candidate = result.get("videoId")
        if candidate == video_id or not is_video_id(candidate):
            continue
        if title_matches(result, title) and artist_matches(result, author):
            return candidate

    for result in results:
        candidate = result.get("videoId")
        if candidate != video_id and is_video_id(candidate):
            return candidate

    return ""


def title_matches(result: dict, title: str) -> bool:
    return normalize_text(result.get("title", "")) == normalize_text(title)


def artist_matches(result: dict, author: str) -> bool:
    if not author:
        return True

    wanted = normalize_text(author)
    for artist in result.get("artists") or []:
        if normalize_text(artist.get("name", "")) == wanted:
            return True
    return False


def normalize_text(value: str) -> str:
    return re.sub(r"\s+", " ", value).strip().casefold()


def video_id_from_identifier(identifier: str) -> str:
    identifier = identifier.strip()
    if is_video_id(identifier):
        return identifier

    parsed = urlparse(identifier)
    host = parsed.hostname or ""
    if host == "youtu.be":
        candidate = parsed.path.strip("/")
        return candidate if is_video_id(candidate) else ""

    candidate = parse_qs(parsed.query).get("v", [""])[0]
    return candidate if is_video_id(candidate) else ""


def is_video_id(value: object) -> bool:
    return isinstance(value, str) and bool(VIDEO_ID_RE.fullmatch(value))


if __name__ == "__main__":
    raise SystemExit(main())
