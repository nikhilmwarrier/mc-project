<script>
  import { onMount } from 'svelte';
  import { createUser, topUp, getWallet, getUsers } from '../api.js';
  import { haptic } from '../lib/haptics.js';

  let { log = $bindable([]) } = $props();
  const push = (msg) => { log = [msg, ...log].slice(0, 20); };

  let users = $state([]);

  function nameOf(id) {
    const u = users.find(x => String(x.id) === String(id));
    return u ? `${u.name} (${u.roll_no})` : `#${id}`;
  }

  async function loadUsers() {
    try { users = (await getUsers()).users ?? []; }
    catch (e) { push(`Users failed: ${e.message}`); }
  }

  // Create user
  let newUser = $state({ roll_no: '', name: '', phone: '', pin: '' });
  async function addUser() {
    haptic(8);
    try {
      const body = { ...newUser };
      if (!body.pin) delete body.pin;
      const res = await createUser(body);
      push(`User created id=${res.id}`);
      newUser = { roll_no: '', name: '', phone: '', pin: '' };
      await loadUsers();
    } catch (e) { push(`Create failed: ${e.message}`); }
  }

  // Top-up — entered in rupees, truncated to two decimals, sent as paise.
  let topupId = $state('');
  let topupRupees = $state('');

  function rupeesToPaise(value) {
    const clean = String(value).trim().replace(/[₹,\s]/g, '');
    const m = /^(\d+)(?:\.(\d+))?$/.exec(clean);
    if (!m) return 0;
    const fraction = (m[2] ?? '').slice(0, 2).padEnd(2, '0');
    return Number(m[1]) * 100 + Number(fraction);
  }

  async function doTopup() {
    if (!topupId) { push('Select a user'); return; }
    const amountPaise = rupeesToPaise(topupRupees);
    if (amountPaise <= 0) { push('Enter a valid amount'); return; }
    haptic(10);
    try {
      const res = await topUp(topupId, amountPaise);
      push(`${nameOf(topupId)} → ₹${(res.balance_paise / 100).toFixed(2)}`);
      topupRupees = '';
    } catch (e) { push(`Top-up failed: ${e.message}`); }
  }

  // Wallet
  let walletId = $state('');
  let walletBalance = $state(null);
  async function checkWallet() {
    if (!walletId) { push('Select a user'); return; }
    haptic(5);
    try {
      const res = await getWallet(walletId);
      walletBalance = res.balance_paise;
    } catch (e) { push(`Wallet failed: ${e.message}`); }
  }

  onMount(loadUsers);
</script>

<section class="panel">
  <div class="grid">
    <div class="box">
      <h2>Create user</h2>
      <div class="field"><label for="user-roll">Roll no</label><input id="user-roll" bind:value={newUser.roll_no} /></div>
      <div class="field"><label for="user-name">Name</label><input id="user-name" bind:value={newUser.name} /></div>
      <div class="field"><label for="user-phone">Phone</label><input id="user-phone" bind:value={newUser.phone} /></div>
      <div class="field"><label for="user-pin">PIN</label><input id="user-pin" type="password" bind:value={newUser.pin} /></div>
      <button class="primary" onclick={addUser}>Create</button>
    </div>

    <div class="box">
      <h2>Top up</h2>
      <div class="field">
        <label for="topup-id">User</label>
        <select id="topup-id" bind:value={topupId}>
          <option value="">— select user —</option>
          {#each users as u}
            <option value={u.id}>{u.name} ({u.roll_no})</option>
          {/each}
        </select>
      </div>
      <div class="field">
        <label for="topup-amt">Amount (₹)</label>
        <input id="topup-amt" type="text" inputmode="decimal" bind:value={topupRupees} placeholder="eg: ₹5000.00" />
      </div>
      <button class="primary" onclick={doTopup}>Top up</button>
    </div>

    <div class="box">
      <h2>Wallet</h2>
      <div class="field">
        <label for="wallet-id">User</label>
        <select id="wallet-id" bind:value={walletId}>
          <option value="">— select user —</option>
          {#each users as u}
            <option value={u.id}>{u.name} ({u.roll_no})</option>
          {/each}
        </select>
      </div>
      <button class="primary" onclick={checkWallet}>Check</button>
      {#if walletBalance !== null}
        <p class="balance">₹{(walletBalance / 100).toFixed(2)}</p>
      {/if}
    </div>
  </div>
</section>

<style>
  .panel { padding: 24px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 16px; }
  .box {
    background: #fff;
    border: 1px solid #ececec;
    border-radius: 16px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .field { margin-bottom: 2px; }
  .balance { font-size: 1.5rem; color: #2e8b57; font-weight: 800; margin: .25rem 0 0; }
</style>
