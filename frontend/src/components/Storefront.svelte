<script>
  import { onMount } from 'svelte';
  import { getMenu, getWallet, createOrder } from '../api.js';

  let { me, onlogout } = $props();

  const EMOJI = {
    MEAL: '🍛', POR: '🫓', CHK: '🍗', SNACK: '🍟',
    PUR: '🥘', DIH: '🍚', IDP: '🍜', CHP: '🍲',
  };
  const ART = ['#fff1e6', '#e8f5e9', '#fdecea', '#fff8e1', '#e3f2fd', '#f3e5f5'];

  let items = $state([]);
  let wallet = $state(null);
  let loading = $state(true);
  let loadError = $state('');

  let search = $state('');
  let section = $state('all');

  let cart = $state({});
  let cartOpen = $state(false);
  let placing = $state(false);
  let placeError = $state('');
  let placed = $state(null);

  const money = (paise) => paise % 100 === 0
    ? `₹${paise / 100}`
    : `₹${(paise / 100).toFixed(2)}`;

  function artwork(code) {
    let h = 0;
    for (const c of code) h = (h * 31 + c.charCodeAt(0)) >>> 0;
    return ART[h % ART.length];
  }
  const emoji = (code) => EMOJI[code] ?? '🍽';

  let visible = $derived(items.filter(i => {
    if (section === 'a' && i.code !== 'MEAL') return false;
    if (section === 'b' && i.code === 'MEAL') return false;
    const q = search.trim().toLowerCase();
    return !q || i.item_name.toLowerCase().includes(q) || i.code.toLowerCase().includes(q);
  }));
  let groupA = $derived(visible.filter(i => i.code === 'MEAL'));
  let groupB = $derived(visible.filter(i => i.code !== 'MEAL'));

  let lines = $derived(Object.entries(cart)
    .map(([code, qty]) => ({ item: items.find(i => i.code === code), code, qty }))
    .filter(l => l.item && l.qty > 0));
  let cartCount = $derived(lines.reduce((n, l) => n + l.qty, 0));
  let cartTotal = $derived(lines.reduce((n, l) => n + l.item.price_paise * l.qty, 0));
  let cartHasSpecial = $derived(lines.some(l => l.code !== 'MEAL'));

  function add(code) {
    cart = { ...cart, [code]: (cart[code] ?? 0) + 1 };
  }
  function dec(code) {
    const next = { ...cart };
    if ((next[code] ?? 0) <= 1) delete next[code];
    else next[code] -= 1;
    cart = next;
  }

  async function refreshWallet() {
    try { wallet = (await getWallet(me.id)).balance_paise; } catch { wallet = null; }
  }

  async function placeOrder() {
    if (!lines.length) return;
    placing = true;
    placeError = '';
    try {
      const res = await createOrder(me.id, lines.map(l => ({ code: l.code, quantity: l.qty })));
      placed = { id: res.id, total: res.total_paise, special: cartHasSpecial, mixed: cartHasSpecial && lines.some(l => l.code === 'MEAL') };
      cart = {};
      cartOpen = false;
      await refreshWallet();
    } catch (e) {
      placeError = e.message;
    } finally {
      placing = false;
    }
  }

  onMount(async () => {
    try {
      const menu = await getMenu();
      items = menu.items ?? [];
    } catch (e) {
      loadError = e.message;
    } finally {
      loading = false;
    }
    await refreshWallet();
  });
</script>

