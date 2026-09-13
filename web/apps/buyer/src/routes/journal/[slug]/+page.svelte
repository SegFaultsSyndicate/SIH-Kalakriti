<!--
  apps/buyer/src/routes/journal/[slug]/+page.svelte

  Virasat Cultural Journal essay pages. The home page's "Chronicles of
  Living Heritage" rail used to send "Read Heritage Essay" to a generic
  /search?q=... link -- there was no essay to read. These are the three
  essays it teases (ajrakh, patola, dhokra), same static-editorial pattern
  as /case-studies.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';

  const t = $derived(locale.t);

  interface Essay {
    slug: string;
    title: string;
    author: string;
    readTime: string;
    cluster: string;
    region: string;
    heroImage: string;
    lead: string;
    body: string[];
    quote: { text: string; author: string };
  }

  const ESSAYS: Essay[] = [
    {
      slug: 'ajrakh',
      title: 'The Alchemist of Dhamadka: 10 Generations of Ajrakh',
      author: 'Dr. Radhika Sen · Anthropologist',
      readTime: '6 min read',
      cluster: 'Dhamadka Cluster',
      region: 'Kutch, Gujarat',
      heroImage: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_03.jpeg',
      lead:
        'How the Khatri master dyers of Kutch sustain 16 chemical-free natural resist phases along the seasonal riverbanks of Dhamadka.',
      body: [
        'Every Ajrakh print begins with water, not dye. The Khatri families of Dhamadka wash raw cotton in the seasonal Saran river for days before a single block ever touches the cloth, because the mineral-rich riverbed water is what lets harde (myrobalan) mordant bind evenly across the weave -- a step no synthetic pre-treatment has ever fully replaced.',
        'The 16-stage process alternates resist and dye: geru clay and gum resist blocks hold back indigo in one pass, iron-rust and tamarind-seed paste hold back alizarin red in the next. A single stole passes through a dyer\'s hands, and the river, more than a dozen times before the geometric star-and-vine motifs -- passed down through pattern blocks some families have carved for three generations -- finally emerge in full.',
        'Synthetic screen-printed imitations undercut this work by a wide margin, because they collapse sixteen slow stages into one fast one. What they cannot imitate is the way natural indigo continues to deepen for months after the cloth leaves the workshop, or the faint mineral scent of harde that authentic Ajrakh keeps for years.',
        'Today roughly forty Khatri families still practise the full sequence in Dhamadka and neighbouring Ajrakhpur. Direct-to-buyer sales -- rather than consignment through urban middlemen -- have let several of those families reinvest in the one resource the craft cannot do without: clean, mineral-rich river water for the next generation\'s vats.',
      ],
      quote: {
        text: 'People think Ajrakh is a print. It is a river, a mineral, and sixteen kinds of patience, worn as cloth.',
        author: 'A Dhamadka master dyer, on the practice his grandfather taught him',
      },
    },
    {
      slug: 'patola',
      title: 'The Secret Math of Double Ikat: Patan Patola Weaving',
      author: 'V. Swaminathan · Master Guild Historian',
      readTime: '8 min read',
      cluster: 'Patan Guild',
      region: 'Patan, Gujarat',
      heroImage: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
      lead:
        'Decoding the sacred geometric algorithms and double-resist warp alignments of Gujarat\'s legendary 800-year Patan guild.',
      body: [
        'Single ikat resists and dyes one set of threads -- either the warp or the weft -- before weaving. Patan\'s double ikat resists and dyes both, independently, to a pattern so precisely calculated that when the two are finally interlaced on the loom, a single unbroken motif appears on both faces of the cloth, with no right or wrong side.',
        'The calculation is done entirely by hand and memory. A master binder maps out where every colour must fall along thousands of individual warp and weft threads before a single one is dyed, because a single thread bound even a few millimetres out of place will blur the motif when it finally meets its partner on the loom -- there is no correcting it afterward.',
        'A single Patola saree can take between six months and a year to complete, and the Salvi families of Patan have kept the sequence within roughly a dozen guild households for eight centuries, largely because the binding calculations were never written down -- they are taught hand to hand, loom to loom.',
        'What buyers are paying for, when they pay for a real Patola, is not silk or dye but a completed calculation: tens of thousands of correctly bound and dyed threads that had exactly one chance to align.',
      ],
      quote: {
        text: 'We do not draw the pattern on the cloth. We calculate it into the thread before the cloth exists.',
        author: 'A Patan Salvi guild weaver',
      },
    },
    {
      slug: 'dhokra',
      title: 'Forest Furnaces: Bastar\'s Lost-Wax Bronze',
      author: 'Anjali Kujur · Field Researcher',
      readTime: '7 min read',
      cluster: 'Bastar Ghadwa',
      region: 'Kondagaon, Bastar, Chhattisgarh',
      heroImage: '/craft-images/metalwork/dhokra-casting.jpg',
      lead:
        'Inside the forest furnaces of Bastar where Ghadwa metalsmiths transform wild honey wax, red clay, and scrap bronze into animist deities.',
      body: [
        'Dhokra casting begins with a core of river clay and rice husk, shaped roughly into the figure\'s body. Over that core, the sculptor threads fine ropes of beeswax -- gathered from wild forest hives -- coiling and pressing each strand by hand to build every surface detail the final bronze will carry, since whatever is not present in the wax cannot appear in the metal.',
        'A second clay coat is packed over the wax, with narrow channels left for molten metal to enter and air to escape. The whole mould is fired: the wax melts and runs out through the channels -- "lost" -- leaving a hollow negative in exactly the shape the sculptor built by hand, into which molten bronze or bell metal alloy is poured.',
        'The clay mould must be broken to free the finished piece, which means every single Dhokra figure is unique -- there is no second casting from the same mould, no production run. What looks like a repeated motif across a market stall is, in fact, dozens of individually wax-threaded, individually broken-open originals.',
        'The technique is one of the oldest continuously practised metal casting methods in the world, tracing back over four thousand years to the Indus Valley, and today survives largely through tribal Ghadwa metalsmith families around Kondagaon, for whom the bull, the horse, and the forest deity remain the same figures their ancestors cast.',
      ],
      quote: {
        text: 'Once the wax is gone, there is no going back to fix it. Every figure gets one chance to be right.',
        author: 'A Kondagaon Ghadwa metalsmith',
      },
    },
  ];

  const slug = $derived(page.params.slug ?? '');
  const essay = $derived(ESSAYS.find((e) => e.slug === slug));

  const breadcrumbs = $derived([
    { label: t('nav.home') || 'Home', href: '/' },
    { label: 'The Virasat Journal', href: '/#journal' },
    { label: essay?.title ?? 'Essay' },
  ]);
