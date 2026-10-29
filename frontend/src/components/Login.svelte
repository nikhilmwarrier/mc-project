<script>
  import { auth, getMe } from '../api.js';

  let { onlogin, scope = 'customer', requireRole = null } = $props();

  // svelte-ignore state_referenced_locally
  const saved = auth.savedCredentials(scope);
  let principal = $state(saved?.principal ?? '');
  let password  = $state(saved?.password ?? '');
  let error     = $state('');
  let loading   = $state(false);

  async function submit() {
    error = '';
    loading = true;
    try {
      await auth.login(principal, password, scope);
      const me = await getMe(scope);
      if (requireRole && me.role !== requireRole) {
        auth.logout(scope);
        auth.clearCredentials(scope);
        error = 'Admin access only';
        return;
      }
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
    <div class="brand">CAMPUS <span>KITCHEN</span></div>
    <p class="sub">{scope === 'admin' ? 'Staff sign in' : 'Sign in with your roll number and PIN'}</p>

    <form onsubmit={(e) => { e.preventDefault(); submit(); }}>
      <div class="field">
        <label for="principal">Roll number / staff ID</label>
        <input id="principal" bind:value={principal} autocomplete="username" required />
      </div>
      <div class="field">
        <label for="password">PIN / password</label>
        <input id="password" type="password" bind:value={password} autocomplete="current-password" required />
      </div>

      {#if error}
        <p class="err">{error}</p>
      {/if}

      <button type="submit" disabled={loading}>
        {loading ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  </div>
</div>

<style>
  .login-wrap {
    display: flex; align-items: center; justify-content: center;
    min-height: 100vh; padding: 1.5rem;
    background: linear-gradient(180deg, #fdfdfd 0%, #f4f4f4 100%);
    color: #1c1c1c;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  }
  .login-box {
    width: 360px;
    max-width: 100%;
    background: #fff;
    border: 1px solid #e8e8e8;
    border-radius: 18px;
    padding: 2rem;
    box-shadow: 0 18px 50px rgba(0, 0, 0, .08);
  }
  .brand { text-align: center; font-size: 1.6rem; font-weight: 800; letter-spacing: -.02em; }
  .brand span { color: #2e8b57; }
  .sub { text-align: center; color: #7a7a7a; font-size: .85rem; margin: .4rem 0 1.6rem; }
  .field { margin-bottom: .9rem; }
  label { display: block; font-size: .75rem; font-weight: 600; color: #555; margin-bottom: .3rem; }
  input {
    width: 100%;
    padding: .65rem .75rem;
    border: 1px solid #ddd;
    border-radius: 10px;
    font-size: .9rem;
    font-family: inherit;
    outline: none;
  }
  input:focus { border-color: #2e8b57; box-shadow: 0 0 0 3px rgba(46, 139, 87, .12); }
  .err { color: #cf2e2e; font-size: .8rem; margin: .5rem 0 0; }
  button {
    width: 100%;
    margin-top: 1rem;
    padding: .75rem;
    border: none;
    border-radius: 12px;
    background: #2e8b57;
    color: #fff;
    font-size: .95rem;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
  }
  button:hover:not(:disabled) { background: #26764a; }
  button:disabled { opacity: .6; cursor: default; }
</style>
