<!--
  apps/buyer/src/lib/ArtisanConciergeModal.svelte

  Live Loom Tour & VIP Concierge booking dialog. Built on @kalakriti/ui's
  accessible native <Dialog> primitive. Allows buyers to schedule a
  10-minute live video meeting with master weavers at their active looms.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Dialog, Button } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    isOpen: boolean;
  }

  let { isOpen = $bindable(false) }: Props = $props();

  const t = $derived(locale.t);

  let name = $state('');
  let contact = $state('');
  let craft = $state('banarasi');
  let date = $state('');
  let timeSlot = $state('morning');
  let submitted = $state(false);

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    setTimeout(() => {
      setTimeout(() => {
        isOpen = false;
        submitted = false;
        name = '';
        contact = '';
      }, 2000);
    }, 300);
  }
</script>

<Dialog bind:open={isOpen} title={t('home.concierge.modalTitle')}>
  <div class="concierge-content">
    <p class="concierge-subtitle">{t('home.concierge.modalSubtitle')}</p>

    {#if submitted}
      <div class="concierge-success" role="status">
        <Icon name="check" size="2rem" />
        <p class="success-heading">{t('home.concierge.success')}</p>
        <p class="success-note">
          A master guild coordinator has reserved your consultation for {name}.
          You will receive the video connection details via SMS / WhatsApp shortly.
        </p>
      </div>
    {:else}
      <form onsubmit={handleSubmit} class="concierge-form">
        <div class="field-row">
          <label for="concierge-name" class="field-label">{t('home.concierge.nameLabel')}</label>
          <input
            id="concierge-name"
            type="text"
            required
            bind:value={name}
            placeholder="e.g. Ananya Sharma"
            class="field-input"
          />
        </div>

        <div class="field-row">
          <label for="concierge-contact" class="field-label">{t('home.concierge.emailLabel')}</label>
          <input
            id="concierge-contact"
            type="text"
            required
            bind:value={contact}
            placeholder="e.g. ananya@example.com or +91 98765 43210"
            class="field-input"
          />
        </div>

        <div class="field-row">
          <label for="concierge-craft" class="field-label">{t('home.concierge.preferredCraft')}</label>
          <select id="concierge-craft" bind:value={craft} class="field-select">
            <option value="banarasi">Varanasi Kadwa Silk & Zari Pit-Loom</option>
            <option value="ajrakh">Kutch 16-Stage Natural Indigo & Dabu</option>
            <option value="pashmina">Kashmir Imperial Sozni Needle Pashmina</option>
            <option value="pochampally">Telangana Double-Ikat Warp Tension</option>
            <option value="dokra">Bastar Lost-Wax Molten Bell Metal</option>
            <option value="paithani">Maharashtra Pure Gold Zari Paithani</option>
          </select>
        </div>

        <div class="split-row">
          <div class="field-row flex-1">
            <label for="concierge-date" class="field-label">{t('home.concierge.dateLabel')}</label>
            <input
              id="concierge-date"
              type="date"
              required
              bind:value={date}
              class="field-input"
            />
          </div>
          <div class="field-row flex-1">
            <label for="concierge-slot" class="field-label">Preferred Window</label>
            <select id="concierge-slot" bind:value={timeSlot} class="field-select">
              <option value="morning">Morning (10:00 AM - 1:00 PM IST)</option>
              <option value="afternoon">Afternoon (2:00 PM - 5:00 PM IST)</option>
              <option value="evening">Evening (5:00 PM - 8:00 PM IST)</option>
            </select>
          </div>
        </div>

        <div class="form-actions">
          <Button variant="primary" type="submit">
            <Icon name="calendar" />
            <span>{t('home.concierge.submit')}</span>
          </Button>
        </div>
      </form>
    {/if}
  </div>
</Dialog>

<style>
  .concierge-content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .concierge-subtitle {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0;
    line-height: 1.5;
  }

  .concierge-form {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .field-row {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .split-row {
    display: flex;
    gap: var(--k-space-3);
  }

  .flex-1 {
    flex: 1;
  }

  .field-label {
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .field-input,
  .field-select {
    font-family: inherit;
    font-size: var(--k-text-sm);
    padding: var(--k-space-2) var(--k-space-3);
    background-color: var(--k-surface-base);
    color: var(--k-text-primary);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    box-sizing: border-box;
    inline-size: 100%;
  }

  .field-input:focus,
  .field-select:focus {
    outline: 2px solid var(--k-focus-ring);
    outline-offset: 1px;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    margin-block-start: var(--k-space-2);
  }

  .concierge-success {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: var(--k-space-5);
    background-color: var(--k-surface-sunken);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    color: var(--k-accent-success);
    gap: var(--k-space-2);
  }

  .success-heading {
    font-family: var(--k-font-display);
    font-size: var(--k-text-lg);
    color: var(--k-text-primary);
    margin: 0;
  }

  .success-note {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin: 0;
    max-inline-size: 40ch;
  }
</style>
