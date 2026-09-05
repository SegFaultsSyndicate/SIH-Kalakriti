<!--
  apps/artisan/src/routes/listing/new/+layout.svelte

  Owns the draft id every /listing/new/* step reads from the `?d=` query
  param. Arriving at /listing/new/capture with none creates a fresh draft
  and stamps its id into the URL (replaceState, so the back button doesn't
  land on a URL with no draft); arriving anywhere else in the wizard with no
  `d` (a stale bookmark, a manual URL edit) sends the artisan back to
  capture rather than crashing every step's `getDraft(draftId)` call on
  undefined. Resuming an existing draft (the home screen's drafts list)
  links straight to the right step with `?d=` already set, so this never
  fires for a resume.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { createDraft } from '$lib/listing-draft';

  let { children }: { children: Snippet } = $props();

  $effect(() => {
    if (page.url.searchParams.get('d')) return;

    if (page.url.pathname.endsWith('/capture')) {
      void createDraft().then((draft) => {
        const url = new URL(page.url);
        url.searchParams.set('d', draft.id);
        void goto(url, { replaceState: true });
      });
    } else {
      void goto('/listing/new/capture');
    }
  });
</script>

{@render children()}
