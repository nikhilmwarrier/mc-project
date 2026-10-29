<script>
  import { onMount, onDestroy } from 'svelte';
  import {
    Search, X, Wallet, LogOut, Download, ShoppingBag, Plus, Minus,
    UtensilsCrossed, Check, ChevronRight, Clock3, Ticket,
  } from 'lucide-svelte';
  import { getMenu, getWallet, createOrder, getOrder } from '../api.js';
  import { haptic } from '../lib/haptics.js';

  let { me, onlogout } = $props();

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

  let tracked = $state([]);
  let tokenOpen = $state(false);
  let pollTimer = null;
  let installEvent = $state(null);

  const money = (paise) => paise % 100 === 0
    ? `₹${paise / 100}`
    : `₹${(paise / 100).toFixed(2)}`;
  const amount = (paise) => paise % 100 === 0
    ? String(paise / 100)
    : (paise / 100).toFixed(2);

  function artwork(code) {
    let h = 0;
    for (const c of code) h = (h * 31 + c.charCodeAt(0)) >>> 0;
    return ART[h % ART.length];
  }

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
  let readyOrders = $derived(tracked.filter(o => o.status === 'pending' && o.ready));
  let preparingOrders = $derived(tracked.filter(o => o.status === 'pending' && !o.ready));

  function add(code) {
    haptic(6);
    cart = { ...cart, [code]: (cart[code] ?? 0) + 1 };
  }
  function dec(code) {
    haptic(4);
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

  function ensurePolling() {
    if (!pollTimer && tracked.length) pollTimer = setInterval(checkTracked, 4000);
  }

  function persistTracked() {
    try { sessionStorage.setItem('canteen_track', JSON.stringify(tracked.map(o => o.id))); }
    catch { /* private mode */ }
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
    if (!tracked.length) { stopPolling(); return; }

    const before = new Map(tracked.map(o => [o.id, o.ready]));
    const results = await Promise.all(tracked.map(async (o) => {
      if (o.status !== 'pending') return o;
      try {
        const d = await getOrder(o.id);
        return { id: o.id, ready: !!d.ready, status: d.status };
      } catch {
        return o;
      }
    }));

    const becameReady = results.filter(o =>
      o.status === 'pending' && o.ready && !before.get(o.id)
    );
    tracked = results.filter(o => o.status === 'pending');
    persistTracked();

    if (becameReady.length) {
      becameReady.forEach(o => notifyReady(o.id));
      cartOpen = false;
      tokenOpen = true;
    }
    if (!tracked.some(o => o.ready)) tokenOpen = false;
    if (!tracked.length) stopPolling();
  }

  function startTracking(id) {
    if (!tracked.some(o => o.id === id)) {
      tracked = [...tracked, { id, ready: false, status: 'pending' }];
    }
    persistTracked();
    ensurePolling();
    checkTracked();
  }

  function askNotifyPermission() {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission().catch(() => {});
    }
  }

  async function placeOrder() {
    if (!lines.length) return;
    haptic(12);
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

  function captureInstall(e) {
    e.preventDefault();
    installEvent = e;
  }

  async function install() {
    if (!installEvent) return;
    haptic(8);
    installEvent.prompt();
    try { await installEvent.userChoice; } finally { installEvent = null; }
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

    let saved = [];
    try {
      const raw = sessionStorage.getItem('canteen_track');
      const parsed = raw ? JSON.parse(raw) : [];
      saved = Array.isArray(parsed) ? parsed : [parsed];
    } catch { /* private mode */ }

    if (saved.length) {
      const results = await Promise.all(saved.map(async (id) => {
        try {
          const o = await getOrder(id);
          return { id, ready: !!o.ready, status: o.status };
        } catch {
          return { id, ready: false, status: 'completed' };
        }
      }));
      tracked = results.filter(o => o.status === 'pending');
      persistTracked();
      if (tracked.some(o => o.ready)) tokenOpen = true;
      if (tracked.length) pollTimer = setInterval(checkTracked, 4000);
    }
  });

  onDestroy(() => {
    stopPolling();
    window.removeEventListener('beforeinstallprompt', captureInstall);
  });
</script>

