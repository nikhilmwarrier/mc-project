<script>
  import { onMount } from 'svelte';
  import { getMenu, createMenuItem, updateMenuItem, deleteMenuItem, patchMenuItem } from '../api.js';

  let { log = $bindable([]) } = $props();
  const push = (msg) => { log = [msg, ...log].slice(0, 20); };

  let items   = $state([]);
  let loading = $state(false);
  let editId  = $state(null);
  let form    = $state({ item_name: '', code: '', price_paise: '', is_available: true });

  async function load() {
    loading = true;
    try { const m = await getMenu(true); items = m.items ?? []; }
    catch (e) { push(`❌ load menu: ${e.message}`); }
    finally { loading = false; }
  }

  async function save() {
    const body = {
      item_name:    form.item_name,
      code:         form.code,
      price_paise:  parseInt(form.price_paise, 10),
      is_available: form.is_available,
    };
    try {
      if (editId) {
        await updateMenuItem(editId, body);
        push(`✅ updated ${form.code}`);
      } else {
        const r = await createMenuItem(body);
        push(`✅ created ${form.code} → id ${r.id}`);
      }
      resetForm();
      await load();
    } catch (e) { push(`❌ save: ${e.message}`); }
  }

  async function toggleAvail(item) {
    try {
      await patchMenuItem(item.id, { is_available: !item.is_available });
      push(`✅ ${item.code} → ${!item.is_available ? 'available' : 'unavailable'}`);
      await load();
    } catch (e) { push(`❌ toggle: ${e.message}`); }
  }

  async function del(item) {
    if (!confirm(`Delete ${item.item_name}?`)) return;
    try {
      await deleteMenuItem(item.id);
      push(`🗑 deleted ${item.code}`);
      await load();
    } catch (e) { push(`❌ delete: ${e.message}`); }
  }

  function startEdit(item) {
    editId = item.id;
    form = { item_name: item.item_name, code: item.code, price_paise: item.price_paise, is_available: item.is_available };
  }

  function resetForm() {
    editId = null;
    form = { item_name: '', code: '', price_paise: '', is_available: true };
  }

  onMount(load);
</script>

<section class="panel">
  <div class="form-box">
    <h2>{editId ? `Edit #${editId}` : 'New Item'}</h2>
    <div class="form-row">
      <div class="field">
        <label for="menu-name">Name</label>
        <input id="menu-name" bind:value={form.item_name} placeholder="e.g. Rice Plate" />
      </div>
      <div class="field short">
        <label for="menu-code">Code</label>
        <input id="menu-code" bind:value={form.code} placeholder="MEAL" />
      </div>
      <div class="field short">
        <label for="menu-price">Price (paise)</label>
        <input id="menu-price" type="number" min="0" bind:value={form.price_paise} placeholder="4500" />
      </div>
      <div class="field check">
        <span aria-hidden="true">&nbsp;</span>
        <label class="toggle">
          <input type="checkbox" bind:checked={form.is_available} />
          Available
        </label>
      </div>
      <div class="field actions">
        <span aria-hidden="true">&nbsp;</span>
        <div class="btn-row">
          <button class="primary" onclick={save}>{editId ? 'Update' : 'Add'}</button>
          {#if editId}<button onclick={resetForm}>Cancel</button>{/if}
        </div>
      </div>
    </div>
  </div>

  {#if loading}
    <p style="color:#999;padding:1rem">Loading…</p>
  {:else}
    <div class="table-card">
    <table>
      <thead>
        <tr><th>ID</th><th>Name</th><th>Code</th><th>Price</th><th>Avail</th><th></th></tr>
      </thead>
      <tbody>
        {#each items as item (item.id)}
          <tr class:inactive={!item.is_available}>
            <td>{item.id}</td>
            <td>{item.item_name}</td>
            <td><code>{item.code}</code></td>
            <td>₹{(item.price_paise / 100).toFixed(2)}</td>
            <td><span class="badge {item.is_available ? 'completed' : 'cancelled'}">{item.is_available ? 'yes' : 'no'}</span></td>
            <td class="row-actions">
              <button onclick={() => startEdit(item)}>Edit</button>
              <button onclick={() => toggleAvail(item)}>{item.is_available ? 'Disable' : 'Enable'}</button>
              <button class="danger" onclick={() => del(item)}>Del</button>
            </td>
          </tr>
        {:else}
          <tr><td colspan="6" style="color:#999;text-align:center">No items</td></tr>
        {/each}
      </tbody>
    </table>
    </div>
  {/if}
</section>

<style>
  .panel { padding: 1.25rem 1.5rem; }

  .form-box {
    background: #fff;
    border: 1px solid #ececec;
    border-radius: 16px;
    padding: 1rem;
    margin-bottom: 1rem;
  }
  .form-row { display: flex; gap: .75rem; align-items: flex-end; flex-wrap: wrap; }
  .field { flex: 1; min-width: 130px; }
  .field.short { max-width: 130px; }
  .field.check { max-width: 120px; }
  .field.actions { flex: 0; }
  .toggle { display: flex; gap: .3rem; align-items: center; cursor: pointer; font-size: .8rem; color: #555; }
  .btn-row { display: flex; gap: .3rem; }

  .table-card { background: #fff; border: 1px solid #ececec; border-radius: 16px; overflow: hidden; }
  table { width: 100%; border-collapse: collapse; font-size: .8rem; }
  th, td { padding: .55rem .75rem; text-align: left; border-bottom: 1px solid #f0f0f0; }
  th { color: #777; font-weight: 600; font-size: .72rem; }
  tbody tr:last-child td { border-bottom: none; }
  tr.inactive td { opacity: .45; }
  .row-actions { display: flex; gap: .3rem; }
  code { background: #f4f4f4; padding: .1rem .3rem; border-radius: 5px; font-size: .75rem; }
</style>
