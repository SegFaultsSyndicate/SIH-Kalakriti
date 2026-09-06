<!--
  apps/buyer/src/lib/BuyerFooter.svelte

  Comprehensive Enterprise Government & Cultural Marketplace Footer:
  - 6 Trust Assurance Badges (Free Shipping, 100% Purchase Protection, Secure Payment, Assured Quality, Handcrafted Products, Best Price Promise)
  - Detailed Navigation Columns: Resources, Policies & Info, Craft Corridors & Institutional Procurement
  - Social Media Links with refined SVG icons (Facebook, X, LinkedIn, Instagram, YouTube)
  - Accepted Payment Instruments (RuPay, UPI, Visa, MasterCard, Net Banking)
  - Official Toll-Free Helpline & Support Email
  - Download Mobile Apps with Google Play badges, vector QR codes, and Share-to-Clipboard actions
  - Full SEO Category Taxonomy Cloud (19 craft & handloom search keywords)
  - Official Copyright & Attribution to Ministry of Social Justice & Empowerment & Digital India Corporation (DIC)
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { LanguageSelector, showToast } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  function copyAppShareLink(type: 'buyer' | 'artisan') {
    const url = type === 'buyer'
      ? `${window.location.origin}/app/buyer`
      : `${window.location.origin.replace('5174', '5173')}/app/artisan`;
    
    if (navigator.clipboard) {
      navigator.clipboard.writeText(url);
      showToast({
        title: type === 'buyer' ? 'Buyer App Link Copied' : 'Artisan Voice App Link Copied',
        message: 'Download link copied to clipboard. Share with craft patrons and weavers.',
        variant: 'success'
      });
    }
  }

  const seoKeywords = [
    { label: 'Buy Handicraft and Handloom Items Online', href: '/catalog' },
    { label: 'Home and Living Products for Sale', href: '/search?category=home' },
    { label: 'Buy Mens Handloom and Handicraft Items', href: '/search?category=men' },
    { label: 'Handmade Furniture for Sale', href: '/search?category=furniture' },
    { label: 'Buy Home Decor Items Online', href: '/search?q=home+decor' },
    { label: 'Handloom Bedsheets and Cushions Online', href: '/search?q=bedsheets' },
    { label: 'Womens Handloom Sarees for Sale', href: '/search?category=women' },
    { label: 'Buy Western Wear for Women Online', href: '/search?q=western+wear' },
    { label: 'Beautiful Ethnic Wear for Women', href: '/search?q=ethnic+wear' },
    { label: 'Buy Mens Handmade Accessories', href: '/search?q=mens+accessories' },
    { label: 'Womens Handmade Accessories for Sale', href: '/search?q=womens+accessories' },
    { label: 'Handmade Floor Coverings Online', href: '/search?q=floor+coverings' },
    { label: 'Handmade Religious Items for Sale Online', href: '/search?q=religious' },
    { label: 'Buy Handmade Mens Footwear', href: '/search?q=mens+footwear' },
    { label: 'Buy Handmade Womens Footwear', href: '/search?q=womens+footwear' },
    { label: 'Handcrafted Lighting Items Online', href: '/search?q=lighting' },
    { label: 'Handmade Stationery Items for Sale', href: '/search?q=stationery' },
    { label: 'Buy Handmade Bags for Women', href: '/search?q=bags' },
    { label: 'Bath Accessories for Sale', href: '/search?q=bath' },
  ];
</script>

