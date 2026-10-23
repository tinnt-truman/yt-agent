# QA for Veo web clips

Every clip is verified across its full duration before it touches the
timeline. The first frame is a thumbnail; the clip is eight seconds.

## The pass

1. **Sample 5 frames** spread across the whole clip. ffmpeg thumbnails or
   the contact-sheet script in the `video-qa` skill both work. Even
   spacing; include the last second, where drift loves to hide.
2. **Check each frame against the clip's brief:**
   - **Identity.** Does the subject match the reference sheet? Faces,
     products, logos: same as the ingredient, every frame.
   - **Anatomy and physics.** Hands, limbs, object coherence. Nothing
     melting, morphing, or popping in and out.
   - **Continuity.** Wardrobe, lighting direction, and art style
     consistent with the clips it will sit between.
   - **Text.** Any on-screen text garbled or invented? (There should be
     none; see the prompting notes.)
   - **Beat.** Does the clip actually show the action the story needs, or
     is it a pretty shot of nothing happening?
3. **Survivors get 10+ frames.** Clips that pass the 5-frame screen get a
   denser pass. Drift often hides mid-clip and only shows on the frames
   you skipped.
4. **Classify the failure, then apply the fix:**
   - Identity drift: reattach reference images, strengthen the Preserve
     slot, simplify the action.
   - Morphing mid-clip: cut the action down to one beat, reduce camera
     movement, regenerate.
   - Wrong action: rewrite the Action slot only. Leave the rest of the
     prompt alone so you change one variable at a time.
   - Style drift: restate Style & Ambiance verbatim from the locked look;
     check the ingredient images are the right ones.
   - Unfixable physics (hands, complex interaction): simplify the shot or
     cut the beat. Some things are still beyond the model; design around
     them.
5. **Attempt budget: 4 generations per clip, max.** After that, cut the
   beat, simplify the shot, or accept a plainer version. Do not burn the
   month's credit pool chasing one clip.
6. **Log the verdict** per clip: ship, regenerate (with the attempt
   count), or cut. The next session should know what happened without
   re-watching everything.
7. **Watch it end to end, with sound.** Frame samples catch spatial
   defects; only a full watch catches temporal ones (flicker, pulsing,
   morphing between samples) and audio problems. No clip ships unwatched.
   This mirrors the `video-qa` layered gates: cheap checks first, the
   expensive full watch last.

## Assembly gate

After QA, the surviving clips go through the `video-qa` skill's normal
gates (continuity across joins, the assembly checklist) and then the
`assembly-and-delivery` skill for stitching. Clip-level QA does not
replace join-level QA: two perfect clips can still cut together badly.
