<script>
  import { createUser, topUp, getWallet, auth } from '../api.js';

  let { log = $bindable([]) } = $props();
  const push = (msg) => { log = [msg, ...log].slice(0, 20); };

  // Create user
  let newUser = $state({ roll_no: '', name: '', phone: '', pin: '' });
  async function addUser() {
    try {
      const body = { ...newUser };
      if (!body.pin) delete body.pin;
      const res = await createUser(body);
      push(`✅ user created id=${res.id}`);
      newUser = { roll_no: '', name: '', phone: '', pin: '' };
    } catch (e) { push(`❌ create user: ${e.message}`); }
  }

  // Top-up
  let topupId  = $state('');
  let topupAmt = $state('');
  async function doTopup() {
    try {
      const res = await topUp(topupId, parseInt(topupAmt, 10));
      push(`💰 user ${topupId} → ₹${(res.balance_paise / 100).toFixed(2)}`);
      topupAmt = '';
    } catch (e) { push(`❌ topup: ${e.message}`); }
  }

  // Wallet
  let walletId      = $state('');
  let walletBalance = $state(null);
  async function checkWallet() {
    const id = auth.isAdmin ? walletId : undefined;
    if (auth.isAdmin && !id) { push('⚠ enter a user id'); return; }
    try {
      const res = await getWallet(id);
      walletBalance = res.balance_paise;
    } catch (e) { push(`❌ wallet: ${e.message}`); }
  }
</script>

<section class="panel">
  <div class="grid">
    <!-- Create user -->
    <div class="box">
      <h2>Create User</h2>
      <div class="field"><label for="user-roll">Roll No</label><input id="user-roll" bind:value={newUser.roll_no} /></div>
      <div class="field"><label for="user-name">Name</label><input id="user-name" bind:value={newUser.name} /></div>
      <div class="field"><label for="user-phone">Phone</label><input id="user-phone" bind:value={newUser.phone} /></div>
      <div class="field"><label for="user-pin">PIN (optional)</label><input id="user-pin" type="password" bind:value={newUser.pin} /></div>
      <button class="primary" onclick={addUser} style="margin-top:.5rem">Create</button>
    </div>

    <!-- Top-up -->
    <div class="box">
      <h2>Top-Up Wallet</h2>
      <div class="field"><label for="topup-id">User ID</label><input id="topup-id" type="number" bind:value={topupId} /></div>
      <div class="field"><label for="topup-amt">Amount (paise)</label><input id="topup-amt" type="number" bind:value={topupAmt} placeholder="10000 = ₹100" /></div>
      <button class="primary" onclick={doTopup} style="margin-top:.5rem">Top Up</button>
    </div>

    <!-- Wallet check -->
    <div class="box">
      <h2>Check Wallet</h2>
      {#if auth.isAdmin}
        <div class="field"><label for="wallet-id">User ID</label><input id="wallet-id" type="number" bind:value={walletId} /></div>
      {/if}
      <button class="primary" onclick={checkWallet} style="margin-top:.5rem">Check</button>
      {#if walletBalance !== null}
        <p class="balance">₹{(walletBalance / 100).toFixed(2)}</p>
      {/if}
    </div>
  </div>
</section>

<style>
  .panel { padding: 1rem 1.5rem; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 1rem; }
  .box {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 6px;
    padding: .75rem 1rem;
    display: flex;
    flex-direction: column;
    gap: .4rem;
  }
  .field { margin-bottom: .25rem; }
  .balance { font-size: 1.5rem; color: #3fb950; font-weight: 700; margin: .75rem 0 0; }
</style>