<div class="storefront">
  <header class="topbar">
    <div class="brand">College <span>Canteen</span></div>
    <div class="top-actions">
      {#if wallet !== null}
        <span class="wallet"><Wallet size={14} strokeWidth={2} aria-hidden="true" />₹{(wallet / 100).toFixed(2)}</span>
      {/if}
      {#if installEvent}
        <button class="icon-btn" onclick={install} aria-label="Install app" title="Install app">
          <Download size={18} strokeWidth={2} />
        </button>
      {/if}
      <button class="icon-btn" onclick={onlogout} aria-label="Log out" title="Log out">
        <LogOut size={18} strokeWidth={2} />
      </button>
    </div>
  </header>

  {#if preparingOrders.length}
    <div class="track">
      <span class="track-text">
        <Clock3 size={15} strokeWidth={2} aria-hidden="true" />
        {preparingOrders.length === 1
          ? `#${preparingOrders[0].id} preparing`
          : `${preparingOrders.length} orders preparing`}
      </span>
    </div>
  {/if}

  <main class="content">
    <div class="searchbox">
      <Search size={17} strokeWidth={2} aria-hidden="true" />
      <input bind:value={search} placeholder="Search menu" aria-label="Search the menu" />
    </div>

    <div class="chips" role="group" aria-label="Filter by counter">
      <button class="chip" class:active={section === 'all'} onclick={() => { haptic(4); section = 'all'; }}>All</button>
      <button class="chip" class:active={section === 'a'} onclick={() => { haptic(4); section = 'a'; }}>Counter 1</button>
      <button class="chip" class:active={section === 'b'} onclick={() => { haptic(4); section = 'b'; }}>Counter 2</button>
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
            <article class="item-row">
              <div class="item-body">
                <h3>{item.item_name}</h3>
                <div class="price"><span class="currency">₹</span>{amount(item.price_paise)}</div>
              </div>
              <div class="item-media">
                {#if IMAGES[item.code]}
                  <img src={IMAGES[item.code]} alt={item.item_name} loading="lazy" />
                {:else}
                  <span class="card-fallback" style="background:{artwork(item.code)}">
                    <UtensilsCrossed size={36} strokeWidth={1.5} aria-hidden="true" />
                  </span>
                {/if}
                {#if cart[item.code]}
                  <div class="stepper">
                    <button onclick={() => dec(item.code)} aria-label={`Remove one ${item.item_name}`}>
                      <Minus size={15} strokeWidth={2.25} />
                    </button>
                    <span>{cart[item.code]}</span>
                    <button onclick={() => add(item.code)} aria-label={`Add one more ${item.item_name}`}>
                      <Plus size={15} strokeWidth={2.25} />
                    </button>
                  </div>
                {:else}
                  <button class="add" onclick={() => add(item.code)}>ADD</button>
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
            <article class="item-row">
              <div class="item-body">
                <h3>{item.item_name}</h3>
                <div class="price"><span class="currency">₹</span>{amount(item.price_paise)}</div>
              </div>
              <div class="item-media">
                {#if IMAGES[item.code]}
                  <img src={IMAGES[item.code]} alt={item.item_name} loading="lazy" />
                {:else}
                  <span class="card-fallback" style="background:{artwork(item.code)}">
                    <UtensilsCrossed size={36} strokeWidth={1.5} aria-hidden="true" />
                  </span>
                {/if}
                {#if cart[item.code]}
                  <div class="stepper">
                    <button onclick={() => dec(item.code)} aria-label={`Remove one ${item.item_name}`}>
                      <Minus size={15} strokeWidth={2.25} />
                    </button>
                    <span>{cart[item.code]}</span>
                    <button onclick={() => add(item.code)} aria-label={`Add one more ${item.item_name}`}>
                      <Plus size={15} strokeWidth={2.25} />
                    </button>
                  </div>
                {:else}
                  <button class="add" onclick={() => add(item.code)}>ADD</button>
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
      <div class="cartbar-info">
        <strong>{cartCount} item{cartCount > 1 ? 's' : ''} · {money(cartTotal)}</strong>
      </div>
      <button class="primary" onclick={() => { haptic(6); cartOpen = true; askNotifyPermission(); }}>
        <ShoppingBag size={16} strokeWidth={2} aria-hidden="true" />
        View Cart
        <ChevronRight size={15} strokeWidth={2.25} aria-hidden="true" />
      </button>
    </div>
  {/if}

  <footer class="credits"><a href="/food/CREDITS.md">Photo credits</a></footer>
</div>

{#if readyOrders.length && !tokenOpen}
  <button class="fab" class:raised={cartCount > 0} onclick={() => { haptic(8); tokenOpen = true; }}>
    <Ticket size={17} strokeWidth={2} aria-hidden="true" />
    Show token ({readyOrders.length})
  </button>
{/if}

{#if cartOpen}
  <div class="overlay" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) cartOpen = false; }}>
    <div class="sheet" role="dialog" aria-modal="true" aria-label="Cart">
      <div class="sheet-head">
        <h2>Cart</h2>
        <button class="icon-btn" onclick={() => cartOpen = false} aria-label="Close cart">
          <X size={18} strokeWidth={2} />
        </button>
      </div>

      {#each lines as { item, code, qty }}
        <div class="line">
          <strong>{item.item_name}</strong>
          <div class="line-right">
            <div class="stepper small">
              <button onclick={() => dec(code)} aria-label={`Remove one ${item.item_name}`}>
                <Minus size={14} strokeWidth={2.25} />
              </button>
              <span>{qty}</span>
              <button onclick={() => add(code)} aria-label={`Add one more ${item.item_name}`}>
                <Plus size={14} strokeWidth={2.25} />
              </button>
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

{#if tokenOpen && readyOrders.length}
  <div class="overlay" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) tokenOpen = false; }}>
    <div class="sheet token-sheet" role="dialog" aria-modal="true" aria-label="Ready token">
      <div class="sheet-head">
        <h2>Ready</h2>
        <button class="icon-btn" onclick={() => tokenOpen = false} aria-label="Dismiss token">
          <X size={18} strokeWidth={2} />
        </button>
      </div>
      {#each readyOrders as o}
        <div class="ready-row">
          <div class="token-big">#{o.id}</div>
          <div class="ready-hint">Counter 2</div>
        </div>
      {/each}
      <button class="primary wide" onclick={() => tokenOpen = false}>Done</button>
    </div>
  </div>
{/if}

{#if placed}
  <div class="overlay">
    <div class="sheet success" role="dialog" aria-modal="true" aria-label="Order placed">
      <div class="check"><Check size={26} strokeWidth={2.5} /></div>
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
    background: #fff;
    color: #1b1b1b;
    font-family: var(--font-sans);
    padding-bottom: 6rem;
  }

  .topbar {
    position: sticky;
    top: 0;
    z-index: 5;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    min-height: 64px;
    padding: 0 20px;
    background: rgba(255, 255, 255, .92);
    backdrop-filter: blur(12px);
    border-bottom: 1px solid #ececec;
    max-width: 760px;
    margin: 0 auto;
  }
  .brand { font-family: var(--font-logo); font-size: 1.55rem; font-weight: 400; letter-spacing: .01em; }
  .brand span { color: #2e8b57; }

  .top-actions { display: flex; align-items: center; gap: 6px; }
  .wallet {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: .78rem;
    font-weight: 700;
    color: #1f6f45;
    background: #e8f5ee;
    border-radius: 999px;
    padding: 6px 10px;
    font-variant-numeric: tabular-nums;
  }

  .icon-btn {
    display: inline-grid;
    place-items: center;
    width: 34px;
    height: 34px;
    padding: 0;
    border: none;
    border-radius: 10px;
    background: transparent;
    color: #777;
  }
  .icon-btn:hover:not(:disabled) { background: #f1f1f1; color: #1b1b1b; }

  .track {
    max-width: 760px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 20px;
    background: #fff8e6;
    color: #8a6d3b;
    border-bottom: 1px solid #f1e6c8;
    font-size: .78rem;
    font-weight: 600;
    transition: background-color 240ms ease, color 240ms ease, border-color 240ms ease;
    animation: slide-down 240ms var(--ease-out) both;
  }
  .track-text { display: inline-flex; align-items: center; gap: 7px; font-variant-numeric: tabular-nums; }

  @keyframes slide-down {
    from { opacity: 0; transform: translateY(-6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .content { max-width: 760px; margin: 0 auto; padding: 0 20px; }

  .searchbox {
    display: flex;
    align-items: center;
    gap: 9px;
    height: 44px;
    background: #fff;
    border: 1px solid #e6e6e6;
    border-radius: 12px;
    padding: 0 12px;
    margin: 16px 0 12px;
    color: #999;
    transition: border-color 160ms ease, box-shadow 160ms ease;
  }
  .searchbox:focus-within { border-color: #2e8b57; box-shadow: 0 0 0 3px rgba(46, 139, 87, .1); }
  .searchbox input {
    border: none;
    outline: none;
    flex: 1;
    height: 100%;
    font-size: .88rem;
    font-family: inherit;
    background: transparent;
    color: #1b1b1b;
  }
  /* The container owns the focus ring; the inner input must not add another. */
  .searchbox input:focus,
  .searchbox input:focus-visible {
    outline: none;
    box-shadow: none;
    border-color: transparent;
  }

  .chips { display: flex; gap: 8px; overflow-x: auto; padding-bottom: 4px; scrollbar-width: none; }
  .chips::-webkit-scrollbar { display: none; }
  .chip {
    white-space: nowrap;
    border: 1px solid #e4e4e4;
    background: #fff;
    color: #666;
    border-radius: 999px;
    padding: 7px 14px;
    font-size: .77rem;
    font-weight: 600;
    font-family: inherit;
    cursor: pointer;
    transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease, transform 160ms var(--ease-out);
  }
  .chip.active { background: #1b1b1b; border-color: #1b1b1b; color: #fff; }

  .group { margin-top: 28px; }
  .group h2 {
    font-size: .72rem;
    font-weight: 600;
    letter-spacing: .09em;
    text-transform: uppercase;
    color: #747474;
    margin: 0 0 2px;
  }

  .item-row {
    display: flex;
    align-items: flex-start;
    gap: 18px;
    padding: 14px 0 22px;
    border-bottom: 1px solid #f0f0f0;
  }
  .item-row:last-child { border-bottom: none; }
  @media (hover: hover) and (pointer: fine) {
    .item-row:hover { background: #fcfcfc; }
  }

  .item-body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: flex-start;
    padding-top: 8px;
    gap: 5px;
  }
  .item-body h3 {
    margin: 0;
    font-size: 1.07rem;
    font-weight: 700;
    letter-spacing: -.02em;
    line-height: 1.25;
    color: #111;
  }
  .price {
    font-size: .92rem;
    font-weight: 600;
    color: #5f5f5f;
    letter-spacing: .005em;
    font-variant-numeric: tabular-nums;
  }
  .currency { font-size: .78em; font-weight: 500; color: #8a8a8a; margin-right: 1px; }

  .item-media {
    position: relative;
    width: 132px;
    height: 132px;
    flex-shrink: 0;
    border-radius: 14px;
    background: #f4f4f4;
  }
  .item-media img,
  .item-media .card-fallback {
    width: 100%;
    height: 100%;
    border-radius: 14px;
    object-fit: cover;
    display: block;
  }
  .item-media .card-fallback { display: grid; place-items: center; color: #6f6f6f; }

  .add {
    position: absolute;
    left: 50%;
    bottom: -20px;
    transform: translateX(-50%);
    width: calc(100% - 16px);
    height: 40px;
    border: 1px solid #e2e2e2;
    background: #fff;
    color: #2e8b57;
    border-radius: 8px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-weight: 800;
    letter-spacing: .07em;
    font-size: .8rem;
    font-family: inherit;
    cursor: pointer;
    box-shadow: 0 3px 10px rgba(0, 0, 0, .14);
  }
  .add:active:not(:disabled) { transform: translateX(-50%) scale(0.97); }
  @media (hover: hover) and (pointer: fine) {
    .add:hover { background: #fff; box-shadow: 0 5px 14px rgba(0, 0, 0, .18); }
  }

  .item-media .stepper {
    position: absolute;
    left: 50%;
    bottom: -20px;
    transform: translateX(-50%);
    width: calc(100% - 16px);
    height: 40px;
    margin: 0;
    justify-content: space-between;
    border: 1px solid #e2e2e2;
    background: #fff;
    border-radius: 8px;
    box-shadow: 0 3px 10px rgba(0, 0, 0, .14);
  }
  .item-media .stepper:active { transform: translateX(-50%); }
  .stepper {
    display: inline-flex;
    align-items: center;
    height: 34px;
    border: 1.5px solid #2e8b57;
    border-radius: 10px;
    overflow: hidden;
  }
  .stepper button {
    display: inline-grid;
    place-items: center;
    border: none;
    border-radius: 0;
    background: #fff;
    color: #2e8b57;
    width: 30px;
    height: 100%;
    padding: 0;
    flex-shrink: 0;
  }
  .stepper button:hover:not(:disabled) { background: #f0f9f3; }
  .stepper span {
    flex: 1;
    text-align: center;
    font-weight: 800;
    font-size: .8rem;
    font-variant-numeric: tabular-nums;
  }
  .stepper.small { height: 30px; margin-top: 0; }
  .stepper.small button { width: 30px; }

  .state { text-align: center; color: #8a8a8a; padding: 32px 0; font-size: .88rem; }
  .state.error, .error { color: #c0392b; }

  .cartbar {
    position: fixed;
    left: 0; right: 0; bottom: 0;
    z-index: 8;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    max-width: 760px;
    margin: 0 auto;
    padding: 12px 20px calc(12px + env(safe-area-inset-bottom));
    background: #fff;
    border-top: 1px solid #e9e9e9;
    box-shadow: 0 -8px 30px rgba(0, 0, 0, .06);
    transition: transform 260ms var(--ease-drawer), opacity 200ms ease;
  }
  .cartbar-info strong { font-size: .93rem; font-variant-numeric: tabular-nums; }

  @starting-style {
    .cartbar { transform: translateY(100%); opacity: 0; }
  }

  .primary {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    background: #2e8b57;
    color: #fff;
    border: none;
    border-radius: 12px;
    padding: 0 16px;
    height: 44px;
    font-size: .88rem;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
    font-variant-numeric: tabular-nums;
  }
  .primary:hover:not(:disabled) { background: #26764a; }
  .primary:disabled { opacity: .5; cursor: default; }
  .wide { width: 100%; margin-top: 16px; }

  .fab {
    position: fixed;
    right: 18px;
    bottom: 20px;
    z-index: 9;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 46px;
    padding: 0 18px;
    border: none;
    border-radius: 999px;
    background: #1f6f45;
    color: #fff;
    font: 700 .85rem/1 var(--font-sans);
    box-shadow: 0 10px 26px rgba(31, 111, 69, .3);
    transition: transform 160ms var(--ease-out), bottom 200ms var(--ease-out);
  }
  .fab:hover:not(:disabled) { background: #1a5f3b; }
  .fab:active:not(:disabled) { transform: scale(.97); }
  .fab.raised { bottom: calc(86px + env(safe-area-inset-bottom)); }
  @starting-style {
    .fab { opacity: 0; transform: translateY(8px); }
  }

  .token-sheet { text-align: center; }
  .token-sheet .sheet-head { text-align: left; }
  .ready-row { padding: 10px 0 16px; }
  .ready-row + .ready-row { border-top: 1px solid #f1f1f1; padding-top: 20px; }
  .token-big {
    font-size: 3.2rem;
    font-weight: 800;
    letter-spacing: -.03em;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
    color: #111;
  }
  .ready-hint { color: #2e8b57; font-weight: 700; font-size: .9rem; margin-top: 4px; }

  .overlay {
    position: fixed;
    inset: 0;
    z-index: 20;
    background: rgba(0, 0, 0, .42);
    backdrop-filter: blur(2px);
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding: 16px;
    transition: opacity 200ms ease;
  }
  @starting-style {
    .overlay { opacity: 0; }
  }
  .sheet {
    width: 100%;
    max-width: 520px;
    background: #fff;
    color: #1b1b1b;
    border-radius: 20px;
    padding: 20px;
    font-family: var(--font-sans);
    max-height: 88vh;
    overflow-y: auto;
    transition: transform 320ms var(--ease-drawer), opacity 200ms ease;
  }
  @starting-style {
    .sheet { transform: translateY(28px); opacity: 0; }
  }
  .sheet-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
  .sheet-head h2 { margin: 0; font-size: 1.02rem; }

  .line {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 14px;
    padding: 11px 0;
    border-bottom: 1px solid #f1f1f1;
    font-size: .86rem;
  }
  .line-right { display: flex; align-items: center; gap: 12px; font-variant-numeric: tabular-nums; }

  .bill-total {
    display: flex;
    justify-content: space-between;
    margin-top: 12px;
    font-weight: 800;
    font-size: .95rem;
    font-variant-numeric: tabular-nums;
  }

  .success { text-align: center; padding: 28px 20px; }
  .check {
    width: 56px; height: 56px; border-radius: 50%;
    background: #e8f5ee; color: #2e8b57;
    display: grid; place-items: center;
    margin: 0 auto 10px;
    animation: pop 260ms var(--ease-out) both;
  }
  @keyframes pop {
    from { opacity: 0; transform: scale(0.88); }
    to { opacity: 1; transform: scale(1); }
  }
  .token { font-size: 1.9rem; font-weight: 800; letter-spacing: -.02em; font-variant-numeric: tabular-nums; }
  .where { color: #2e8b57; font-size: .98rem; font-weight: 700; margin: 4px 0 0; }

  .credits { text-align: center; padding: 28px 0 12px; font-size: .67rem; }
  .credits a { color: #aaa; text-decoration: none; }
  .credits a:hover { color: #777; }

  @media (max-width: 480px) {
    .item-row { gap: 16px; padding: 12px 0 24px; }
    .item-body h3 { font-size: 1.02rem; }
    .item-media { width: 120px; height: 120px; }
    .add, .item-media .stepper { bottom: -19px; height: 38px; font-size: .78rem; }
    .content, .topbar, .track { padding-left: 16px; padding-right: 16px; }
  }
</style>
