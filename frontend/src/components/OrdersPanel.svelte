<script>
  import { onMount } from 'svelte';
  import { X } from 'lucide-svelte';
  import { listOrders, getOrder, cancelOrder, getUsers } from '../api.js';
  import { haptic } from '../lib/haptics.js';

  let { log = $bindable([]) } = $props();
  const push = (msg) => { log = [msg, ...log].slice(0, 20); };

  let status  = $state('pending');
  let userId  = $state('');
  let users   = $state([]);
  let orders  = $state([]);
  let detail  = $state(null);
  let loading = $state(false);

  const nameOf = (id) => {
    const u = users.find(x => x.id === id);
    return u ? u.name : `#${id}`;
  };

  async function loadUsers() {
    try { users = (await getUsers()).users ?? []; } catch { users = []; }
  }

  async function load() {
    loading = true;
    detail = null;
    try {
      const q = {};
      if (status) q.status = status;
      if (userId) q.user_id = userId;
      const res = await listOrders(q);
      orders = res.orders ?? [];
    } catch (e) { push(`Orders failed: ${e.message}`); }
    finally { loading = false; }
  }

  async function openDetail(id) {
    try { detail = await getOrder(id); }
    catch (e) { push(`Order failed: ${e.message}`); }
  }

  async function doCancel(id) {
    haptic(10);
    try {
      const res = await cancelOrder(id);
      push(`Cancelled #${id} — balance ₹${(res.balance_paise / 100).toFixed(2)}`);
      detail = null;
      await load();
    } catch (e) { push(`Cancel failed: ${e.message}`); }
  }

  onMount(async () => {
    await loadUsers();
    await load();
  });
</script>

<section class="panel">
  <div class="filter-row">
    <div class="field">
      <label for="orders-status">Status</label>
      <select id="orders-status" bind:value={status} onchange={load}>
        <option value="">all</option>
        <option value="pending">pending</option>
        <option value="completed">completed</option>
        <option value="cancelled">cancelled</option>
      </select>
    </div>
    <div class="field">
      <label for="orders-userid">User</label>
      <select id="orders-userid" bind:value={userId} onchange={load}>
        <option value="">All users</option>
        {#each users as u}
          <option value={u.id}>{u.name} ({u.roll_no})</option>
        {/each}
      </select>
    </div>
    <div class="field actions">
      <span aria-hidden="true">&nbsp;</span>
      <button class="primary" onclick={load}>Refresh</button>
    </div>
  </div>

  <div class="body">
    <div class="list">
      {#if loading}
        <p style="color:#999;padding:1rem">Loading…</p>
      {:else}
        <table>
          <thead>
            <tr><th>#</th><th>User</th><th>Status</th><th>Total</th><th>At</th><th></th></tr>
          </thead>
          <tbody>
            {#each orders as o (o.id)}
              <tr class:selected={detail?.id === o.id} onclick={() => openDetail(o.id)}>
                <td>{o.id}</td>
                <td>{nameOf(o.user_id)}</td>
                <td><span class="badge {o.status}">{o.status}</span></td>
                <td>₹{(o.total_paise / 100).toFixed(2)}</td>
                <td class="ts">{o.created_at.slice(0,16).replace('T',' ')}</td>
                <td>
                  {#if o.status === 'pending'}
                    <button class="danger sm" aria-label={`Cancel order ${o.id}`} onclick={(e) => { e.stopPropagation(); doCancel(o.id); }}>
                      <X size={14} strokeWidth={2.5} />
                    </button>
                  {/if}
                </td>
              </tr>
            {:else}
              <tr><td colspan="6" style="color:#999;text-align:center">No orders</td></tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>

    {#if detail}
      <div class="detail">
        <div class="detail-header">
          <strong>Order #{detail.id}</strong>
          <span class="badge {detail.status}">{detail.status}</span>
          {#if detail.status === 'pending'}
            <button class="danger sm" onclick={() => doCancel(detail.id)}>Cancel</button>
          {/if}
        </div>
        <table class="detail-table">
          <thead><tr><th>Item</th><th>Qty</th><th>Unit</th><th>Total</th></tr></thead>
          <tbody>
            {#each detail.items as item}
              <tr>
                <td>{item.item_name}</td>
                <td>{item.quantity}</td>
                <td>₹{(item.unit_price_paise / 100).toFixed(2)}</td>
                <td>₹{(item.line_total_paise / 100).toFixed(2)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
        <div class="detail-total">Total: ₹{(detail.total_paise / 100).toFixed(2)}</div>
      </div>
    {/if}
  </div>
</section>

<style>
  .panel { padding: 24px; }

  .filter-row { display: flex; gap: .75rem; align-items: flex-end; margin-bottom: 1rem; flex-wrap: wrap; }
  .field { flex: 1; max-width: 190px; }
  .field.actions { flex: 0; }

  .body { display: flex; gap: 1rem; align-items: flex-start; }
  .list {
    flex: 1;
    overflow-x: auto;
    background: #fff;
    border: 1px solid #ececec;
    border-radius: 16px;
  }

  table { width: 100%; border-collapse: collapse; font-size: .8rem; }
  th, td { padding: .55rem .75rem; text-align: left; border-bottom: 1px solid #f0f0f0; }
  th { color: #777; font-weight: 600; font-size: .72rem; }
  tbody tr:last-child td { border-bottom: none; }
  tr:hover td { background: #fafafa; cursor: pointer; }
  tr.selected td { background: #eef7f1; }
  .ts { font-size: .7rem; color: #999; white-space: nowrap; }
  .sm { display: inline-grid; place-items: center; width: 26px; height: 26px; padding: 0; border-radius: 8px; }

  .detail {
    width: 320px;
    background: #fff;
    border: 1px solid #ececec;
    border-radius: 16px;
    padding: 1rem;
    font-size: .8rem;
    flex-shrink: 0;
  }
  .detail-header { display: flex; align-items: center; gap: .4rem; margin-bottom: .5rem; }
  .detail-table { width: 100%; margin-bottom: .5rem; }
  .detail-total { text-align: right; font-weight: 800; color: #1f6f45; }

  @media (max-width: 760px) { .body { flex-direction: column; } .detail { width: 100%; } }
</style>
