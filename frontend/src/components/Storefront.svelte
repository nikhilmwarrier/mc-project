<script>
  import { onMount, onDestroy } from 'svelte';
  import { getMenu, getWallet, createOrder, getOrder } from '../api.js';

  let { me, onlogout } = $props();

  const EMOJI = {
    MEAL: '🍛', POR: '🫓', CHK: '🍗', SNACK: '🍟',
    PUR: '🥘', DIH: '🍚', IDP: '🍜', CHP: '🍲',
  };
  const IMAGES = {
    MEAL: '/food/meal.jpg',
    POR: '/food/porotta.jpg',
    CHP: '/food/chapati.jpg',
    CHK: '/food/chicken-curry.jpg',
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

  let tracked = $state(null);
  let pollTimer = null;
  let installEvent = $state(null);

  function captureInstall(e) {
    e.preventDefault();
    installEvent = e;
  }

  async function install() {
    if (!installEvent) return;
    installEvent.prompt();
    try { await installEvent.userChoice; } finally { installEvent = null; }
  }

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

  function stopPolling() {
    if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
  }

  function forgetTracking() {
    stopPolling();
    tracked = null;
    try { sessionStorage.removeItem('canteen_track'); } catch { /* private mode */ }
  }

  function rememberTracking(id) {
    try { sessionStorage.setItem('canteen_track', String(id)); } catch { /* private mode */ }
  }

  function notifyReady(id) {
    if (!('Notification' in window) || Notification.permission !== 'granted') return;
    try {
      new Notification(`Token #${id} ready`, {
        body: 'Collect at Counter 2',
        tag: `canteen-order-${id}`,
      });
    } catch { /* some browsers require a service worker */ }
  }

  async function checkTracked() {
    if (!tracked || tracked.status !== 'pending') return;
    try {
      const o = await getOrder(tracked.id);
      const becameReady = o.ready && !tracked.ready;
      tracked = { id: o.id, ready: !!o.ready, status: o.status };
      if (becameReady) notifyReady(o.id);
      if (o.status !== 'pending') stopPolling();
    } catch { /* transient; poll again */ }
  }

  function startTracking(id) {
    stopPolling();
    tracked = { id, ready: false, status: 'pending' };
    rememberTracking(id);
    pollTimer = setInterval(checkTracked, 4000);
    checkTracked();
  }

  function askNotifyPermission() {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission().catch(() => {});
    }
  }

  async function placeOrder() {
    if (!lines.length) return;
    placing = true;
    placeError = '';
    try {
      const res = await createOrder(me.id, lines.map(l => ({ code: l.code, quantity: l.qty })));
      placed = { id: res.id, special: cartHasSpecial };
      cart = {};
      cartOpen = false;
      await refreshWallet();
      startTracking(res.id);
    } catch (e) {
      placeError = e.message;
    } finally {
      placing = false;
    }
  }

  onMount(async () => {
    window.addEventListener('beforeinstallprompt', captureInstall);
    try {
      const menu = await getMenu();
      items = menu.items ?? [];
    } catch (e) {
      loadError = e.message;
    } finally {
      loading = false;
    }
    await refreshWallet();

    let saved = null;
    try { saved = Number(sessionStorage.getItem('canteen_track')); } catch { /* private mode */ }
    if (saved) {
      try {
        const o = await getOrder(saved);
        if (o.status === 'pending') {
          tracked = { id: o.id, ready: !!o.ready, status: o.status };
          pollTimer = setInterval(checkTracked, 4000);
        }
      } catch { /* order gone */ }
    }
  });

  onDestroy(() => {
    stopPolling();
    window.removeEventListener('beforeinstallprompt', captureInstall);
  });
</script>

