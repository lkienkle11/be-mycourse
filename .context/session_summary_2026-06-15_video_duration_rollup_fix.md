# Session summary — VIDEO duration rollup fix (BE)

**Date:** 2026-06-15  
**Bug:** Section/lesson total showed only TEXT `2p`; VIDEO sub-lesson `estimated_duration_ms` was 0 despite Bunny video ~190s.

## Root cause

`batchMediaDurationMs` only read `media_files.duration` column. Dev video row had `duration = 0` and `metadata_json.duration_seconds = 0` (webhook had not persisted length), while Bunny API `length = 190`.

## GitNexus

- `impact(batchMediaDurationMs)` — CRITICAL blast (loadOutline + sub-lesson paths); behavior fix only, no API contract break.

## Fix

- `mediaDurationSecondsFromStored` — column → metadata (`duration_seconds`, `duration`, `length`)
- `batchMediaDurationMs` — Bunny Stream `GetBunnyVideoByID` fallback when `bunny_video_id` set and local resolve is 0

## Verified

```
SECTION ms= 310000  (190s video + 120s text)
VIDEO sub ms= 190000
TEXT sub ms= 120000
```

## Docs

- `docs/modules/course.md`, `docs/reusable-assets.md`
