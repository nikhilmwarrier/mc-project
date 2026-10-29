<script>
  import { onMount } from 'svelte';
  import { listOrders, getOrder, cancelOrder, auth } from '../api.js';

  let { log = $bindable([]) } = $props();
  const push = (msg) => { log = [msg, ...log].slice(0, 20); };

  let status  = $state('pending');
  let userId  = $state('');
  let orders  = $state([]);
  let detail  = $state(null);
  let loading = $state(false);

  async function load() {
    loading = true;
    detail = null;
    try {
      const q = {};
      if (status) q.status = status;
      if (userId && auth.isAdmin) q.user_id = userId;
      const res = await listOrders(q);
      orders = res.orders ?? [];
    } catch (e) { push(`❌ orders: ${e.message}`); }
    finally { loading = false; }
  }

  async function openDetail(id) {
    try { detail = await getOrder(id); }
    catch (e) { push(`❌ get order: ${e.message}`); }
  }

  async function doCancel(id) {
    try {
      const res = await cancelOrder(id);
      push(`🚫 cancelled #${id} — balance ₹${(res.balance_paise / 100).toFixed(2)}`);
      detail = null;
      await load();
    } catch (e) { push(`❌ cancel: ${e.message}`); }
  }

  onMount(load);
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
    {#if auth.isAdmin}
      <div class="field">
        <label for="orders-userid">User ID</label>
        <input id="orders-userid" type="number" bind:value={userId} placeholder="all users" />
      </div>
    {/if}
    <div class="field actions">
      <span aria-hidden="true">&nbsp;</span>
      <button class="primary" onclick={load}>Refresh</button>
    </div>
  </div>

  <div class="body">
    <div class="list">
      {#if loading}
        <p style="color:#8b949e;padding:1rem">Loading…</p>
      {:else}
        <table>
          <thead>
            <tr><th>#</th><th>User</th><th>Status</th><th>Total</th><th>At</th><th></th></tr>
          </thead>
          <tbody>
            {#each orders as o (o.id)}
              <tr class:selected={detail?.id === o.id} onclick={() => openDetail(o.id)}>
                <td>{o.id}</td>
                <td>{o.user_id}</td>
                <td><span class="badge {o.status}">{o.status}</span></td>
                <td>₹{(o.total_paise / 100).toFixed(2)}</td>
                <td class="ts">{o.created_at.slice(0,16).replace('T',' ')}</td>
                <td>
                  {#if o.status === 'pending'}
                    <button class="danger sm" onclick={(e) => { e.stopPropagation(); doCancel(o.id); }}>✕</button>
                  {/if}
                </td>
              </tr>
            {:else}
              <tr><td colspan="6" style="color:#8b949e;text-align:center">No orders</td></tr>
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
  .panel { padding: 1rem 1.5rem; }

  .filter-row { display: flex; gap: .75rem; align-items: flex-end; margin-bottom: 1rem; }
  .field { flex: 1; max-width: 160px; }
  .field.actions { flex: 0; }

  .body { display: flex; gap: 1rem; align-items: flex-start; }
  .list { flex: 1; overflow-x: auto; }

  table { width: 100%; border-collapse: collapse; font-size: .8rem; }
  th, td { padding: .4rem .6rem; text-align: left; border-bottom: 1px solid #21262d; }
  th { color: #8b949e; font-weight: 600; }
  tr:hover td { background: #161b22; cursor: pointer; }
  tr.selected td { background: #1c2128; }
  .ts { font-size: .7rem; color: #8b949e; white-space: nowrap; }
  .sm { padding: .15rem .35rem; font-size: .7rem; }

  .detail {
    width: 320px;
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 6px;
    padding: .75rem 1rem;
    font-size: .8rem;
    flex-shrink: 0;
  }
  .detail-header { display: flex; align-items: center; gap: .4rem; margin-bottom: .5rem; }
  .detail-table { width: 100%; margin-bottom: .5rem; }
  .detail-total { text-align: right; font-weight: 700; color: #58a6ff; }
</style>
