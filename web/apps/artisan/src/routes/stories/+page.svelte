<!--
  apps/artisan/src/routes/stories/+page.svelte

  The design-system review page: every packages/ui primitive, in every
  state it can be in, with a theme switch at the top. This is how Batch 3
  gets reviewed -- not by reading the component source, by looking at this
  page in all three themes and by tabbing through it with a keyboard.

  Dev-only: not linked from the app's own navigation, not precached
  specially, and excluded from anything that ships as a reviewable app
  screen. It stays in apps/artisan because Playwright's axe pass and the
  keyboard-traversal test target it directly (e2e/tests/artisan/stories.spec.ts).
-->
<script lang="ts">
  import { locale, tooltip } from '@kalakriti/i18n';
  import {
    Button,
    Card,
    Skeleton,
    Label,
    Input,
    Textarea,
    Select,
    Checkbox,
    Radio,
    Switch,
    NumberStepper,
    FieldGroup,
    VoiceInput,
    AudioPlayback,
    Sheet,
    Dialog,
    showToast,
    Tabs,
    Accordion,
    Tooltip,
    Popover,
    Image,
    EmptyState,
    SectionHeader,
    Money,
    Stepper,
  } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  // A real (if tiny) 0.2s silent WAV, generated rather than hand-typed -- a
  // malformed one previously reported Infinity for duration and crashed
  // AudioPlayback's <progress> element. That failure mode is now guarded in
  // the component itself (AudioPlayback.svelte's `finite` helper), but a
  // valid clip is still the honest thing to demo with.
  const SILENT_WAV =
    'data:audio/wav;base64,UklGRmQGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YUAGAACAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICA';

  let theme = $state<'light' | 'dark' | 'high-contrast'>('light');
  let textScale = $state('');

  $effect(() => {
    document.documentElement.dataset.theme = theme;
  });
  $effect(() => {
    if (textScale) document.documentElement.dataset.textScale = textScale;
    else delete document.documentElement.dataset.textScale;
  });

  let name = $state('');
  let bio = $state('');
  let craft = $state('weaving');
  let agreed = $state(false);
  let paymentTerms = $state('advance');
  let notifications = $state(true);
  let quantity = $state(3);

  let dialogOpen = $state(false);
  let sheetOpen = $state(false);
  let popoverOpen = $state(false);

  let activeTab = $state('photos');
  let openAccordion = $state<string[]>([]);

  function fireToast(variant: 'info' | 'success' | 'error'): void {
    showToast({
      message:
        variant === 'error'
          ? 'Upload failed. Check your connection.'
          : variant === 'success'
            ? 'Listing saved.'
            : 'Syncing in the background.',
      variant,
      action: variant === 'error' ? { label: 'Retry', onclick: () => {} } : undefined,
    });
  }
</script>

<svelte:head>
  <title>Kalakriti — component stories</title>
</svelte:head>

