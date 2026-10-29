<script>
  import { onMount, onDestroy } from 'svelte';
  import { getQueue, createOrder, ackQueue, getMenu } from '../api.js';

  let { me, log = $bindable([]) } = $props();
  const push = (msg) => { log = [msg, ...log].slice(0, 20); };

  const LANES = [
    { key: 'a',       label: 'Counter 1 — Meals',        emoji: '🍽' },
    { key: 'kitchen', label: 'Kitchen',                   emoji: '👨‍🍳' },
    { key: 'ready',   label: 'Counter 2',                 emoji: '🔔' },
  ];

  let queues = $state({
    a:       { head: null, depth: 0, items: [] },
    kitchen: { head: null, depth: 0, items: [] },
    ready:   { head: null, depth: 0, items: [] },
  });

  let menuItems = $state([]);
  let selectedCode = $state('');
  let qty = $state(1);
  let userId = $state('');
  let working = $state(false);

  let availableItems = $derived(menuItems.filter(m => m.is_available));
  let selectedItem = $derived(availableItems.find(m => m.code === selectedCode));
  // Lane is derived by the backend from the code: only MEAL goes to Counter 1.
  let route = $derived(!selectedItem ? '—' : selectedCode === 'MEAL' ? 'Counter 1' : 'Counter 2');

  async function refresh() {
    const results = await Promise.allSettled(
      LANES.map(({ key }) => getQueue(key).then(d => ({ key, d })))
    );
    for (const r of results) {
      if (r.status === 'fulfilled') queues[r.value.key] = r.value.d;
    }
  }

  async function enqueue() {
    if (!selectedItem) { push('⚠ pick a menu item'); return; }
    const buyer = me?.role === 'admin' ? Number(userId) : Number(me?.id);
    if (!buyer) { push('⚠ enter the customer user id'); return; }
    working = true;
    try {
      const res = await createOrder(buyer, [{ code: selectedCode, quantity: qty }]);
      push(`✅ order #${res.id} for user ${buyer} → ${route} (₹${(res.total_paise / 100).toFixed(2)})`);
      await refresh();
    } catch (e) {
      push(`❌ enqueue: ${e.message}`);
    } finally {
      working = false;
    }
  }

  async function ack(lane) {
    const q = queues[lane];
    if (!q.head) { push(`⚠ ${lane}: queue empty`); return; }
    working = true;
    try {
      await ackQueue(lane, q.head);
      push(`🔔 ack #${q.head} on ${lane}`);
      await refresh();
    } catch (e) {
      push(`❌ ack: ${e.message}`);
    } finally {
      working = false;
    }
  }

  let interval;
  onMount(async () => {
    try { const m = await getMenu(); menuItems = m.items ?? []; } catch {}
    await refresh();
    interval = setInterval(refresh, 2000);
  });
  onDestroy(() => clearInterval(interval));
</script>

<section class="panel">
  <h2>Live Queues</h2>
  <div class="lane-grid">
    {#each LANES as { key, label, emoji }}
      {@const q = queues[key]}
      <div class="lane-card">
        <div class="lane-header">
          <span class="lane-emoji">{emoji}</span>
          <span class="lane-label">{label}</span>
          <span class="badge {q.depth > 0 ? 'pending' : 'offline'}">Q{q.depth}</span>
        </div>
        <div class="lane-head">HEAD: <strong>{q.head ?? '—'}</strong></div>
        <div class="lane-items">
          {#if q.items?.length}
            {#each q.items as id, i}
              <span class="token" class:head={i === 0}>#{id}</span>
            {/each}
          {:else}
            <span class="empty">empty</span>
          {/if}
        </div>
        <button disabled={working || !q.head} onclick={() => ack(key)}>Ack ✓</button>
      </div>
    {/each}
  </div>

  <div class="enqueue-box">
    <h2>Enqueue</h2>
    <div class="enqueue-row">
      <div class="field">
        <label for="enqueue-code">Menu item</label>
        <select id="enqueue-code" bind:value={selectedCode}>
          <option value="">— pick item —</option>
          {#each availableItems as m}
            <option value={m.code}>{m.item_name} ({m.code})</option>
          {/each}
        </select>
      </div>
      <div class="field">
        <label for="enqueue-user">Customer user id</label>
        {#if me?.role === 'admin'}
          <input id="enqueue-user" type="number" min="1" bind:value={userId} placeholder="required" />
        {:else}
          <input id="enqueue-user" value={me?.id ?? ''} disabled />
        {/if}
      </div>
      <div class="field qty">
        <label for="enqueue-qty">Qty</label>
        <input id="enqueue-qty" type="number" min="1" bind:value={qty} />
      </div>
      <div class="field route">
        <span class="route-label">Routes to</span>
        <strong>{route}</strong>
      </div>
      <button class="primary enqueue-btn" disabled={working || !selectedItem} onclick={enqueue}>+ Enqueue</button>
    </div>
  </div>
</section>

<style>
  .panel { padding: 1rem 1.5rem; }

  .lane-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .lane-card {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 6px;
    padding: .75rem 1rem;
    display: flex;
    flex-direction: column;
    gap: .4rem;
  }

  .lane-header { display: flex; align-items: center; gap: .4rem; font-size: .85rem; font-weight: 600; }
  .lane-emoji  { font-size: 1rem; }
  .lane-label  { flex: 1; }
  .lane-head   { font-size: .75rem; color: #8b949e; }

  .lane-items { display: flex; flex-wrap: wrap; gap: .25rem; min-height: 1.5rem; }
  .token {
    font-size: .7rem;
    background: #21262d;
    border: 1px solid #30363d;
    border-radius: 4px;
    padding: .1rem .3rem;
    color: #c9d1d9;
  }
  .token.head { border-color: #58a6ff; color: #58a6ff; font-weight: 700; }
  .empty { font-size: .75rem; color: #484f58; }

  .enqueue-box {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 6px;
    padding: .75rem 1rem;
  }
  .enqueue-row { display: flex; gap: .75rem; align-items: flex-end; flex-wrap: wrap; }
  .field { flex: 1; min-width: 130px; }
  .field.qty { max-width: 70px; }
  .field.route { flex: 0 0 auto; min-width: 150px; }
  .route-label { display: block; font-size: .7rem; color: #8b949e; }
  .field.route strong { font-size: .8rem; color: #58a6ff; }
  .enqueue-btn { white-space: nowrap; height: 2rem; }
</style>
