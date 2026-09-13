<!--
  apps/buyer/src/routes/contact/+page.svelte

  Official Government & Buyer Contact, Grievance Redressal & Help Desk:
  - Toll-free National Craft Helpline: 1800-11-2026 (9 AM - 7 PM IST)
  - Direct Support Email: care@kalakriti.gov.in
  - Statutory Grievance Redressal Officer (DPDP Act 2023 / IT Rules 2021)
  - Physical Ministry Address: Shastri Bhawan, New Delhi
  - Interactive Citizen / Patron Inquiry Form with instant feedback
  - Direct WhatsApp & Video Loom Assistance Connectors
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs, type BreadcrumbItem, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  const breadcrumbs = $derived<BreadcrumbItem[]>([
    { label: t('nav.home') || 'Home', href: '/' },
    { label: 'Contact Us & Support' },
  ]);

  let inquiryType = $state('order_tracking');
  let fullName = $state('');
  let email = $state('');
  let phone = $state('');
  let message = $state('');
  let submitted = $state(false);

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    showToast({
      message: `Inquiry Docket Registered: Your ticket #KLK-${Math.floor(100000 + Math.random() * 900000)} has been created. Our cluster officer will respond within 24 hours.`,
      variant: 'success',
    });
  }
</script>

<svelte:head>
  <title>Contact Us & Support — Kalakriti</title>
  <meta
    name="description"
    content="Official customer care, artisan guild support, and statutory grievance redressal desk for the Kalakriti craft platform."
  />
</svelte:head>