<footer class="buyer-footer" aria-label="Official Site Footer">
  <!-- 1. Top Trust Assurance Strip (6 Key Guarantees) -->
  <div class="footer-trust-strip">
    <div class="footer-trust-container">
      <!-- 1. Free Shipping -->
      <div class="trust-pill">
        <div class="trust-pill__icon">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <rect x="1" y="3" width="15" height="13"></rect>
            <polygon points="16 8 20 8 23 11 23 16 16 16 16 8"></polygon>
            <circle cx="5.5" cy="18.5" r="2.5"></circle>
            <circle cx="18.5" cy="18.5" r="2.5"></circle>
          </svg>
        </div>
        <div class="trust-pill__content">
          <span class="trust-pill__title">Free Shipping</span>
          <span class="trust-pill__sub">Orders above ₹999</span>
        </div>
      </div>

      <!-- 2. 100% Purchase Protection -->
      <div class="trust-pill">
        <div class="trust-pill__icon">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="4" width="18" height="16" rx="2"></rect>
            <line x1="7" y1="8" x2="17" y2="8"></line>
            <line x1="7" y1="12" x2="13" y2="12"></line>
            <circle cx="16.5" cy="14.5" r="2.5"></circle>
            <path d="m15.5 14.5 1 1 2-2"></path>
          </svg>
        </div>
        <div class="trust-pill__content">
          <span class="trust-pill__title">100% Purchase Protection</span>
          <span class="trust-pill__sub">Escrow Authenticity</span>
        </div>
      </div>

      <!-- 3. Secure Payment -->
      <div class="trust-pill">
        <div class="trust-pill__icon">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
            <path d="m9 12 2 2 4-4"></path>
          </svg>
        </div>
        <div class="trust-pill__content">
          <span class="trust-pill__title">Secure Payment</span>
          <span class="trust-pill__sub">256-Bit Encrypted</span>
        </div>
      </div>

      <!-- 4. Assured Quality -->
      <div class="trust-pill">
        <div class="trust-pill__icon">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="8" r="6"></circle>
            <path d="m15.4 12.5 2.1 7.5-5.5-3-5.5 3 2.1-7.5"></path>
          </svg>
        </div>
        <div class="trust-pill__content">
          <span class="trust-pill__title">Assured Quality</span>
          <span class="trust-pill__sub">Certified GI Heritage</span>
        </div>
      </div>

      <!-- 5. Handcrafted Products -->
      <div class="trust-pill">
        <div class="trust-pill__icon">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 11V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0"></path>
            <path d="M14 10V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v2"></path>
            <path d="M10 10.5V6a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v8"></path>
            <path d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15"></path>
          </svg>
        </div>
        <div class="trust-pill__content">
          <span class="trust-pill__title">Handcrafted Products</span>
          <span class="trust-pill__sub">0% Machine Imitations</span>
        </div>
      </div>

      <!-- 6. Best Price Promise -->
      <div class="trust-pill">
        <div class="trust-pill__icon">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="9"></circle>
            <path d="M8 8h8M8 12h5.5M10.5 12a3 3 0 0 0 3-3H8M11 15l4 5"></path>
          </svg>
        </div>
        <div class="trust-pill__content">
          <span class="trust-pill__title">Best Price Promise</span>
          <span class="trust-pill__sub">Direct DBT to Weavers</span>
        </div>
      </div>
    </div>
  </div>

  <!-- 2. Main Navigation Directory Grid -->
  <div class="footer-main">
    <div class="footer-main-container">
      
      <!-- Column 1: Resources -->
      <div class="footer-col">
        <h3 class="footer-col-heading">Resources</h3>
        <ul class="footer-nav-list" role="list">
          <li><a href="/privacy">Privacy and Cookie Policy</a></li>
          <li><a href="/catalog">Search Terms</a></li>
          <li><a href="/#faq">FAQs</a></li>
          <li><a href="/catalog">Advanced Search</a></li>
          <li><a href="/orders">Orders and Returns</a></li>
          <li><a href="/account">Contact Us</a></li>
          <li><a href="/feed">RSS Feed</a></li>
          <li><a href="/orders">Return Requests</a></li>
          <li><a href="http://localhost:5173" target="_blank" rel="noopener noreferrer">Vendor & Artisan Login</a></li>
          <li><a href="http://localhost:5173/onboarding" target="_blank" rel="noopener noreferrer">Vendor Registration</a></li>
          <li><a href="/case-studies">Indiahandmade & Guild Blog</a></li>
          <li><a href="/feed">Vendor Help & Guides</a></li>
        </ul>
      </div>

      <!-- Column 2: Policies & Info -->
      <div class="footer-col">
        <h3 class="footer-col-heading">Policies & Info</h3>
        <ul class="footer-nav-list" role="list">
          <li><a href="/about">About Kalakriti & Indiahandmade</a></li>
          <li><a href="/#faq">FAQs & Inquiries</a></li>
          <li><a href="/case-studies">Media & Craft Gallery</a></li>
          <li><a href="/privacy#grievance">Grievance Redressal Mechanism</a></li>
          <li><a href="/terms#disclaimer">Legal Disclaimer</a></li>
          <li><a href="/terms#refunds">Returns, Cancel, Refund Policy</a></li>
          <li><a href="/privacy">Privacy Policy (DPDP Act 2023)</a></li>
          <li><a href="/terms#shipping">Shipping & Transit Policy</a></li>
          <li><a href="/terms">Terms of Use (Buyers)</a></li>
          <li><a href="/terms#sellers">Terms of Use (Sellers & Weavers)</a></li>
        </ul>
      </div>

      <!-- Column 3: Regional Corridors & Procurement -->
      <div class="footer-col">
        <h3 class="footer-col-heading">Craft Corridors & B2B</h3>
        <ul class="footer-nav-list" role="list">
          <li><a href="/search?belt=north">Northern Plains & Foothills</a></li>
          <li><a href="/search?belt=west">Western Deserts & Kutch Rann</a></li>
          <li><a href="/search?belt=south">Deccan & Southern Temple Looms</a></li>
          <li><a href="/search?belt=east">Eastern & Assam Loomlands</a></li>
          <li><a href="/search?belt=central">Central Heartland & Tribal Bell Metal</a></li>
          <li><a href="/gi-tagged">All 74 Certified GI Clusters</a></li>
          <li><a href="/bulk-order">Institutional & Summit Procurement</a></li>
          <li><a href="/bulk-order?preset=hospitality">Handloom Hospitality Linen</a></li>
          <li><a href="/bulk-order?preset=corporate">Eco-Festive Corporate Gifting</a></li>
          <li><a href="/verify">Cryptographic Provenance Terminal</a></li>
        </ul>
      </div>

      <!-- Column 4: Social Media, Payments & App Downloads -->
      <div class="footer-col footer-col--actions">
        
        <!-- Social Media Row -->
        <div class="footer-block">
          <h3 class="footer-col-heading">Social Media</h3>
          <div class="social-links-row" aria-label="Official Social Channels">
            <!-- Facebook -->
            <a href="https://facebook.com" target="_blank" rel="noopener noreferrer" class="social-btn" aria-label="Facebook">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M22 12c0-5.52-4.48-10-10-10S2 6.48 2 12c0 4.84 3.44 8.87 8 9.8V15H8v-3h2V9.5C10 7.57 11.57 6 13.5 6H16v3h-2c-.55 0-1 .45-1 1v2h3v3h-3v6.95C18.05 21.45 22 17.19 22 12z"/>
              </svg>
            </a>
            <!-- Twitter / X -->
            <a href="https://x.com" target="_blank" rel="noopener noreferrer" class="social-btn" aria-label="Twitter X">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
              </svg>
            </a>
            <!-- LinkedIn -->
            <a href="https://linkedin.com" target="_blank" rel="noopener noreferrer" class="social-btn" aria-label="LinkedIn">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M19 3a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h14m-.5 15.5v-5.3a3.26 3.26 0 0 0-3.26-3.26c-.85 0-1.84.52-2.28 1.3v-1.11h-2.79v8.37h2.79v-4.93c0-.77.62-1.4 1.39-1.4a1.4 1.4 0 0 1 1.4 1.4v4.93h2.75M6.88 8.56a1.68 1.68 0 0 0 1.68-1.68c0-.93-.75-1.69-1.68-1.69a1.69 1.69 0 0 0-1.69 1.69c0 .93.76 1.68 1.69 1.68m1.39 9.94v-8.37H5.5v8.37h2.77z"/>
              </svg>
            </a>
            <!-- Instagram -->
            <a href="https://instagram.com" target="_blank" rel="noopener noreferrer" class="social-btn" aria-label="Instagram">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838c-3.403 0-6.162 2.759-6.162 6.162s2.759 6.163 6.162 6.163 6.162-2.759 6.162-6.163c0-3.403-2.759-6.162-6.162-6.162zm0 10.162c-2.209 0-4-1.79-4-4 0-2.209 1.791-4 4-4s4 1.791 4 4c0 2.21-1.791 4-4 4zm6.406-11.845c-.796 0-1.441.645-1.441 1.44s.645 1.44 1.441 1.44c.795 0 1.439-.645 1.439-1.44s-.644-1.44-1.439-1.44z"/>
              </svg>
            </a>
            <!-- YouTube -->
            <a href="https://youtube.com" target="_blank" rel="noopener noreferrer" class="social-btn" aria-label="YouTube">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/>
              </svg>
            </a>
          </div>
        </div>

        <!-- We Accept Payment Instruments -->
        <div class="footer-block">
          <h3 class="footer-col-heading">We Accept</h3>
          <div class="payment-badges-grid">
            <span class="pay-badge pay-badge--card">💳 Credit Card</span>
            <span class="pay-badge pay-badge--debit">💳 Debit Card</span>
            <span class="pay-badge pay-badge--netbank">🏦 Net Banking</span>
            <span class="pay-badge pay-badge--upi">⚡ UPI / RuPay</span>
          </div>
        </div>

        <!-- Direct Contact Strip -->
        <div class="footer-contact-box">
          <div class="contact-row">
            <span class="contact-label">Tollfree Helpline:</span>
            <a href="tel:1800112026" class="contact-value">1800-11-2026</a>
          </div>
          <div class="contact-row">
            <span class="contact-label">Direct Email:</span>
            <a href="mailto:care@kalakriti.gov.in" class="contact-value">care@kalakriti.gov.in</a>
          </div>
        </div>

        <!-- Download Apps Dual QR Box -->
        <div class="app-download-section">
          
          <!-- Buyer App Card -->
          <div class="app-qr-card">
            <div class="app-qr-card__meta">
              <span class="app-qr-card__label">Download Buyer App</span>
              <div class="google-play-badge">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                  <path d="m3.609 1.814 11.275 11.275L3.609 24.364A2.08 2.08 0 0 1 2 22.427V3.75c0-.853.51-1.603 1.609-1.936zM16.31 14.515 19.4 12.73a1.9 1.9 0 0 0 0-3.3l-3.09-1.785-2.148 2.148 2.148 2.148zM4.938 1.05 15.02 11.133l-2.148 2.148L3.25 3.656a2.02 2.02 0 0 1 1.688-2.606zM15.02 13.025l-10.082 10.08a2.02 2.02 0 0 1-1.688-2.606l9.622-9.622 2.148 2.148z"/>
                </svg>
                <span>Google Play</span>
              </div>
              <button 
                type="button" 
                class="app-share-btn" 
                onclick={() => copyAppShareLink('buyer')}
                title="Share Buyer App download link"
              >
                <Icon name="share" size="0.85rem" />
                <span>Share</span>
              </button>
            </div>
            <!-- Scalable Crisp Vector QR Code -->
            <div class="qr-preview-box">
              <svg viewBox="0 0 100 100" class="qr-svg" aria-label="QR Code to Download Buyer App">
                <!-- Outer Border -->
                <rect width="100" height="100" fill="#ffffff" rx="6" />
                <!-- Corner 1 (Top Left) -->
                <rect x="10" y="10" width="24" height="24" fill="#1c1917" rx="3" />
                <rect x="14" y="14" width="16" height="16" fill="#ffffff" rx="1.5" />
                <rect x="18" y="18" width="8" height="8" fill="#c65d3b" rx="1" />
                <!-- Corner 2 (Top Right) -->
                <rect x="66" y="10" width="24" height="24" fill="#1c1917" rx="3" />
                <rect x="70" y="14" width="16" height="16" fill="#ffffff" rx="1.5" />
                <rect x="74" y="18" width="8" height="8" fill="#c65d3b" rx="1" />
                <!-- Corner 3 (Bottom Left) -->
                <rect x="10" y="66" width="24" height="24" fill="#1c1917" rx="3" />
                <rect x="14" y="70" width="16" height="16" fill="#ffffff" rx="1.5" />
                <rect x="18" y="74" width="8" height="8" fill="#c65d3b" rx="1" />
                <!-- Data Blocks -->
                <rect x="38" y="12" width="6" height="6" fill="#1c1917" />
                <rect x="48" y="12" width="6" height="6" fill="#1c1917" />
                <rect x="38" y="24" width="6" height="6" fill="#1c1917" />
                <rect x="48" y="24" width="8" height="8" fill="#c65d3b" />
                <rect x="12" y="38" width="6" height="6" fill="#1c1917" />
                <rect x="24" y="38" width="8" height="6" fill="#1c1917" />
                <rect x="38" y="38" width="10" height="10" fill="#1c1917" />
                <rect x="54" y="38" width="6" height="6" fill="#1c1917" />
                <rect x="66" y="38" width="10" height="6" fill="#1c1917" />
                <rect x="80" y="38" width="8" height="8" fill="#1c1917" />
                <rect x="38" y="54" width="6" height="8" fill="#1c1917" />
                <rect x="48" y="54" width="8" height="6" fill="#1c1917" />
                <rect x="60" y="54" width="6" height="6" fill="#c65d3b" />
                <rect x="72" y="54" width="8" height="8" fill="#1c1917" />
                <rect x="84" y="54" width="6" height="6" fill="#1c1917" />
                <rect x="38" y="68" width="10" height="6" fill="#1c1917" />
                <rect x="52" y="68" width="6" height="8" fill="#1c1917" />
                <rect x="66" y="68" width="8" height="8" fill="#1c1917" />
                <rect x="80" y="68" width="8" height="8" fill="#1c1917" />
                <rect x="38" y="80" width="8" height="8" fill="#1c1917" />
                <rect x="52" y="80" width="10" height="8" fill="#1c1917" />
                <rect x="68" y="80" width="6" height="8" fill="#1c1917" />
                <rect x="80" y="80" width="8" height="8" fill="#c65d3b" />
              </svg>
            </div>
          </div>

          <!-- Seller / Artisan Voice App Card -->
          <div class="app-qr-card">
            <div class="app-qr-card__meta">
              <span class="app-qr-card__label">Download Seller App</span>
              <div class="google-play-badge">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                  <path d="m3.609 1.814 11.275 11.275L3.609 24.364A2.08 2.08 0 0 1 2 22.427V3.75c0-.853.51-1.603 1.609-1.936zM16.31 14.515 19.4 12.73a1.9 1.9 0 0 0 0-3.3l-3.09-1.785-2.148 2.148 2.148 2.148zM4.938 1.05 15.02 11.133l-2.148 2.148L3.25 3.656a2.02 2.02 0 0 1 1.688-2.606zM15.02 13.025l-10.082 10.08a2.02 2.02 0 0 1-1.688-2.606l9.622-9.622 2.148 2.148z"/>
                </svg>
                <span>Google Play</span>
              </div>
              <button 
                type="button" 
                class="app-share-btn" 
                onclick={() => copyAppShareLink('artisan')}
                title="Share Artisan Voice App download link"
              >
                <Icon name="share" size="0.85rem" />
                <span>Share</span>
              </button>
            </div>
            <!-- Scalable Crisp Vector QR Code -->
            <div class="qr-preview-box">
              <svg viewBox="0 0 100 100" class="qr-svg" aria-label="QR Code to Download Artisan Seller App">
                <!-- Outer Border -->
                <rect width="100" height="100" fill="#ffffff" rx="6" />
                <!-- Corner 1 (Top Left) -->
                <rect x="10" y="10" width="24" height="24" fill="#1c1917" rx="3" />
                <rect x="14" y="14" width="16" height="16" fill="#ffffff" rx="1.5" />
                <rect x="18" y="18" width="8" height="8" fill="#1e3a8a" rx="1" />
                <!-- Corner 2 (Top Right) -->
                <rect x="66" y="10" width="24" height="24" fill="#1c1917" rx="3" />
                <rect x="70" y="14" width="16" height="16" fill="#ffffff" rx="1.5" />
                <rect x="74" y="18" width="8" height="8" fill="#1e3a8a" rx="1" />
                <!-- Corner 3 (Bottom Left) -->
                <rect x="10" y="66" width="24" height="24" fill="#1c1917" rx="3" />
                <rect x="14" y="70" width="16" height="16" fill="#ffffff" rx="1.5" />
                <rect x="18" y="74" width="8" height="8" fill="#1e3a8a" rx="1" />
                <!-- Data Blocks -->
                <rect x="40" y="14" width="8" height="6" fill="#1c1917" />
                <rect x="52" y="14" width="6" height="6" fill="#1c1917" />
                <rect x="40" y="24" width="6" height="6" fill="#1e3a8a" />
                <rect x="50" y="24" width="8" height="8" fill="#1c1917" />
                <rect x="14" y="40" width="6" height="6" fill="#1c1917" />
                <rect x="24" y="40" width="6" height="6" fill="#1c1917" />
                <rect x="36" y="40" width="10" height="10" fill="#1e3a8a" />
                <rect x="50" y="40" width="8" height="6" fill="#1c1917" />
                <rect x="64" y="40" width="10" height="6" fill="#1c1917" />
                <rect x="80" y="40" width="8" height="8" fill="#1c1917" />
                <rect x="38" y="54" width="6" height="8" fill="#1c1917" />
                <rect x="48" y="54" width="8" height="6" fill="#1c1917" />
                <rect x="62" y="54" width="8" height="8" fill="#1e3a8a" />
                <rect x="74" y="54" width="8" height="6" fill="#1c1917" />
                <rect x="84" y="54" width="6" height="6" fill="#1c1917" />
                <rect x="40" y="68" width="8" height="6" fill="#1c1917" />
                <rect x="52" y="68" width="6" height="8" fill="#1c1917" />
                <rect x="66" y="68" width="8" height="8" fill="#1c1917" />
                <rect x="80" y="68" width="8" height="8" fill="#1c1917" />
                <rect x="38" y="80" width="8" height="8" fill="#1e3a8a" />
                <rect x="50" y="80" width="10" height="8" fill="#1c1917" />
                <rect x="66" y="80" width="6" height="8" fill="#1c1917" />
                <rect x="80" y="80" width="8" height="8" fill="#1c1917" />
              </svg>
            </div>
          </div>

        </div>

      </div>

    </div>
  </div>

  <!-- 3. SEO Category Keyword Cloud / Craft Directory Ribbon -->
  <div class="footer-seo-cloud">
    <div class="footer-seo-container">
      <nav aria-label="Craft and Handloom Directory Keywords" class="seo-keywords-nav">
        {#each seoKeywords as item, i}
          <a href={item.href} class="seo-keyword-link">{item.label}</a>
          {#if i < seoKeywords.length - 1}
            <span class="seo-keyword-sep" aria-hidden="true">|</span>
          {/if}
        {/each}
      </nav>
    </div>
  </div>

  <!-- 4. Official Government Copyright & Attribution Ribbon -->
  <div class="footer-bottom">
    <div class="footer-bottom-container">
      <div class="bottom-legal">
        <span class="legal-copy">
          Copyright © 2026 <strong>Ministry of Social Justice & Empowerment</strong>, Government of India. All rights reserved.
        </span>
        <span class="dot-sep" aria-hidden="true">•</span>
        <span class="legal-dev">
          Designed &amp; Developed by : <strong>Digital India Corporation (DIC)</strong>
        </span>
        <span class="dot-sep" aria-hidden="true">•</span>
        <span class="legal-host">
          Hosted on National NIC Cloud Infrastructure (GIGW 3.0 &amp; WCAG 2.1 AAA Compliant)
        </span>
      </div>

      <div class="bottom-controls">
        <LanguageSelector />
      </div>
    </div>
  </div>
</footer>

<style>
  /* Base Footer Container — Warm Antique Walnut & Deep Charcoal Theme */
  .buyer-footer {
    background-color: #171513;
    color: #a8a29e;
    font-size: 0.8125rem;
    line-height: 1.5;
    border-block-start: 1px solid rgba(255, 255, 255, 0.08);
    margin-block-start: var(--k-space-8);
  }

  /* 1. Trust Assurance Strip */
  .footer-trust-strip {
    background-color: #1f1c19;
    border-block-end: 1px solid rgba(255, 255, 255, 0.07);
    padding-block: 1.25rem;
  }

  .footer-trust-container {
    max-inline-size: 82rem;
    margin-inline: auto;
    padding-inline: 1.5rem;
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 1rem;
  }

  @media (max-width: 72rem) {
    .footer-trust-container {
      grid-template-columns: repeat(3, 1fr);
      row-gap: 1.25rem;
    }
  }

  @media (max-width: 42rem) {
    .footer-trust-container {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 26rem) {
    .footer-trust-container {
      grid-template-columns: 1fr;
    }
  }

  .trust-pill {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .trust-pill__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.5rem;
    aspect-ratio: 1;
    border-radius: 50%;
    background-color: rgba(198, 93, 59, 0.14);
    color: #e0805a;
    flex-shrink: 0;
    transition: transform 0.2s ease, background-color 0.2s ease;
  }

  .trust-pill:hover .trust-pill__icon {
    transform: scale(1.06);
    background-color: rgba(198, 93, 59, 0.22);
  }

  .trust-pill__content {
    display: flex;
    flex-direction: column;
    min-inline-size: 0;
  }

  .trust-pill__title {
    font-size: 0.78rem;
    font-weight: 700;
    color: #f5f5f4;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .trust-pill__sub {
    font-size: 0.69rem;
    color: #a8a29e;
    white-space: nowrap;
  }

  /* 2. Main Navigation Directory */
  .footer-main {
    padding-block: 2.75rem 2rem;
  }

  .footer-main-container {
    max-inline-size: 82rem;
    margin-inline: auto;
    padding-inline: 1.5rem;
    display: grid;
    grid-template-columns: 1.1fr 1.1fr 1.2fr 1.6fr;
    gap: 2.5rem;
  }

  @media (max-width: 68rem) {
    .footer-main-container {
      grid-template-columns: 1fr 1fr;
      gap: 2rem;
    }
  }

  @media (max-width: 38rem) {
    .footer-main-container {
      grid-template-columns: 1fr;
      gap: 2rem;
    }
  }

  .footer-col {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
  }

  .footer-col--actions {
    gap: 1.5rem;
  }

  .footer-col-heading {
    font-size: 0.8125rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #f5f5f4;
    margin: 0;
    padding-block-end: 0.35rem;
    border-block-end: 1px solid rgba(255, 255, 255, 0.1);
    inline-size: fit-content;
  }

  .footer-nav-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .footer-nav-list a {
    color: #a8a29e;
    text-decoration: none;
    font-size: 0.775rem;
    line-height: 1.4;
    transition: color 0.15s ease, padding-inline-start 0.15s ease;
  }

  .footer-nav-list a:hover {
    color: #f5f5f4;
    text-decoration: underline;
    text-underline-offset: 3px;
  }

  /* Social Media Links */
  .social-links-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-block-start: 0.5rem;
  }

  .social-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    inline-size: 2.25rem;
    aspect-ratio: 1;
    border-radius: 6px;
    background-color: rgba(255, 255, 255, 0.06);
    color: #d6d3d1;
    transition: background-color 0.2s ease, color 0.2s ease, transform 0.2s ease;
  }

  .social-btn:hover {
    background-color: #c65d3b;
    color: #ffffff;
    transform: translateY(-2px);
  }

  /* We Accept Payment Badges */
  .payment-badges-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem;
    margin-block-start: 0.5rem;
  }

  .pay-badge {
    font-size: 0.69rem;
    font-weight: 600;
    padding: 0.25rem 0.55rem;
    border-radius: 4px;
    background-color: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #e7e5e4;
    letter-spacing: 0.02em;
  }

  /* Direct Helpline Strip */
  .footer-contact-box {
    background-color: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    padding: 0.75rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .contact-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.75rem;
  }

  .contact-label {
    color: #a8a29e;
  }

  .contact-value {
    color: #e0805a;
    font-weight: 600;
    text-decoration: none;
  }

  .contact-value:hover {
    text-decoration: underline;
  }

  /* Download App Dual Cards */
  .app-download-section {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }

  @media (max-width: 40rem) {
    .app-download-section {
      grid-template-columns: 1fr;
    }
  }

  .app-qr-card {
    background-color: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.09);
    border-radius: 8px;
    padding: 0.75rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
  }

  .app-qr-card__meta {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    min-inline-size: 0;
  }

  .app-qr-card__label {
    font-size: 0.72rem;
    font-weight: 600;
    color: #f5f5f4;
    line-height: 1.2;
  }

  .google-play-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    background-color: #000000;
    color: #ffffff;
    font-size: 0.65rem;
    font-weight: 600;
    padding: 0.2rem 0.45rem;
    border-radius: 4px;
    border: 1px solid rgba(255, 255, 255, 0.15);
    inline-size: fit-content;
  }

  .app-share-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    background: none;
    border: 1px solid rgba(255, 255, 255, 0.18);
    color: #d6d3d1;
    font-size: 0.68rem;
    font-weight: 500;
    padding: 0.18rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.15s ease;
    inline-size: fit-content;
  }

  .app-share-btn:hover {
    background-color: rgba(255, 255, 255, 0.1);
    color: #ffffff;
    border-color: rgba(255, 255, 255, 0.3);
  }

  .qr-preview-box {
    inline-size: 3.5rem;
    aspect-ratio: 1;
    flex-shrink: 0;
    border-radius: 4px;
    overflow: hidden;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.4);
  }

  .qr-svg {
    inline-size: 100%;
    block-size: 100%;
    display: block;
  }

  /* 3. SEO Category Keyword Cloud Ribbon */
  .footer-seo-cloud {
    background-color: #131110;
    border-block-start: 1px solid rgba(255, 255, 255, 0.05);
    border-block-end: 1px solid rgba(255, 255, 255, 0.05);
    padding-block: 1rem;
  }

  .footer-seo-container {
    max-inline-size: 82rem;
    margin-inline: auto;
    padding-inline: 1.5rem;
  }

  .seo-keywords-nav {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    gap: 0.45rem 0.6rem;
    line-height: 1.6;
    text-align: center;
  }

  .seo-keyword-link {
    font-size: 0.7125rem;
    color: #78716c;
    text-decoration: none;
    transition: color 0.15s ease;
  }

  .seo-keyword-link:hover {
    color: #e0805a;
    text-decoration: underline;
  }

  .seo-keyword-sep {
    color: rgba(255, 255, 255, 0.1);
    font-size: 0.65rem;
  }

  /* 4. Bottom Government Copyright & Attribution Ribbon */
  .footer-bottom {
    background-color: #0f0d0c;
    padding-block: 1.25rem;
  }

  .footer-bottom-container {
    max-inline-size: 82rem;
    margin-inline: auto;
    padding-inline: 1.5rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 1rem;
    font-size: 0.72rem;
    color: #78716c;
  }

  .bottom-legal {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
    line-height: 1.5;
  }

  .bottom-legal strong {
    color: #d6d3d1;
    font-weight: 600;
  }

  .dot-sep {
    color: rgba(255, 255, 255, 0.12);
  }

  .bottom-controls {
    flex-shrink: 0;
  }
</style>
