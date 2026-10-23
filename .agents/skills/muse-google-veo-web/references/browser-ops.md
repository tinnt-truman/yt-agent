# Running Flow in a browser

Flow is operated by hand or by browser automation. Either way, these keep
a generation batch smooth.

## Queue discipline

- **One session, sequential queue.** Run generations one after another in
  a single browser session. Parallel sessions on the same account fight
  over credit approvals and project state, and you lose track of which
  clip is which.
- **Render patience.** A clip takes a long time end to end: queue wait,
  render, optional upscale. A full batch runs for hours in the background.
  That is normal. Queue the batch, do other work, come back.

## The prompt composer

- Flow's prompt composer sometimes rejects scripted or autofilled text.
  The reliable fallback is the clipboard: fill the prompt into any normal
  text field, copy it, click into the composer, paste. If pasting still
  fails, type the first few characters by hand, then paste the rest.
- **State the medium.** In Flow's agent-assisted modes, begin with an
  explicit instruction such as "Generate a video: ...". Without it, the
  assistant may default to generating an image.

## Session hygiene

- Stay signed into the same Google account for the whole project. Do not
  switch accounts mid-batch; assets, ingredients, and credit counts follow
  the account, and a mid-batch switch is how clips end up in the wrong
  project.
- Download approved clips immediately with slug filenames
  (`03-market-morning.mp4`). The web app is a workbench, not an archive.

## Working with an agent

If an AI agent drives the browser for you, give it the clip list with
prompts in order and let it work through the queue in one task. One task,
one session, sequential generations. Hand it this skill's references so
it writes prompts, applies the QA pass, and names downloads the same way
you would.
