<script>
  import { onMount } from 'svelte';
  import { auth, getMe } from './api.js';
  import { navigate, pathNow } from './lib/router.js';
  import Login from './components/Login.svelte';
  import Storefront from './components/Storefront.svelte';
  import AdminApp from './AdminApp.svelte';

  let path = $state(pathNow());
  let me = $state(null);
  let booting = $state(!!auth.token);

  const isAdminRoute = $derived(path === '/admin' || path.startsWith('/admin/'));
  // Everything else is the customer storefront; unknown paths normalise there.
  const isKnownRoute = $derived(path === '/' || isAdminRoute);

  onMount(() => {
    const sync = () => { path = pathNow(); };
    window.addEventListener('popstate', sync);

    (async () => {
      if (auth.token) {
        try { me = await getMe(); } catch { auth.logout(); }
      }
      booting = false;
    })();

    return () => window.removeEventListener('popstate', sync);
  });

  $effect(() => {
    if (!isKnownRoute) { navigate('/', true); return; }
    if (!me) return;
    if (me.role === 'admin' && !isAdminRoute) navigate('/admin', true);
    if (me.role !== 'admin' && isAdminRoute) navigate('/', true);
  });

  function handleLogin(user) {
    me = user;
    navigate(user.role === 'admin' ? '/admin' : '/');
  }

  function logout() {
    auth.logout();
    me = null;
    navigate('/');
  }
</script>

{#if booting}
  <div class="splash">🍽</div>
{:else if !me}
  <Login onlogin={handleLogin} />
{:else if isAdminRoute}
  <AdminApp {me} onlogout={logout} />
{:else}
  <Storefront {me} onlogout={logout} />
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