</script>

<svelte:head>
  <title>{essay ? `${essay.title} — Kalakriti Journal` : 'Essay — Kalakriti Journal'}</title>
  {#if essay}
    <meta name="description" content={essay.lead} />
  {/if}
</svelte:head>

{#if !essay}
  <div class="essay-page">
    <div class="essay-container">
      <Breadcrumbs items={[{ label: t('nav.home') || 'Home', href: '/' }, { label: 'Essay not found' }]} />
      <h1>Essay not found</h1>
      <p><a href="/">Return home</a></p>
    </div>
  </div>
{:else}
  <div class="essay-page">
    <div class="essay-container">
      <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

      <header class="essay-header">
        <span class="essay-kicker">THE VIRASAT JOURNAL · {essay.cluster.toUpperCase()}</span>
        <h1 class="essay-title">{essay.title}</h1>
        <p class="essay-lead">{essay.lead}</p>
        <div class="essay-meta">
          <span>{essay.author}</span>
          <span class="dot">•</span>
          <span>{essay.readTime}</span>
          <span class="dot">•</span>
          <span>{essay.region}</span>
        </div>
      </header>

      <div class="essay-hero">
        <img src={essay.heroImage} alt={essay.title} loading="lazy" />
      </div>

      <div class="essay-body">
        {#each essay.body as paragraph, i (i)}
          <p>{paragraph}</p>
        {/each}
      </div>

      <blockquote class="essay-quote">
        <p>"{essay.quote.text}"</p>
        <footer>{essay.quote.author}</footer>
      </blockquote>

      <div class="essay-actions">
        <a href="/" class="essay-back-link">
          <Icon name="arrow-right" size="0.85rem" />
          <span>Back to Kalakriti Home</span>
        </a>
      </div>
    </div>
  </div>
{/if}

<style>
  .essay-page {
    padding-block: var(--k-space-6) var(--k-space-12);
    background: var(--k-surface-base);
  }

  .essay-container {
    max-inline-size: 42rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .essay-header {
    margin-block: var(--k-space-6) var(--k-space-5);
  }

  .essay-kicker {
    display: inline-block;
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    color: var(--k-accent-primary-text);
    margin-block-end: var(--k-space-2);
  }

  .essay-title {
    font-family: var(--k-font-display, serif);
    font-size: clamp(1.6rem, 4vw, 2.2rem);
    font-weight: 800;
    color: var(--k-text-primary);
    line-height: 1.2;
    margin: 0 0 var(--k-space-3);
  }

  .essay-lead {
    font-size: var(--k-text-md);
    color: var(--k-text-secondary);
    line-height: var(--k-leading-normal);
    margin: 0 0 var(--k-space-3);
  }

  .essay-meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .essay-meta .dot {
    opacity: 0.5;
  }

  .essay-hero {
    border-radius: var(--k-radius-lg);
    overflow: hidden;
    aspect-ratio: 16 / 9;
    margin-block-end: var(--k-space-6);
  }

  .essay-hero img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .essay-body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    font-size: var(--k-text-base);
    line-height: 1.75;
    color: var(--k-text-primary);
  }

  .essay-quote {
    margin: var(--k-space-7) 0;
    padding: var(--k-space-5);
    background: var(--k-surface-base);
    border-inline-start: 4px solid var(--k-border-accent);
    border-radius: 0 var(--k-radius-md) var(--k-radius-md) 0;
  }

  .essay-quote p {
    font-size: var(--k-text-md);
    font-style: italic;
    color: var(--k-stone-700);
    line-height: var(--k-leading-normal);
    margin: 0 0 var(--k-space-2);
  }

  .essay-quote footer {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .essay-actions {
    border-block-start: 1px solid var(--k-border-subtle);
    padding-block-start: var(--k-space-5);
  }

  .essay-back-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
    font-weight: 700;
    color: var(--k-accent-primary-text);
    text-decoration: none;
  }

  .essay-back-link :global(svg) {
    transform: rotate(180deg);
  }
</style>
