<script>
  import QueuePanel  from './components/QueuePanel.svelte';
  import MenuPanel   from './components/MenuPanel.svelte';
  import OrdersPanel from './components/OrdersPanel.svelte';
  import UsersPanel  from './components/UsersPanel.svelte';

  let { me, onlogout } = $props();

  const TABS = [
    { key: 'queue',  label: '📡 Queues' },
    { key: 'orders', label: '📋 Orders' },
    { key: 'menu',   label: '🍛 Menu' },
    { key: 'users',  label: '👤 Users' },
  ];

  let tab = $state('queue');
  let log = $state([]);
</script>

<div class="shell">
  <nav class="sidebar">
    <div class="logo">🍽 Canteen <span class="tag">admin</span></div>

    {#each TABS as t}
      <button class="nav-btn" class:active={tab === t.key} onclick={() => tab = t.key}>
        {t.label}
      </button>
    {/each}

    <div class="spacer"></div>

    <div class="who">
      <span class="badge admin">{me.role}</span>
      <span class="username">{me.roll_no ?? me.name}</span>
    </div>
    <button class="nav-btn logout" onclick={onlogout}>← Logout</button>

    <div class="log-box">
      <div class="log-title">— log —</div>
      {#each log as entry}
        <div class="log-entry">{entry}</div>
      {:else}
        <div class="log-entry empty">no activity yet</div>
      {/each}
    </div>
  </nav>

  <main class="main">
    {#if tab === 'queue'}
      <QueuePanel {me} bind:log />
    {:else if tab === 'orders'}
      <OrdersPanel bind:log />
    {:else if tab === 'menu'}
      <MenuPanel bind:log />
    {:else if tab === 'users'}
      <UsersPanel bind:log />
    {/if}
  </main>
</div>

<style>
  .shell { display: flex; height: 100vh; overflow: hidden; }

  .sidebar {
    width: 200px;
    min-width: 200px;
    background: #010409;
    border-right: 1px solid #21262d;
    display: flex;
    flex-direction: column;
    padding: .75rem .5rem;
    gap: .2rem;
    overflow-y: auto;
  }

  .logo {
    font-size: .9rem;
    font-weight: 700;
    color: #58a6ff;
    letter-spacing: .1em;
    padding: .3rem .6rem .6rem;
  }
  .tag { font-size: .6rem; color: #8b949e; letter-spacing: .1em; text-transform: uppercase; }

  .nav-btn {
    width: 100%;
    text-align: left;
    background: transparent;
    border: none;
    border-radius: 5px;
    padding: .4rem .6rem;
    font-size: .8rem;
    color: #8b949e;
    cursor: pointer;
    transition: background .1s, color .1s;
    font-family: inherit;
  }
  .nav-btn:hover  { background: #21262d; color: #e6edf3; }
  .nav-btn.active { background: #1f3556; color: #58a6ff; }
  .nav-btn.logout { color: #f85149; margin-top: .25rem; }
  .nav-btn.logout:hover { background: #3d1f1f; }

  .spacer { flex: 1; }

  .who {
    display: flex;
    align-items: center;
    gap: .4rem;
    padding: .3rem .6rem;
    font-size: .75rem;
  }
  .username { color: #c9d1d9; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .log-box {
    margin-top: .5rem;
    border-top: 1px solid #21262d;
    padding-top: .5rem;
    font-size: .65rem;
    display: flex;
    flex-direction: column;
    gap: .1rem;
    max-height: 200px;
    overflow-y: auto;
  }
  .log-title { color: #30363d; text-transform: uppercase; letter-spacing: .08em; margin-bottom: .2rem; }
  .log-entry { color: #6e7681; word-break: break-all; }
  .log-entry.empty { color: #30363d; }

  .main { flex: 1; overflow-y: auto; background: #0d1117; }
</style>
