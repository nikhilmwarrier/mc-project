<script>
  import { onMount, onDestroy } from 'svelte';
  import { getQueue, createOrder, ackQueue, getMenu, getUsers } from '../api.js';

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
  let users = $state([]);
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
    try { users = (await getUsers()).users ?? []; } catch {}
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
        <label for="enqueue-user">Customer</label>
        <select id="enqueue-user" bind:value={userId}>
          <option value="">— select user —</option>
          {#each users as u}
            <option value={u.id}>{u.name} ({u.roll_no})</option>
          {/each}
        </select>
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
  .panel { padding: 1.25rem 1.5rem; }
  .panel > h2 { font-size: 1.05rem; margin-bottom: .75rem; }

  .lane-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
    margin-bottom: 1.25rem;
  }
  @media (max-width: 720px) { .lane-grid { grid-template-columns: 1fr; } }

  .lane-card {
    background: #fff;
    border: 1px solid #ececec;
    border-radius: 16px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: .45rem;
  }

  .lane-header { display: flex; align-items: center; gap: .4rem; font-size: .88rem; font-weight: 700; }
  .lane-emoji  { font-size: 1rem; }
  .lane-label  { flex: 1; }
  .lane-head   { font-size: .78rem; color: #777; }

  .lane-items { display: flex; flex-wrap: wrap; gap: .3rem; min-height: 1.5rem; }
  .token {
    font-size: .7rem;
    background: #f4f4f4;
    border: 1px solid #e8e8e8;
    border-radius: 6px;
    padding: .1rem .35rem;
    color: #444;
  }
  .token.head { border-color: #2e8b57; color: #1f6f45; background: #e8f5ee; font-weight: 700; }
  .empty { font-size: .75rem; color: #aaa; }

  .lane-card button { margin-top: .35rem; }

  .enqueue-box {
    background: #fff;
    border: 1px solid #ececec;
    border-radius: 16px;
    padding: 1rem;
  }
  .enqueue-row { display: flex; gap: .75rem; align-items: flex-end; flex-wrap: wrap; }
  .field { flex: 1; min-width: 140px; }
  .field.qty { max-width: 80px; }
  .field.route { flex: 0 0 auto; min-width: 130px; }
  .route-label { display: block; font-size: .72rem; color: #777; margin-bottom: .35rem; }
  .field.route strong { font-size: .85rem; color: #1f6f45; }
  .enqueue-btn { white-space: nowrap; height: 2.1rem; }
</style>
