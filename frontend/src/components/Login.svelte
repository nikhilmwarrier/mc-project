<script>
  import { Eye, EyeOff } from 'lucide-svelte';
  import { auth, getMe } from '../api.js';
  import { haptic } from '../lib/haptics.js';

  let { onlogin, scope = 'customer', requireRole = null } = $props();

  // svelte-ignore state_referenced_locally
  const saved = auth.savedCredentials(scope);
  let principal = $state(saved?.principal ?? '');
  let password  = $state(saved?.password ?? '');
  let showPassword = $state(false);
  let error     = $state('');
  let loading   = $state(false);

  async function submit() {
    error = '';
    loading = true;
    haptic(8);
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
        <label for="principal">{scope === 'admin' ? 'Staff ID' : 'Roll number'}</label>
        <input id="principal" bind:value={principal} autocomplete="username" required />
      </div>
      <div class="field">
        <label for="password">PIN / password</label>
        <div class="password">
          <input
            id="password"
            type={showPassword ? 'text' : 'password'}
            bind:value={password}
            autocomplete="current-password"
            required
          />
          <button
            type="button"
            class="reveal"
            onclick={() => { haptic(4); showPassword = !showPassword; }}
            aria-label={showPassword ? 'Hide password' : 'Show password'}
          >
            {#if showPassword}
              <EyeOff size={17} strokeWidth={2} />
            {:else}
              <Eye size={17} strokeWidth={2} />
            {/if}
          </button>
        </div>
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
    min-height: 100vh; padding: 24px;
    background: linear-gradient(180deg, #fdfdfd 0%, #f4f4f4 100%);
    color: #1c1c1c;
    font-family: var(--font-sans);
  }
  .login-box {
    width: 360px;
    max-width: 100%;
    background: #fff;
    border: 1px solid #e8e8e8;
    border-radius: 18px;
    padding: 32px;
    box-shadow: 0 18px 50px rgba(0, 0, 0, .07);
  }
  .brand { font-family: var(--font-serif); text-align: center; font-size: 1.55rem; font-weight: 700; letter-spacing: -.01em; }
  .brand span { color: #2e8b57; }
  .sub { text-align: center; color: #7a7a7a; font-size: .84rem; margin: 7px 0 26px; }
  .field { margin-bottom: 14px; }
  label { display: block; font-size: .73rem; font-weight: 600; color: #555; margin-bottom: 6px; }
  input {
    width: 100%;
    height: 44px;
    padding: 0 12px;
    border: 1px solid #ddd;
    border-radius: 10px;
    font-size: .9rem;
    font-family: inherit;
    outline: none;
    transition: border-color 160ms ease, box-shadow 160ms ease;
  }
  input:focus { border-color: #2e8b57; box-shadow: 0 0 0 3px rgba(46, 139, 87, .12); }

  .password { position: relative; }
  .password input { padding-right: 44px; }
  .reveal {
    position: absolute;
    right: 4px;
    top: 50%;
    transform: translateY(-50%);
    width: 34px;
    height: 34px;
    display: grid;
    place-items: center;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: #999;
    padding: 0;
  }
  .reveal:hover:not(:disabled) { background: #f2f2f2; color: #333; }
  .reveal:active:not(:disabled) { transform: translateY(-50%) scale(0.94); }

  .err { color: #cf2e2e; font-size: .79rem; margin: 8px 0 0; }
  button[type="submit"] {
    width: 100%;
    height: 44px;
    margin-top: 18px;
    border: none;
    border-radius: 12px;
    background: #2e8b57;
    color: #fff;
    font-size: .93rem;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
  }
  button[type="submit"]:hover:not(:disabled) { background: #26764a; }
  button[type="submit"]:disabled { opacity: .6; cursor: default; }
</style>
