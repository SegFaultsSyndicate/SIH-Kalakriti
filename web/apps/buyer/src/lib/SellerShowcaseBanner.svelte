<!--
  apps/buyer/src/lib/SellerShowcaseBanner.svelte

  Artisan Empowerment & Seller Showcase Section:
  - "Sell Hassle Free / Become a Kalakriti Seller" banner with 0% commission, doorstep pickup, and direct DBT bank settlement.
  - "Hear from our Happy Sellers" video testimonial cards with national and state awardee artisans.
  - Official National Partner & Government Accreditation strip (Make in India, Digital India, india.gov.in, GI Registry, ONDC).
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  const testimonials = [
    {
      id: 'seller-1',
      name: 'Altaf Hussain',
      title: 'National Awardee Woodcarver',
      cluster: 'Saharanpur, Uttar Pradesh',
      craft: 'Shisham Inlay & Jali Carving',
      quote: 'A National Awardee Seller showcasing the beauty of handcrafted heritage creations. Through Kalakriti, we connect directly with global patrons without commission brokers.',
      image: 'https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?auto=format&fit=crop&w=600&q=80',
    },
    {
      id: 'seller-2',
      name: 'Mohammed Shafiuddin',
      title: 'State Awardee Master Craftsman',
      cluster: 'Bidar, Karnataka',
      craft: 'GI-Certified Bidriware Metalwork',
      quote: 'Showcasing our 600-year-old silver inlay art globally through digital infrastructure provided by the Government of India. Orders reach our village workshop with zero intermediaries.',
      image: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?auto=format&fit=crop&w=600&q=80',
    },
    {
      id: 'seller-3',
      name: 'Amrita Saavan',
      title: 'Master Weaver & SHG Leader',
      cluster: 'Imphal, Manipur',
      craft: 'Shaphee Lanphee & Loin Loom',
      quote: 'Our women’s cooperative registered in minutes using the low-literacy Voice PWA. Prompt doorstep logistics and DBT bank settlements have revived our traditional loin looms.',
      image: 'https://images.unsplash.com/photo-1573496359142-b8d87734a5a2?auto=format&fit=crop&w=600&q=80',
    },
  ];
</script>