<div class="storefront">
  <header class="topbar">
    <div class="brand">CAMPUS <span>KITCHEN</span></div>
    <div class="top-actions">
      {#if wallet !== null}
        <span class="wallet">₹{(wallet / 100).toFixed(2)}</span>
      {/if}
      {#if installEvent}
        <button class="ghost" onclick={install}>Install</button>
      {/if}
      <button class="ghost" onclick={onlogout}>Logout</button>
    </div>
  </header>

  {#if tracked}
    <div class="track" class:ready={tracked.ready} class:done={tracked.status !== 'pending'}>
      <span class="track-text">
        {#if tracked.status === 'completed'}
          #{tracked.id} collected
        {:else if tracked.ready}
          #{tracked.id} ready · Counter 2
        {:else}
          #{tracked.id} preparing
        {/if}
      </span>
      <button class="ghost" onclick={forgetTracking}>✕</button>
    </div>
  {/if}

  <main class="content">
    <div class="searchbox">
      <span aria-hidden="true">🔍</span>
      <input bind:value={search} placeholder="Search menu" aria-label="Search the menu" />
    </div>

    <div class="chips" role="group" aria-label="Filter by counter">
      <button class="chip" class:active={section === 'all'} onclick={() => section = 'all'}>All</button>
      <button class="chip" class:active={section === 'a'} onclick={() => section = 'a'}>Counter 1</button>
      <button class="chip" class:active={section === 'b'} onclick={() => section = 'b'}>Counter 2</button>
    </div>

    {#if loading}
      <p class="state">Loading…</p>
    {:else if loadError}
      <p class="state error">{loadError}</p>
    {:else if !visible.length}
      <p class="state">No items.</p>
    {:else}
      {#if groupA.length && section !== 'b'}
        <section class="group">
          <h2>Counter 1</h2>
          {#each groupA as item (item.id)}
            <article class="card">
              <div class="card-body">
                <h3>{item.item_name}</h3>
                <div class="price">{money(item.price_paise)}</div>
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
              <div class="card-art">
                {#if IMAGES[item.code]}
                  <img src={IMAGES[item.code]} alt={item.item_name} loading="lazy" />
                {:else}
                  <span class="card-emoji" style="background:{artwork(item.code)}">{emoji(item.code)}</span>
                {/if}
              </div>
            </article>
          {/each}
        </section>
      {/if}

      {#if groupB.length && section !== 'a'}
        <section class="group">
          <h2>Counter 2</h2>
          {#each groupB as item (item.id)}
            <article class="card">
              <div class="card-body">
                <h3>{item.item_name}</h3>
                <div class="price">{money(item.price_paise)}</div>
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
              <div class="card-art">
                {#if IMAGES[item.code]}
                  <img src={IMAGES[item.code]} alt={item.item_name} loading="lazy" />
                {:else}
                  <span class="card-emoji" style="background:{artwork(item.code)}">{emoji(item.code)}</span>
                {/if}
              </div>
            </article>
          {/each}
        </section>
      {/if}
    {/if}
  </main>

  {#if cartCount > 0}
    <div class="cartbar">
      <strong>{cartCount} item{cartCount > 1 ? 's' : ''} · {money(cartTotal)}</strong>
      <button class="primary" onclick={() => { cartOpen = true; askNotifyPermission(); }}>View Cart →</button>
    </div>
  {/if}

  <footer class="credits"><a href="/food/CREDITS.md">Photo credits</a></footer>
</div>

{#if cartOpen}
  <div class="overlay" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) cartOpen = false; }}>
    <div class="sheet" role="dialog" aria-modal="true" aria-label="Cart">
      <div class="sheet-head">
        <h2>Cart</h2>
        <button class="ghost" onclick={() => cartOpen = false}>Close</button>
      </div>

      {#each lines as { item, code, qty }}
        <div class="line">
          <strong>{item.item_name}</strong>
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

      <div class="bill-total"><span>Total</span><strong>{money(cartTotal)}</strong></div>

      {#if placeError}
        <p class="error">{placeError}</p>
      {/if}

      <button class="primary wide" disabled={placing || cartTotal > (wallet ?? 0)} onclick={placeOrder}>
        {placing ? 'Placing…' : `Place order · ${money(cartTotal)}`}
      </button>
      {#if wallet !== null && cartTotal > wallet}
        <p class="error">Balance too low.</p>
      {/if}
    </div>
  </div>
{/if}

{#if placed}
  <div class="overlay">
    <div class="sheet success" role="dialog" aria-modal="true" aria-label="Order placed">
      <div class="check">✓</div>
      <h2>Order placed</h2>
      <div class="token">#{placed.id}</div>
      <p class="where">{placed.special ? 'Counter 2' : 'Counter 1'}</p>
      <button class="primary wide" onclick={() => placed = null}>Done</button>
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

  .track {
    max-width: 760px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: .5rem;
    padding: .5rem 1.25rem;
    background: #fff8e6;
    color: #8a6d3b;
    border-bottom: 1px solid #f1e6c8;
    font-size: .8rem;
    font-weight: 600;
  }
  .track.ready { background: #e8f5ee; color: #1f6f45; border-color: #cfe9da; }
  .track.done { background: #f2f2f2; color: #777; border-color: #e6e6e6; }

  .content { max-width: 760px; margin: 0 auto; padding: 0 1.25rem; }

  .searchbox {
    display: flex;
    align-items: center;
    gap: .5rem;
    background: #fff;
    border: 1px solid #e6e6e6;
    border-radius: 12px;
    padding: .65rem .8rem;
    margin: 1rem 0 .75rem;
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
  .group h2 { font-size: 1.05rem; margin: 0 0 .6rem; letter-spacing: -.01em; }

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
  .price { font-weight: 800; font-size: .95rem; }

  .card-art {
    width: 108px;
    height: 108px;
    flex-shrink: 0;
    border-radius: 14px;
    overflow: hidden;
    align-self: center;
    background: #f4f4f4;
  }
  .card-art img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .card-emoji {
    width: 100%;
    height: 100%;
    display: grid;
    place-items: center;
    font-size: 2.8rem;
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
    font-size: .95rem;
  }

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

  .credits { text-align: center; padding: 2rem 0 1rem; font-size: .68rem; }
  .credits a { color: #aaa; text-decoration: none; }
  .credits a:hover { color: #777; }

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
  .line-right { display: flex; align-items: center; gap: .75rem; }

  .bill-total {
    display: flex;
    justify-content: space-between;
    margin-top: .75rem;
    font-weight: 800;
    font-size: .95rem;
  }

  .success { text-align: center; padding: 2rem 1.5rem; }
  .check {
    width: 56px; height: 56px; border-radius: 50%;
    background: #e8f5ee; color: #2e8b57;
    display: grid; place-items: center;
    font-size: 1.8rem; margin: 0 auto .75rem;
  }
  .token { font-size: 1.9rem; font-weight: 800; letter-spacing: -.02em; }
  .where { color: #2e8b57; font-size: 1rem; font-weight: 700; margin: .25rem 0 0; }

  @media (max-width: 480px) {
    .card-art { width: 88px; height: 88px; font-size: 2.3rem; }
  }
</style>
