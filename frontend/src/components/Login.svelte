<script>
  import { auth, getMe } from '../api.js';

  let { onlogin } = $props();

  let principal = $state('');
  let password  = $state('');
  let error     = $state('');
  let loading   = $state(false);

  async function submit() {
    error = '';
    loading = true;
    try {
      await auth.login(principal, password);
      const me = await getMe();
      onlogin(me);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-wrap">
  <div class="login-box">
    <h1>🍽 Canteen</h1>
    <p class="sub">staff / test rig</p>

    <form onsubmit={(e) => { e.preventDefault(); submit(); }}>
      <div class="field">
        <label for="principal">Principal</label>
        <input id="principal" bind:value={principal} autocomplete="username" required />
      </div>
      <div class="field">
        <label for="password">Password / PIN</label>
        <input id="password" type="password" bind:value={password} autocomplete="current-password" required />
      </div>

      {#if error}
        <p class="err">{error}</p>
      {/if}

      <button type="submit" class="primary" disabled={loading} style="width:100%;margin-top:.5rem">
        {loading ? 'Logging in…' : 'Login'}
      </button>
    </form>
  </div>
</div>

<style>
  .login-wrap {
    display: flex; align-items: center; justify-content: center;
    min-height: 100vh;
  }
  .login-box {
    width: 320px;
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 2rem;
  }
  h1 { text-align: center; margin-bottom: .25rem; }
  .sub { text-align: center; color: #8b949e; font-size: .8rem; margin: 0 0 1.5rem; }
  .field { margin-bottom: .75rem; }
  .err { color: #f85149; font-size: .8rem; margin: .5rem 0 0; }
</style>
