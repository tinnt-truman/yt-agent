# Generating in Google Flow

_Last verified: 2026-10-11. Credit costs, limits, and the Flow interface
change; verify against Google's current docs before budgeting a project._

## What you need

- A Google account. Flow gives every account a small daily credit
  allowance (no rollover) that covers trying the service, including some
  video generation. A Pro subscription adds a monthly credit pool sized
  for real batch production. Allowances and availability vary by region;
  check Flow's credit guide for your account before planning.
- A desktop browser signed into that account. Flow is a web app; there is
  no supported API path in this skill.

## Project setup

- **One Flow project per video project.** Keeps generations, ingredients,
  and downloads from bleeding across jobs.
- **One Google account for the whole job.** Assets and credit counts follow
  the account. Switching accounts mid-project splits everything and wastes
  time re-uploading ingredients.
- **Know the credit allowance before planning.** A subscription adds
  a monthly credit pool; each clip spends from it. Count planned clips
  times cost-per-clip before you start, and hold back a 20-30%
  regeneration buffer. Some clips will fail QA; that is the normal case,
  not the exception.

## Generation modes

- **Ingredients-to-Video** is the workhorse for anything with continuity:
  it accepts reference images (character sheets, product photos, location
  plates) that steer identity and style. Attach them on every generation.
- **Text-to-video** is fine for one-off establishing shots with no
  continuity burden: skies, landscapes, abstract transitions.

## Settings per clip

- Aspect ratio to match delivery (16:9 for standard video).
- 8 seconds per clip: the web sweet spot for coherence. Longer asks drift
  more; shorter asks waste the credit.
- Fast tier for drafts and exploration; the Quality tier costs several
  times more per clip. Spend it only on clips that already passed QA on
  the cheap tier.

## Downloads

- Download the MP4 as soon as a clip is approved. Take the upscaled
  version when Flow offers it; it is the best quality you will get.
- Name each file with the clip's slug immediately
  (`03-market-morning.mp4`). Do not rely on the web app as your archive;
  projects and history are not permanent storage.
- Clips arrive with native audio (ambience, sometimes dialogue). If your
  pipeline does all sound in post, strip it during assembly. Decide once
  per project and apply it everywhere.
