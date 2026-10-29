<script>
  import QueuePanel  from './components/QueuePanel.svelte';
  import MenuPanel   from './components/MenuPanel.svelte';
  import OrdersPanel from './components/OrdersPanel.svelte';
  import UsersPanel  from './components/UsersPanel.svelte';

  let { me, onlogout } = $props();

  const TABS = [
    { key: 'queue',  label: 'Queues' },
    { key: 'orders', label: 'Orders' },
    { key: 'menu',   label: 'Menu' },
    { key: 'users',  label: 'Users' },
  ];

  let tab = $state('queue');
  let log = $state([]);
  let recent = $derived(log.slice(0, 3));
</script>

<div class="admin">
  <header class="topbar">
    <div class="brand">CAMPUS <span>KITCHEN</span><em>admin</em></div>

    <nav class="tabs">
      {#each TABS as t}
        <button class="tab" class:active={tab === t.key} onclick={() => tab = t.key}>{t.label}</button>
      {/each}
    </nav>

    <div class="who">
      <span>{me.roll_no ?? me.name}</span>
      <button class="ghost" onclick={onlogout}>Logout</button>
    </div>
  </header>

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

  {#if recent.length}
    <div class="toasts" aria-live="polite">
      {#each recent as entry}
        <div class="toast">{entry}</div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .admin { min-height: 100vh; background: #f7f7f7; }

  .topbar {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    gap: 1.25rem;
    flex-wrap: wrap;
    padding: .85rem 1.5rem;
    background: rgba(255, 255, 255, .94);
    backdrop-filter: blur(10px);
    border-bottom: 1px solid #ececec;
  }

  .brand { font-size: 1.05rem; font-weight: 800; letter-spacing: -.02em; }
  .brand span { color: #2e8b57; }
  .brand em {
    font-style: normal;
    font-size: .6rem;
    font-weight: 700;
    letter-spacing: .1em;
    text-transform: uppercase;
    color: #999;
    margin-left: .5rem;
  }

  .tabs { display: flex; gap: .25rem; flex: 1; flex-wrap: wrap; }
  .tab {
    border: none;
    background: transparent;
    color: #777;
    font-weight: 600;
    font-size: .82rem;
    padding: .4rem .7rem;
    border-radius: 999px;
  }
  .tab:hover:not(.active) { background: #f2f2f2; color: #333; }
  .tab.active { background: #e8f5ee; color: #1f6f45; }

  .who { display: flex; align-items: center; gap: .75rem; font-size: .78rem; color: #777; }
  .ghost {
    border: none;
    background: transparent;
    color: #999;
    font-size: .78rem;
    padding: .3rem .4rem;
  }
  .ghost:hover { color: #1b1b1b; background: transparent; }

  .main { max-width: 1120px; margin: 0 auto; }

  .toasts {
    position: fixed;
    right: 1rem;
    bottom: 1rem;
    z-index: 30;
    display: flex;
    flex-direction: column;
    gap: .4rem;
    max-width: 340px;
  }
  .toast {
    background: #1b1b1b;
    color: #f4f4f4;
    font-size: .72rem;
    line-height: 1.35;
    border-radius: 10px;
    padding: .5rem .7rem;
    box-shadow: 0 8px 24px rgba(0, 0, 0, .18);
    word-break: break-word;
  }

  @media (max-width: 640px) {
    .topbar { gap: .5rem; padding: .7rem 1rem; }
    .tabs { order: 3; flex-basis: 100%; }
  }
</style>