<div class="seller-showcase-container">
  <!-- 1. Become a Seller Hero Banner -->
  <div class="seller-cta-banner">
    <div class="seller-cta-content">
      <span class="seller-kicker">Empowering Master Craftspersons</span>
      <h2 class="seller-title">Sell Hassle Free • Become a {t('app.name')} Seller</h2>
      <p class="seller-subtitle">
        Join over 12,000 verified rural artisans, weavers, and self-help guilds across 74 national craft corridors.
      </p>

      <!-- 3 Key Value Props -->
      <div class="seller-perks-row">
        <div class="perk-item">
          <div class="perk-icon">
            <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
          </div>
          <div class="perk-meta">
            <strong>Zero Commission</strong>
            <span>100% fair-trade proceeds directly to you</span>
          </div>
        </div>

        <div class="perk-item">
          <div class="perk-icon">
            <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="1" y="3" width="15" height="13"></rect>
              <polygon points="16 8 20 8 23 11 23 16 16 16 16 8"></polygon>
              <circle cx="5.5" cy="18.5" r="2.5"></circle>
              <circle cx="18.5" cy="18.5" r="2.5"></circle>
            </svg>
          </div>
          <div class="perk-meta">
            <strong>Easy Pickup &amp; Delivery</strong>
            <span>Doorstep postal dispatch from your village</span>
          </div>
        </div>

        <div class="perk-item">
          <div class="perk-icon">
            <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="5" width="20" height="14" rx="2"></rect>
              <line x1="2" y1="10" x2="22" y2="10"></line>
            </svg>
          </div>
          <div class="perk-meta">
            <strong>Direct DBT Bank Payments</strong>
            <span>Immediate escrow settlement upon delivery</span>
          </div>
        </div>
      </div>

      <!-- Action Button -->
      <div class="seller-cta-action">
        <a 
          href="http://localhost:5173/onboarding" 
          target="_blank" 
          rel="noopener noreferrer" 
          class="seller-register-btn"
        >
          <span>Register Now as Master Artisan</span>
          <Icon name="arrow-right" size="1.1rem" />
        </a>
        <span class="seller-helper-text">Supports Voice Onboarding in 11 Indian Languages</span>
      </div>
    </div>
  </div>

  <!-- 2. Hear from our Happy Sellers / Testimonials -->
  <div class="sellers-testimonials-section">
    <div class="testimonials-header">
      <span class="testimonials-kicker">Voices of Guild Transformation</span>
      <h3 class="testimonials-title">Hear from our Happy Sellers</h3>
    </div>

    <div class="testimonials-grid">
      {#each testimonials as seller}
        <article class="testimonial-card">
          <div class="testimonial-card__media">
            <img src={seller.image} alt={seller.name} class="testimonial-img" loading="lazy" />
            <div class="video-play-indicator" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
                <polygon points="5 3 19 12 5 21 5 3"></polygon>
              </svg>
            </div>
            <div class="card-award-badge">{seller.title}</div>
          </div>
          <div class="testimonial-card__body">
            <h4 class="seller-name">{seller.name}</h4>
            <span class="seller-cluster"><Icon name="location" size="0.85rem" /> {seller.cluster}</span>
            <span class="seller-craft"><Icon name="verified-artisan" size="0.85rem" /> {seller.craft}</span>
            <p class="seller-quote">"{seller.quote}"</p>
          </div>
        </article>
      {/each}
    </div>
  </div>

  <!-- 3. Official Government Accreditation & Partner Logos Strip -->
  <div class="official-accreditation-strip" aria-label="Official Government Partners">
    <div class="accreditation-item">
      <span class="emblem-text">MAKE IN INDIA</span>
    </div>
    <div class="accreditation-item">
      <span class="emblem-text">DIGITAL INDIA</span>
    </div>
    <div class="accreditation-item">
      <span class="emblem-text">INDIA.GOV.IN</span>
    </div>
    <div class="accreditation-item">
      <span class="emblem-text">MINISTRY OF SOCIAL JUSTICE &amp; EMPOWERMENT</span>
    </div>
    <div class="accreditation-item">
      <span class="emblem-text">GEOGRAPHICAL INDICATIONS REGISTRY</span>
    </div>
    <div class="accreditation-item">
      <span class="emblem-text">ONDC COMPLIANT</span>
    </div>
  </div>
</div>

<style>
  .seller-showcase-container {
    display: flex;
    flex-direction: column;
    gap: 3.5rem;
    margin-block: 2.5rem 1rem;
  }

  /* 1. Become a Seller Hero Banner */
  .seller-cta-banner {
    background: linear-gradient(135deg, #fdfbf7 0%, #f4ede4 100%);
    border: 1px solid var(--k-border-subtle, #e7e5e4);
    border-radius: 12px;
    padding: 3rem 2.5rem;
    position: relative;
    overflow: hidden;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.03);
  }

  .seller-cta-content {
    max-inline-size: 58rem;
    margin-inline: auto;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.25rem;
  }

  .seller-kicker {
    font-size: 0.8125rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--k-accent-secondary, #c65d3b);
  }

  .seller-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: clamp(1.6rem, 3.5vw, 2.4rem);
    font-weight: 700;
    color: var(--k-text-primary, #1c1917);
    margin: 0;
    line-height: 1.25;
  }

  .seller-subtitle {
    font-size: 1rem;
    color: var(--k-text-secondary, #57534e);
    max-inline-size: 44rem;
    line-height: 1.5;
    margin: 0;
  }

  /* Perks Row */
  .seller-perks-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.5rem;
    inline-size: 100%;
    margin-block: 1rem;
    text-align: left;
  }

  @media (max-width: 50rem) {
    .seller-perks-row {
      grid-template-columns: 1fr;
      gap: 1rem;
    }
  }

  .perk-item {
    background-color: #ffffff;
    border: 1px solid rgba(198, 93, 59, 0.15);
    border-radius: 8px;
    padding: 1.15rem 1.25rem;
    display: flex;
    align-items: center;
    gap: 1rem;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
  }

  .perk-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    aspect-ratio: 1;
    border-radius: 50%;
    background-color: rgba(198, 93, 59, 0.1);
    color: var(--k-accent-secondary, #c65d3b);
    flex-shrink: 0;
  }

  .perk-meta {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .perk-meta strong {
    font-size: 0.9rem;
    color: var(--k-text-primary, #1c1917);
    font-weight: 700;
  }

  .perk-meta span {
    font-size: 0.775rem;
    color: var(--k-text-secondary, #78716c);
    line-height: 1.35;
  }

  .seller-cta-action {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    margin-block-start: 0.5rem;
  }

  .seller-register-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.75rem;
    background-color: #7b1d24; /* Refined deep madder red matching government craft initiatives */
    color: #ffffff;
    font-size: 0.95rem;
    font-weight: 700;
    padding: 0.85rem 2rem;
    border-radius: 6px;
    text-decoration: none;
    box-shadow: 0 4px 12px rgba(123, 29, 36, 0.25);
    transition: transform 0.15s ease, background-color 0.15s ease;
  }

  .seller-register-btn:hover {
    background-color: #63141a;
    transform: translateY(-2px);
  }

  .seller-helper-text {
    font-size: 0.75rem;
    color: var(--k-text-secondary, #78716c);
    font-style: italic;
  }

  /* 2. Testimonials Section */
  .testimonials-header {
    text-align: center;
    margin-block-end: 2rem;
  }

  .testimonials-kicker {
    display: block;
    font-size: 0.78rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--k-accent-secondary, #c65d3b);
  }

  .testimonials-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: clamp(1.5rem, 3vw, 2rem);
    font-weight: 700;
    color: var(--k-text-primary, #1c1917);
    margin: 0.35rem 0 0 0;
  }

  .testimonials-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.5rem;
  }

  @media (max-width: 60rem) {
    .testimonials-grid {
      grid-template-columns: 1fr;
    }
  }

  .testimonial-card {
    background-color: #ffffff;
    border: 1px solid var(--k-border-subtle, #e7e5e4);
    border-radius: 8px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.03);
    transition: transform 0.2s ease, box-shadow 0.2s ease;
  }

  .testimonial-card:hover {
    transform: translateY(-3px);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.06);
  }

  .testimonial-card__media {
    position: relative;
    aspect-ratio: 16 / 10;
    background-color: #f5f5f4;
    overflow: hidden;
  }

  .testimonial-img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
    transition: transform 0.3s ease;
  }

  .testimonial-card:hover .testimonial-img {
    transform: scale(1.03);
  }

  .video-play-indicator {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    inline-size: 3rem;
    aspect-ratio: 1;
    border-radius: 50%;
    background-color: var(--k-surface-inverse);
    color: var(--k-text-on-inverse);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .card-award-badge {
    position: absolute;
    bottom: 0.75rem;
    left: 0.75rem;
    background-color: var(--k-surface-inverse);
    color: var(--k-text-on-inverse);
    font-size: 0.69rem;
    font-weight: 600;
    padding: 0.25rem 0.6rem;
    border-radius: 4px;
  }

  .testimonial-card__body {
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    flex: 1;
  }

  .seller-name {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--k-text-primary, #1c1917);
    margin: 0;
  }

  .seller-cluster {
    font-size: 0.75rem;
    color: var(--k-text-secondary, #78716c);
    display: flex;
    align-items: center;
    gap: 0.35rem;
  }

  .seller-craft {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--k-accent-secondary, #c65d3b);
    display: flex;
    align-items: center;
    gap: 0.35rem;
  }

  .seller-quote {
    font-size: 0.825rem;
    line-height: 1.5;
    color: var(--k-text-secondary, #44403c);
    margin: 0.5rem 0 0 0;
    font-style: italic;
  }

  /* 3. Official Government Accreditation Strip */
  .official-accreditation-strip {
    border-block-start: 1px solid var(--k-border-subtle, #e7e5e4);
    border-block-end: 1px solid var(--k-border-subtle, #e7e5e4);
    padding-block: 1.25rem;
    display: flex;
    align-items: center;
    justify-content: space-around;
    flex-wrap: wrap;
    gap: 1.5rem;
    background-color: #fafaf9;
    border-radius: 6px;
  }

  .accreditation-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .emblem-text {
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    color: #78716c;
    text-transform: uppercase;
  }
</style>
