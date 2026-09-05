// apps/admin/src/lib/demo.ts
//
// PUBLIC_DEMO_MODE=1 marks a build for the ministry demo -- it does not
// change any data path (there is no seed-data backend endpoint yet, see
// web/DEMO.md), only cosmetic/timing choices a screen may read this flag
// for: e.g. skipping a randomized empty-state illustration in favour of a
// fixed one, so a rehearsed script always shows the same screen.
import { env } from '$env/dynamic/public';

export const demoMode = env.PUBLIC_DEMO_MODE === '1';
