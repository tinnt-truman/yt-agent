# Writing prompts for Veo

Veo follows structured prompts unusually well. Write every clip prompt in
five parts, in this order, then the Preserve/Avoid slots.

1. **Cinematography.** The shot itself: angle, lens, movement, framing.
   Lead with this. Veo keys on camera language more than most models, so
   write it precisely (the `cinematic-direction` skill's vocabulary).
2. **Subject.** Who or what is in frame, using the locked description
   verbatim. Paste it; never paraphrase mid-project.
3. **Action.** One clear action. A clip gets one beat. Two actions in one
   prompt is how morphing starts.
4. **Context.** Where and when: environment, lighting conditions, time of
   day, weather.
5. **Style & Ambiance.** The look: medium, color grade, mood. If you are
   keeping native audio, describe the sound bed here too.

Then:

- **Preserve:** what must not change across clips (identity, wardrobe,
  logo placement, art style, color palette).
- **Avoid:** what must not appear (extra limbs, garbled text, watermarks,
  photoreal drift in a stylized project, style drift between clips).

## Worked example

A product spot, one 8-second clip:

> Cinematography: slow dolly-in, eye level, 35mm, shallow depth of field.
> Subject: a matte-black insulated steel bottle, brushed finish, no label.
> Action: morning light sweeps across the bottle as fine mist drifts past.
> Context: minimalist concrete kitchen counter at dawn, soft window light
> from the left.
> Style & Ambiance: premium commercial photography, muted warm grade,
> quiet morning stillness.
> Preserve: bottle shape, matte finish, no label, warm muted grade.
> Avoid: text, logos, hands, people, plastic look.

## Notes

- For Ingredients-to-Video, describe only the motion and the shot. Do not
  redescribe the reference image; the image already says what the subject
  looks like, and restating it invites drift. (The full principle is in
  `../../video-prompt-craft/references/i2v-prompting.md`.)
- Keep on-screen text out of prompts. Veo renders text unreliably; plan
  to composite titles and signage in post.
- If a clip needs dialogue, specify speaker, line, and language
  explicitly, and verify sync per line. Native dialogue is a Veo strength,
  but "strength" still means "check it."