<div class="storefront">
  <header class="topbar">
    <div class="brand">CAMPUS <span>KITCHEN</span></div>
    <div class="top-actions">
      {#if wallet !== null}
        <span class="wallet" title="Wallet balance">₹{(wallet / 100).toFixed(2)}</span>
      {/if}
      <button class="ghost" onclick={onlogout}>Logout</button>
    </div>
  </header>

  <main class="content">
    <div class="hello">
      <h1>Hey, {me.name ?? me.roll_no} 👋</h1>
      <p>Skip the queue. Order, then collect at your counter.</p>
    </div>

    <div class="searchbox">
      <span aria-hidden="true">🔍</span>
      <input bind:value={search} placeholder="Search for meals, porotta, snacks…" aria-label="Search the menu" />
    </div>

    <div class="chips" role="group" aria-label="Filter by counter">
      <button class="chip" class:active={section === 'all'} onclick={() => section = 'all'}>All</button>
      <button class="chip" class:active={section === 'a'} onclick={() => section = 'a'}>Counter A · Meals</button>
      <button class="chip" class:active={section === 'b'} onclick={() => section = 'b'}>Counter B · Other</button>
    </div>

    {#if loading}
      <p class="state">Loading today's menu…</p>
    {:else if loadError}
      <p class="state error">Could not load the menu: {loadError}</p>
    {:else if !visible.length}
      <p class="state">Nothing matches that search.</p>
    {:else}
      {#if groupA.length && section !== 'b'}
        <section class="group">
          <div class="group-head">
            <h2>Counter A</h2>
            <span>Meals · served in seconds</span>
          </div>
          {#each groupA as item (item.id)}
            <article class="card">
              <div class="card-body">
                <h3>{item.item_name}</h3>
                <div class="price">{money(item.price_paise)}</div>
                <div class="meta"><code>{item.code}</code> · meal line</div>
                <div class="tagline">Collect at Counter A</div>
                {#if cart[item.code]}
                  <div class="stepper">
                    <button onclick={() => dec(item.code)} aria-label={`Remove one ${item.item_name}`}>−</button>
                    <span>{cart[item.code]}</span>
                    <button onclick={() => add(item.code)} aria-label={`Add one more ${item.item_name}`}>+</button>
                  </div>
                {:else}
                  <button class="add" onclick={() => add(item.code)}>ADD <span>+</span></button>
                {/if}
              </div>
              <div class="card-art" style="background:{artwork(item.code)}" aria-hidden="true">{emoji(item.code)}</div>
            </article>
          {/each}
        </section>
      {/if}

      {#if groupB.length && section !== 'a'}
        <section class="group">
          <div class="group-head">
            <h2>Counter B</h2>
            <span>Short orders · kitchen to counter</span>
          </div>
          {#each groupB as item (item.id)}
            <article class="card">
              <div class="card-body">
                <h3>{item.item_name}</h3>
                <div class="price">{money(item.price_paise)}</div>
                <div class="meta"><code>{item.code}</code> · made to order</div>
                <div class="tagline">Kitchen preps it, collect at Counter B</div>
                {#if cart[item.code]}
                  <div class="stepper">
                    <button onclick={() => dec(item.code)} aria-label={`Remove one ${item.item_name}`}>−</button>
                    <span>{cart[item.code]}</span>
                    <button onclick={() => add(item.code)} aria-label={`Add one more ${item.item_name}`}>+</button>
                  </div>
                {:else}
                  <button class="add" onclick={() => add(item.code)}>ADD <span>+</span></button>
                {/if}
              </div>
              <div class="card-art" style="background:{artwork(item.code)}" aria-hidden="true">{emoji(item.code)}</div>
            </article>
          {/each}
        </section>
      {/if}
    {/if}
  </main>

  {#if cartCount > 0}
    <div class="cartbar">
      <div class="cartbar-info">
        <strong>{cartCount} item{cartCount > 1 ? 's' : ''} · {money(cartTotal)}</strong>
        <span>Wallet ₹{wallet !== null ? (wallet / 100).toFixed(2) : '—'}</span>
      </div>
      <button class="primary" onclick={() => cartOpen = true}>View Cart <span aria-hidden="true">→</span></button>
    </div>
  {/if}
</div>

{#if cartOpen}
  <div class="overlay" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) cartOpen = false; }}>
    <div class="sheet" role="dialog" aria-modal="true" aria-label="Your cart">
      <div class="sheet-head">
        <h2>Your order</h2>
        <button class="ghost" onclick={() => cartOpen = false}>Close</button>
      </div>

      {#each lines as { item, code, qty }}
        <div class="line">
          <div>
            <strong>{item.item_name}</strong>
            <div class="line-sub">{money(item.price_paise)} each · {code}</div>
          </div>
          <div class="line-right">
            <div class="stepper small">
              <button onclick={() => dec(code)} aria-label={`Remove one ${item.item_name}`}>−</button>
              <span>{qty}</span>
              <button onclick={() => add(code)} aria-label={`Add one more ${item.item_name}`}>+</button>
            </div>
            <strong>{money(item.price_paise * qty)}</strong>
          </div>
        </div>
      {/each}

      <div class="bill">
        <div><span>Item total</span><strong>{money(cartTotal)}</strong></div>
        <div class="bill-total"><span>To pay from wallet</span><strong>{money(cartTotal)}</strong></div>
        <div class="wallet-line">Balance after order: {wallet !== null ? money(Math.max(0, wallet - cartTotal)) : '—'}</div>
      </div>

      {#if cartHasSpecial}
        <p class="hint">Includes kitchen items — they'll be ready at Counter B.</p>
      {/if}
      {#if placeError}
        <p class="error">{placeError}</p>
      {/if}

      <button class="primary wide" disabled={placing || cartTotal > (wallet ?? 0)} onclick={placeOrder}>
        {placing ? 'Placing…' : `Place order · ${money(cartTotal)}`}
      </button>
      {#if wallet !== null && cartTotal > wallet}
        <p class="error">Wallet balance is too low for this order.</p>
      {/if}
    </div>
  </div>
{/if}

{#if placed}
  <div class="overlay">
    <div class="sheet success" role="dialog" aria-modal="true" aria-label="Order placed">
      <div class="check">✓</div>
      <h2>Order placed</h2>
      <p class="token">Token <strong>#{placed.id}</strong></p>
      <p class="where">
        {#if placed.mixed}
          Counter A for the meal, Counter B for the rest.
        {:else if placed.special}
          Show this token at <strong>Counter B</strong>. The kitchen is packing it now.
        {:else}
          Show this token at <strong>Counter A</strong>.
        {/if}
      </p>
      <p class="paid">{money(placed.total)} paid from your wallet.</p>
      <button class="primary wide" onclick={() => placed = null}>Order something else</button>
    </div>
  </div>
{/if}

<style>
  .storefront {
    min-height: 100vh;
    background: #f7f7f7;
    color: #1b1b1b;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    padding-bottom: 6rem;
  }

  .topbar {
    position: sticky;
    top: 0;
    z-index: 5;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem 1.25rem;
    background: rgba(255, 255, 255, .92);
    backdrop-filter: blur(10px);
    border-bottom: 1px solid #ececec;
    max-width: 760px;
    margin: 0 auto;
  }
  .brand { font-size: 1.25rem; font-weight: 800; letter-spacing: -.02em; }
  .brand span { color: #2e8b57; }

  .top-actions { display: flex; align-items: center; gap: .6rem; }
  .wallet {
    font-size: .8rem;
    font-weight: 700;
    color: #1f6f45;
    background: #e8f5ee;
    border-radius: 999px;
    padding: .3rem .65rem;
  }
  .ghost {
    background: none;
    border: none;
    color: #777;
    font-size: .8rem;
    cursor: pointer;
    font-family: inherit;
  }
  .ghost:hover { color: #1b1b1b; }

  .content { max-width: 760px; margin: 0 auto; padding: 0 1.25rem; }

  .hello { padding: 1.25rem 0 .75rem; }
  .hello h1 { margin: 0 0 .25rem; font-size: 1.35rem; letter-spacing: -.02em; }
  .hello p { margin: 0; color: #7a7a7a; font-size: .85rem; }

  .searchbox {
    display: flex;
    align-items: center;
    gap: .5rem;
    background: #fff;
    border: 1px solid #e6e6e6;
    border-radius: 12px;
    padding: .65rem .8rem;
    margin: .5rem 0 .75rem;
  }
  .searchbox input {
    border: none;
    outline: none;
    flex: 1;
    font-size: .9rem;
    font-family: inherit;
    background: transparent;
  }

  .chips { display: flex; gap: .5rem; overflow-x: auto; padding-bottom: .5rem; }
  .chip {
    white-space: nowrap;
    border: 1px solid #e0e0e0;
    background: #fff;
    color: #555;
    border-radius: 999px;
    padding: .4rem .8rem;
    font-size: .78rem;
    font-weight: 600;
    font-family: inherit;
    cursor: pointer;
  }
  .chip.active { background: #1b1b1b; border-color: #1b1b1b; color: #fff; }

  .group { margin-top: 1.4rem; }
  .group-head { display: flex; align-items: baseline; gap: .6rem; margin-bottom: .6rem; }
  .group-head h2 { font-size: 1.05rem; margin: 0; letter-spacing: -.01em; }
  .group-head span { color: #9a9a9a; font-size: .72rem; }

  .card {
    display: flex;
    gap: .9rem;
    background: #fff;
    border: 1px solid #ececec;
    border-radius: 16px;
    padding: 1rem;
    margin-bottom: .8rem;
    box-shadow: 0 2px 10px rgba(0, 0, 0, .03);
  }
  .card-body { flex: 1; min-width: 0; }
  .card-body h3 { margin: 0 0 .25rem; font-size: 1rem; letter-spacing: -.01em; }
  .price { font-weight: 800; font-size: .95rem; margin-bottom: .3rem; }
  .meta { color: #8a8a8a; font-size: .72rem; }
  .meta code { background: #f2f2f2; border-radius: 3px; padding: 0 .25rem; }
  .tagline {
    display: inline-block;
    margin-top: .5rem;
    font-size: .68rem;
    color: #2e8b57;
    background: #eef7f1;
    border-radius: 6px;
    padding: .15rem .4rem;
  }

  .card-art {
    width: 116px;
    height: 116px;
    flex-shrink: 0;
    border-radius: 14px;
    display: grid;
    place-items: center;
    font-size: 3rem;
    align-self: center;
  }

  .add {
    margin-top: .7rem;
    border: 1.5px solid #2e8b57;
    background: #fff;
    color: #2e8b57;
    border-radius: 10px;
    padding: .4rem 1rem;
    font-weight: 800;
    letter-spacing: .06em;
    font-size: .75rem;
    font-family: inherit;
    cursor: pointer;
  }
  .add span { margin-left: .45rem; font-size: 1rem; vertical-align: -1px; }
  .add:hover { background: #f0f9f3; }

  .stepper {
    display: inline-flex;
    align-items: center;
    margin-top: .7rem;
    border: 1.5px solid #2e8b57;
    border-radius: 10px;
    overflow: hidden;
  }
  .stepper button {
    border: none;
    background: #fff;
    color: #2e8b57;
    width: 34px;
    height: 32px;
    font-size: 1.05rem;
    font-weight: 700;
    cursor: pointer;
    font-family: inherit;
  }
  .stepper button:hover { background: #f0f9f3; }
  .stepper span { min-width: 30px; text-align: center; font-weight: 800; font-size: .85rem; }
  .stepper.small button { width: 28px; height: 26px; font-size: .9rem; }

  .state { text-align: center; color: #8a8a8a; padding: 2rem 0; font-size: .9rem; }
  .state.error, .error { color: #c0392b; }

  .cartbar {
    position: fixed;
    left: 0; right: 0; bottom: 0;
    z-index: 8;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    max-width: 760px;
    margin: 0 auto;
    padding: .8rem 1.25rem calc(.8rem + env(safe-area-inset-bottom));
    background: #fff;
    border-top: 1px solid #e9e9e9;
    box-shadow: 0 -8px 30px rgba(0, 0, 0, .07);
  }
  .cartbar-info { display: flex; flex-direction: column; font-size: .78rem; color: #777; }
  .cartbar-info strong { font-size: .95rem; color: #1b1b1b; }

  .primary {
    background: #2e8b57;
    color: #fff;
    border: none;
    border-radius: 12px;
    padding: .7rem 1.1rem;
    font-size: .9rem;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
  }
  .primary:hover:not(:disabled) { background: #26764a; }
  .primary:disabled { opacity: .5; cursor: default; }
  .wide { width: 100%; margin-top: 1rem; padding: .85rem; }

  .overlay {
    position: fixed;
    inset: 0;
    z-index: 20;
    background: rgba(0, 0, 0, .45);
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding: 1rem;
  }
  .sheet {
    width: 100%;
    max-width: 520px;
    background: #fff;
    color: #1b1b1b;
    border-radius: 18px;
    padding: 1.25rem;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    max-height: 88vh;
    overflow-y: auto;
  }
  .sheet-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: .75rem; }
  .sheet-head h2 { margin: 0; font-size: 1.05rem; }

  .line {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    padding: .65rem 0;
    border-bottom: 1px solid #f0f0f0;
    font-size: .85rem;
  }
  .line-sub { color: #9a9a9a; font-size: .7rem; margin-top: .1rem; }
  .line-right { display: flex; align-items: center; gap: .75rem; }

  .bill { margin-top: 1rem; font-size: .85rem; }
  .bill > div { display: flex; justify-content: space-between; padding: .2rem 0; color: #666; }
  .bill-total { color: #1b1b1b !important; font-weight: 800; font-size: .95rem; border-top: 1px dashed #e5e5e5; margin-top: .35rem; padding-top: .55rem !important; }
  .wallet-line { font-size: .72rem; color: #9a9a9a !important; justify-content: flex-end !important; }
  .hint { font-size: .74rem; color: #8a6d3b; background: #fff8e6; border-radius: 8px; padding: .5rem .6rem; margin: .8rem 0 0; }

  .success { text-align: center; padding: 2rem 1.5rem; }
  .check {
    width: 56px; height: 56px; border-radius: 50%;
    background: #e8f5ee; color: #2e8b57;
    display: grid; place-items: center;
    font-size: 1.8rem; margin: 0 auto .75rem;
  }
  .token { font-size: 1.05rem; margin: .25rem 0; }
  .token strong { font-size: 1.8rem; display: block; letter-spacing: -.02em; }
  .where { color: #666; font-size: .88rem; line-height: 1.4; }
  .paid { color: #9a9a9a; font-size: .75rem; }

  @media (max-width: 480px) {
    .card-art { width: 92px; height: 92px; font-size: 2.4rem; }
    .hello h1 { font-size: 1.2rem; }
  }
</style>