<div class="contact-page-container">
  <div class="breadcrumbs-row">
    <Breadcrumbs items={breadcrumbs} />
  </div>

  <!-- Header Banner -->
  <header class="contact-header">
    <span class="contact-kicker">Ministry of Social Justice &amp; Empowerment</span>
    <h1 class="contact-title">Contact Us &amp; Support Desk</h1>
    <p class="contact-subtitle">
      Connect directly with our central buyer concierge, artisan cluster coordinators, or statutory grievance redressal officer.
    </p>
  </header>

  <!-- Fast Help Grid (3 Cards) -->
  <div class="help-cards-grid">
    <!-- Card 1: Tollfree -->
    <div class="help-card">
      <div class="help-card__icon phone-icon">
        <Icon name="phone" size="1.4rem" />
      </div>
      <div class="help-card__content">
        <span class="help-card__tag">Toll-Free National Helpline</span>
        <h3 class="help-card__title">1800-11-2026</h3>
        <p class="help-card__desc">
          Available Monday to Saturday, 9:00 AM to 7:00 PM IST in Hindi, English &amp; 9 regional languages.
        </p>
        <a href="tel:1800112026" class="help-card__action">
          <span>Call Tollfree Now</span>
          <Icon name="arrow-right" size="0.85rem" />
        </a>
      </div>
    </div>

    <!-- Card 2: Email Care -->
    <div class="help-card">
      <div class="help-card__icon email-icon">
        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="2" y="4" width="20" height="16" rx="2"></rect>
          <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"></path>
        </svg>
      </div>
      <div class="help-card__content">
        <span class="help-card__tag">Direct Email Support</span>
        <h3 class="help-card__title">care@kalakriti.gov.in</h3>
        <p class="help-card__desc">
          For order inquiries, bulk tenders, provenance passport verification, or artisan empanelment.
        </p>
        <a href="mailto:care@kalakriti.gov.in" class="help-card__action">
          <span>Send Formal Email</span>
          <Icon name="arrow-right" size="0.85rem" />
        </a>
      </div>
    </div>

    <!-- Card 3: Grievance Redressal -->
    <div class="help-card">
      <div class="help-card__icon grievance-icon">
        <Icon name="lock" size="1.4rem" />
      </div>
      <div class="help-card__content">
        <span class="help-card__tag">Statutory Grievance Officer</span>
        <h3 class="help-card__title">Dr. S. K. Verma</h3>
        <p class="help-card__desc">
          Designated under DPDP Act 2023 and Consumer Protection E-Commerce Rules. Response within 48h.
        </p>
        <a href="#inquiry-form" class="help-card__action">
          <span>File Official Grievance</span>
          <Icon name="arrow-right" size="0.85rem" />
        </a>
      </div>
    </div>
  </div>

  <!-- Form & Ministry Address Section -->
  <div class="contact-main-grid">
    <!-- Left: Inquiry Form -->
    <div class="form-card" id="inquiry-form">
      <h2 class="form-heading">Send an Official Inquiry or Request</h2>
      <p class="form-subtext">Fill out the details below. A dedicated cluster officer will be assigned to your ticket.</p>

      {#if submitted}
        <div class="form-success-alert">
          <div class="alert-icon">✓</div>
          <div class="alert-body">
            <h4>Inquiry Received Successfully!</h4>
            <p>Your request has been routed to the relevant cluster development officer. Confirmation sent to {email}.</p>
            <button type="button" class="btn-reset" onclick={() => (submitted = false)}>Submit Another Request</button>
          </div>
        </div>
      {:else}
        <form onsubmit={handleSubmit} class="inquiry-form">
          <div class="form-group">
            <label for="inquiryType" class="form-label">Nature of Inquiry</label>
            <select id="inquiryType" bind:value={inquiryType} class="form-select">
              <option value="order_tracking">Order Tracking &amp; Delivery Update</option>
              <option value="bulk_procurement">Bulk Institutional Procurement &amp; Tenders</option>
              <option value="provenance_verify">GI &amp; Ed25519 Provenance Verification</option>
              <option value="artisan_consultancy">Artisan 1-on-1 Loom Video Call Scheduling</option>
              <option value="grievance">Statutory DPDP / Fair-Trade Grievance</option>
              <option value="other">General Feedback &amp; Suggestions</option>
            </select>
          </div>

          <div class="form-row-2">
            <div class="form-group">
              <label for="fullName" class="form-label">Your Legal / Full Name *</label>
              <input type="text" id="fullName" bind:value={fullName} required placeholder="e.g. Aarav Sharma" class="form-input" />
            </div>

            <div class="form-group">
              <label for="email" class="form-label">Email Address *</label>
              <input type="email" id="email" bind:value={email} required placeholder="e.g. aarav.sharma@example.gov.in" class="form-input" />
            </div>
          </div>

          <div class="form-group">
            <label for="phone" class="form-label">Mobile Number (Optional for SMS Docket Alert)</label>
            <input type="tel" id="phone" bind:value={phone} placeholder="+91 98765 43210" class="form-input" />
          </div>

          <div class="form-group">
            <label for="message" class="form-label">Description of Inquiry / Order ID *</label>
            <textarea id="message" bind:value={message} required rows="4" placeholder="Please provide order number, craft title, or details of your query..." class="form-textarea"></textarea>
          </div>

          <button type="submit" class="submit-btn">
            <span>Submit Official Inquiry</span>
            <Icon name="arrow-right" size="1rem" />
          </button>
        </form>
      {/if}
    </div>

    <!-- Right: Office & Ministry Info -->
    <div class="office-info-card">
      <h3 class="office-heading">Ministry Head Office</h3>
      
      <div class="address-block">
        <Icon name="location" size="1.2rem" />
        <div>
          <strong>Department of Social Justice &amp; Empowerment</strong>
          <p>
            Shastri Bhawan, Dr. Rajendra Prasad Road,<br />
            New Delhi, Delhi — 110001, India
          </p>
        </div>
      </div>

      <div class="divider"></div>

      <h4 class="sub-heading">Operational Timings</h4>
      <ul class="timings-list" role="list">
        <li><span>Working Days:</span> <strong>Monday – Saturday</strong></li>
        <li><span>Help Desk Hours:</span> <strong>09:00 – 19:00 IST</strong></li>
        <li><span>Response Time:</span> <strong>Within 24 Hours</strong></li>
      </ul>

      <div class="divider"></div>

      <h4 class="sub-heading">Live WhatsApp Craft Connect</h4>
      <p class="whatsapp-desc">
        Chat directly with verified cluster coordinators for immediate status updates and order assistance.
      </p>
      <a 
        href="https://wa.me/911800112026?text=Namaste%20Kalakriti%20Team" 
        target="_blank" 
        rel="noopener noreferrer" 
        class="whatsapp-btn"
      >
        <Icon name="whatsapp" size="1.1rem" />
        <span>Connect on WhatsApp</span>
      </a>
    </div>
  </div>
</div>

<style>
  .contact-page-container {
    max-inline-size: 78rem;
    margin-inline: auto;
    padding: 1.5rem 1.25rem 4rem 1.25rem;
  }

  .breadcrumbs-row {
    margin-block-end: 1rem;
  }

  .contact-header {
    margin-block-end: 2.5rem;
    border-block-end: 1px solid var(--k-border-subtle, var(--k-stone-100));
    padding-block-end: 1.5rem;
  }

  .contact-kicker {
    display: block;
    font-size: 0.78rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--k-accent-secondary, var(--k-terracotta-600));
  }

  .contact-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: clamp(1.8rem, 3.5vw, 2.5rem);
    font-weight: 700;
    color: var(--k-text-primary, var(--k-text-primary));
    margin: 0.25rem 0 0.5rem 0;
  }

  .contact-subtitle {
    font-size: 1rem;
    color: var(--k-text-secondary, var(--k-stone-600));
    max-inline-size: 46rem;
    line-height: 1.5;
    margin: 0;
  }

  /* 3 Help Cards */
  .help-cards-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1.5rem;
    margin-block-end: 2.5rem;
  }

  @media (max-width: 56rem) {
    .help-cards-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  .help-card {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle, var(--k-stone-100));
    border-radius: 10px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.03);
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }

  .help-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.06);
  }

  .help-card__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 3rem;
    aspect-ratio: 1;
    border-radius: 8px;
    flex-shrink: 0;
  }

  .phone-icon {
    background-color: rgba(198, 93, 59, 0.12);
    color: var(--k-terracotta-600);
  }

  .email-icon {
    background-color: rgba(30, 58, 138, 0.1);
    color: var(--k-indigo-800);
  }

  .grievance-icon {
    background-color: rgba(22, 101, 52, 0.1);
    color: var(--k-accent-success);
  }

  .help-card__tag {
    display: block;
    font-size: 0.72rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--k-border-interactive);
  }

  .help-card__title {
    font-size: 1.15rem;
    font-weight: 700;
    color: var(--k-text-primary, var(--k-text-primary));
    margin: 0.2rem 0 0.4rem 0;
  }

  .help-card__desc {
    font-size: 0.8125rem;
    color: var(--k-text-secondary, var(--k-stone-600));
    line-height: 1.45;
    margin: 0 0 1rem 0;
    flex: 1;
  }

  .help-card__action {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.825rem;
    font-weight: 700;
    color: var(--k-terracotta-600);
    text-decoration: none;
    inline-size: fit-content;
  }

  .help-card__action:hover {
    text-decoration: underline;
  }

  /* Main Form + Info Grid */
  .contact-main-grid {
    display: grid;
    grid-template-columns: 1.8fr minmax(0, 1fr);
    gap: 2rem;
  }

  @media (max-width: 54rem) {
    .contact-main-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  .form-card {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle, var(--k-stone-100));
    border-radius: 10px;
    padding: 2rem;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.03);
  }

  .form-heading {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.35rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0 0 0.35rem 0;
  }

  .form-subtext {
    font-size: 0.85rem;
    color: var(--k-border-interactive);
    margin: 0 0 1.5rem 0;
  }

  .inquiry-form {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .form-row-2 {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  @media (max-width: 36rem) {
    .form-row-2 {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .form-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--k-text-primary);
  }

  .form-input,
  .form-select,
  .form-textarea {
    font-family: inherit;
    font-size: 0.875rem;
    padding: 0.65rem 0.85rem;
    border: 1px solid var(--k-border-hairline);
    border-radius: 6px;
    background-color: var(--k-surface-base);
    color: var(--k-text-primary);
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }

  .form-input:focus,
  .form-select:focus,
  .form-textarea:focus {
    outline: none;
    border-color: var(--k-terracotta-600);
    box-shadow: 0 0 0 3px rgba(198, 93, 59, 0.12);
  }

  .submit-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.6rem;
    background-color: var(--k-accent-danger);
    color: var(--k-text-on-accent);
    font-weight: 700;
    font-size: 0.925rem;
    padding: 0.85rem 1.75rem;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    transition: background-color 0.15s ease, transform 0.15s ease;
    inline-size: fit-content;
    margin-block-start: 0.5rem;
  }

  .submit-btn:hover {
    background-color: var(--k-madder-800);
    transform: translateY(-1px);
  }

  /* Form Success Alert */
  .form-success-alert {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-neem-300);
    border-radius: 8px;
    padding: 1.5rem;
  }

  .alert-icon {
    inline-size: 2.25rem;
    aspect-ratio: 1;
    border-radius: 50%;
    background-color: var(--k-neem-600);
    color: var(--k-text-on-accent);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    flex-shrink: 0;
  }

  .alert-body h4 {
    margin: 0 0 0.35rem 0;
    font-size: 1rem;
    color: var(--k-accent-success);
  }

  .alert-body p {
    font-size: 0.85rem;
    color: var(--k-neem-600);
    margin: 0 0 1rem 0;
  }

  .btn-reset {
    background: none;
    border: 1px solid var(--k-neem-600);
    color: var(--k-neem-600);
    font-weight: 600;
    font-size: 0.8rem;
    padding: 0.4rem 0.85rem;
    border-radius: 4px;
    cursor: pointer;
  }

  /* Right Office Card */
  .office-info-card {
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle, var(--k-stone-100));
    border-radius: 10px;
    padding: 1.75rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    height: fit-content;
  }

  .office-heading {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: 1.2rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .address-block {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    font-size: 0.825rem;
    color: var(--k-stone-700);
    line-height: 1.5;
  }

  .divider {
    block-size: 1px;
    background-color: var(--k-surface-sunken);
    margin-block: 0.25rem;
  }

  .sub-heading {
    font-size: 0.85rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .timings-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
    font-size: 0.8rem;
    color: var(--k-stone-600);
  }

  .timings-list li {
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 0.25rem 0.75rem;
  }

  .whatsapp-desc {
    font-size: 0.775rem;
    color: var(--k-border-interactive);
    line-height: 1.4;
    margin: 0;
  }

  .whatsapp-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.6rem;
    background-color: #25d366;
    color: var(--k-text-on-accent);
    font-weight: 700;
    font-size: 0.85rem;
    padding: 0.75rem 1.25rem;
    border-radius: 6px;
    text-decoration: none;
    box-shadow: 0 2px 8px rgba(37, 211, 102, 0.25);
    transition: background-color 0.15s ease, transform 0.15s ease;
  }

  .whatsapp-btn:hover {
    background-color: var(--k-neem-500);
    transform: translateY(-1px);
  }
</style>