<div class="stories">
  <div class="stories__controls">
    <label>
      theme
      <select bind:value={theme}>
        <option value="light">light</option>
        <option value="dark">dark</option>
        <option value="high-contrast">high-contrast</option>
      </select>
    </label>
    <label>
      text scale
      <select bind:value={textScale}>
        <option value="">100%</option>
        <option value="large">125%</option>
        <option value="xlarge">150%</option>
        <option value="xxlarge">200%</option>
      </select>
    </label>
  </div>

  <h1>Component stories</h1>

  <section aria-labelledby="s-button">
    <h2 id="s-button">Button</h2>
    <div class="stories__row">
      <Button variant="primary">Primary</Button>
      <Button variant="secondary">Secondary</Button>
      <Button variant="ghost">Ghost</Button>
      <Button variant="danger">Danger</Button>
      <Button loading>Saving</Button>
      <Button icon="trash" label="Delete photograph" variant="ghost" tooltip={tooltip('tooltip.delete')} />
    </div>
    <div class="stories__row">
      <Button size="sm">Small</Button>
      <Button size="md">Medium</Button>
      <Button size="lg">Large</Button>
      <Button size="xl">Extra large (artisan default)</Button>
    </div>
  </section>

  <section aria-labelledby="s-fields">
    <h2 id="s-fields">Fields</h2>
    <FieldGroup label="Your name" description="As you would like it printed on your certificate">
      {#snippet children({ id, describedBy })}
        <Input {id} aria-describedby={describedBy} bind:value={name} />
      {/snippet}
    </FieldGroup>

    <FieldGroup label="Tell us about your craft" optional error={bio.length > 200 ? 'Keep it under 200 characters' : ''}>
      {#snippet children({ id, describedBy })}
        <Textarea {id} aria-describedby={describedBy} bind:value={bio} />
      {/snippet}
    </FieldGroup>

    <FieldGroup label="Craft type">
      {#snippet children({ id })}
        <Select
          {id}
          bind:value={craft}
          options={[
            { value: 'weaving', label: 'Weaving' },
            { value: 'pottery', label: 'Pottery' },
            { value: 'block-printing', label: 'Block printing' },
          ]}
        />
      {/snippet}
    </FieldGroup>

    <Checkbox bind:checked={agreed}>I confirm this listing is my own work</Checkbox>

    <fieldset class="stories__fieldset">
      <legend>Payment terms</legend>
      <Radio name="terms" value="advance" bind:group={paymentTerms}>50% advance</Radio>
      <Radio name="terms" value="delivery" bind:group={paymentTerms}>On delivery</Radio>
    </fieldset>

    <Switch bind:checked={notifications}>Order notifications</Switch>

    <div class="stories__field-block">
      <Label for="qty">Quantity available</Label>
      <NumberStepper id="qty" bind:value={quantity} min={0} max={50} />
    </div>
  </section>

  <section aria-labelledby="s-voice">
    <h2 id="s-voice">VoiceInput</h2>
    <div class="stories__row">
      <VoiceInput mode="tap" onrecording={() => {}} />
      <VoiceInput mode="hold" onrecording={() => {}} />
    </div>
  </section>

  <section aria-labelledby="s-audio">
    <h2 id="s-audio">AudioPlayback</h2>
    <AudioPlayback src={SILENT_WAV} label="Listing description" />
  </section>

  <section aria-labelledby="s-overlay">
    <h2 id="s-overlay">Sheet, Dialog, Toast, Popover</h2>
    <div class="stories__row">
      <Button onclick={() => (dialogOpen = true)}>Open dialog</Button>
      <Button onclick={() => (sheetOpen = true)}>Open sheet</Button>
      <Button onclick={() => fireToast('info')}>Info toast</Button>
      <Button onclick={() => fireToast('success')}>Success toast</Button>
      <Button variant="danger" onclick={() => fireToast('error')}>Error toast</Button>
      <Popover bind:open={popoverOpen}>
        {#snippet trigger(props)}
          <Button icon="more-vertical" label="More actions" variant="secondary" {...props} tooltip={tooltip('tooltip.openMenu')} />
        {/snippet}
        {#snippet children()}
          <Button variant="ghost" size="sm">Edit</Button>
          <Button variant="ghost" size="sm">Archive</Button>
        {/snippet}
      </Popover>
    </div>

    <Dialog bind:open={dialogOpen} title="Delete this listing?">
      <p>This cannot be undone. Your photographs stay saved on this phone.</p>
      <div class="stories__row">
        <Button variant="danger" onclick={() => (dialogOpen = false)}>Delete</Button>
        <Button variant="ghost" onclick={() => (dialogOpen = false)}>Cancel</Button>
      </div>
    </Dialog>

    <Sheet bind:open={sheetOpen} title="Filter results">
      <Checkbox checked>Handloom verified</Checkbox>
      <Checkbox>GI tagged</Checkbox>
    </Sheet>
  </section>

  <section aria-labelledby="s-tabs">
    <h2 id="s-tabs">Tabs</h2>
    <Tabs
      tabs={[
        { id: 'photos', label: 'Photographs' },
        { id: 'details', label: 'Details' },
        { id: 'pricing', label: 'Pricing' },
      ]}
      bind:selected={activeTab}
    >
      {#snippet children(id)}
        <p>Panel content for "{id}".</p>
      {/snippet}
    </Tabs>
  </section>

  <section aria-labelledby="s-accordion">
    <h2 id="s-accordion">Accordion</h2>
    <Accordion
      items={[
        { id: 'materials', heading: 'Materials used' },
        { id: 'care', heading: 'Care instructions' },
      ]}
      bind:open={openAccordion}
    >
      {#snippet children(id)}
        <p>Details for "{id}".</p>
      {/snippet}
    </Accordion>
  </section>

  <section aria-labelledby="s-tooltip">
    <h2 id="s-tooltip">Tooltip</h2>
    <Tooltip text="A legal mark of geographical origin, verified by the registry.">
      {#snippet trigger(props)}
        <button class="stories__gi" {...props}>
          GI tagged <Icon name="info" size="1rem" />
        </button>
      {/snippet}
    </Tooltip>
  </section>

  <section aria-labelledby="s-card">
    <h2 id="s-card">Card</h2>
    <div class="stories__row">
      <Card variant="flat"><p>Flat</p></Card>
      <Card variant="hairline"><p>Hairline</p></Card>
      <Card variant="printed"><p>Printed</p></Card>
      <Card variant="media">
        <Image src="/favicon.svg" alt="" ratio="4/3" />
        <div class="k-card__body"><p>Media-led</p></div>
      </Card>
    </div>
  </section>

  <section aria-labelledby="s-image">
    <h2 id="s-image">Image</h2>
    <div class="stories__row">
      <Image src="/favicon.svg" alt="Kalakriti mark" ratio="1/1" />
      <Image src="/does-not-exist.jpg" alt="A photograph that failed to load" ratio="1/1" />
    </div>
  </section>

  <section aria-labelledby="s-skeleton">
    <h2 id="s-skeleton">Skeleton</h2>
    <div class="stories__row">
      <Skeleton shape="text" />
      <Skeleton shape="media" />
      <Skeleton shape="card" />
    </div>
  </section>

  <section aria-labelledby="s-empty">
    <h2 id="s-empty">EmptyState</h2>
    <EmptyState
      illustration="empty-no-listings"
      heading={t('state.empty.title')}
      body="Photograph your first piece to start a listing."
    >
      {#snippet action()}
        <Button>Photograph a piece</Button>
      {/snippet}
    </EmptyState>
  </section>

  <section aria-labelledby="s-section-header">
    <h2 id="s-section-header">SectionHeader</h2>
    <SectionHeader kicker="drape tradition with grace" heading="Dupattas" href="/c/dupattas" />
  </section>

  <section aria-labelledby="s-money">
    <h2 id="s-money">Money</h2>
    <div class="stories__row">
      <Money paise={4500000} />
      <Money paise={99} />
      <Money paise={-1250} />
    </div>
  </section>

  <section aria-labelledby="s-stepper">
    <h2 id="s-stepper">Stepper</h2>
    <Stepper label="Listing progress" steps={['Photograph', 'Details', 'Price', 'Review']} current={1} />
  </section>
</div>

<style>
  .stories {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-7);
    max-inline-size: 100%;
  }
  .stories__controls {
    position: sticky;
    inset-block-start: 0;
    z-index: var(--k-z-sticky);
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-4);
    padding-block: var(--k-space-3);
    background: var(--k-surface-base);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }
  .stories__row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--k-space-3);
  }
  .stories__field-block { margin-block-end: var(--k-space-5); }
  .stories__fieldset {
    border: none;
    padding: 0;
    margin: 0 0 var(--k-space-5);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }
  .stories__fieldset legend {
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-medium);
    margin-block-end: var(--k-space-2);
    padding: 0;
  }
  .stories__gi {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    background: transparent;
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    padding: var(--k-space-2) var(--k-space-3);
    color: var(--k-text-primary);
  }
</style>
