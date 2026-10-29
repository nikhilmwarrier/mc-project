<script>
  import { onMount } from 'svelte';
  import { auth, getMe } from './api.js';
  import { navigate, pathNow } from './lib/router.js';
  import Login from './components/Login.svelte';
  import Storefront from './components/Storefront.svelte';
  import AdminApp from './AdminApp.svelte';

  let path = $state(pathNow());
  let me = $state(null);
  let booting = $state(true);

  const isAdminRoute = $derived(path === '/admin' || path.startsWith('/admin/'));
  const isKnownRoute = $derived(path === '/' || isAdminRoute);

  async function loadRoute() {
    if (!isKnownRoute) { navigate('/', true); return; }

    booting = true;
    me = null;
    const scope = isAdminRoute ? 'admin' : 'customer';
    if (auth.hasSession(scope)) {
      try { me = await getMe(scope); }
      catch { auth.logout(scope); }
    }
    booting = false;
  }

  onMount(() => {
    const sync = () => { path = pathNow(); };
    window.addEventListener('popstate', sync);
    loadRoute();
    return () => window.removeEventListener('popstate', sync);
  });

  // Route changes (link clicks, back/forward) reload the identity for that
  // route's own session. Initial load is handled above.
  let mounted = false;
  $effect(() => {
    path;
    if (!mounted) { mounted = true; return; }
    loadRoute();
  });

  function handleLogin(user) {
    if (isAdminRoute) { me = user; return; }
    // An admin signing in from the storefront belongs on the dashboard.
    if (user.role === 'admin') { auth.promoteToAdmin(); navigate('/admin'); return; }
    me = user;
  }

  function logout() {
    auth.logout(isAdminRoute ? 'admin' : 'customer');
    me = null;
  }
</script>

{#if booting}
  <div class="splash">🍽</div>
{:else if isAdminRoute}
  {#if me}
    <AdminApp {me} onlogout={logout} />
  {:else}
    <Login scope="admin" requireRole="admin" onlogin={handleLogin} />
  {/if}
{:else}
  {#if me}
    <Storefront {me} onlogout={logout} />
  {:else}
    <Login scope="customer" onlogin={handleLogin} />
  {/if}
{/if}

<style>
  .splash {
    min-height: 100vh;
    display: grid;
    place-items: center;
    font-size: 3rem;
    background: #f7f7f7;
  }
</style>
