<script>
  import { ListOrdered, ClipboardList, UtensilsCrossed, Users, LogOut, UserRound } from 'lucide-svelte';
  import { haptic } from './lib/haptics.js';
  import QueuePanel  from './components/QueuePanel.svelte';
  import MenuPanel   from './components/MenuPanel.svelte';
  import OrdersPanel from './components/OrdersPanel.svelte';
  import UsersPanel  from './components/UsersPanel.svelte';

  let { me, onlogout } = $props();

  const TABS = [
    { key: 'queue',  label: 'Queues', icon: ListOrdered },
    { key: 'orders', label: 'Orders', icon: ClipboardList },
    { key: 'menu',   label: 'Menu',   icon: UtensilsCrossed },
    { key: 'users',  label: 'Users',  icon: Users },
  ];

  let tab = $state('queue');
  let log = $state([]);
  let recent = $derived(log.slice(0, 3));
</script>

<div class="admin">
  <header class="topbar">
    <div class="brand">CAMPUS <span>KITCHEN</span></div>

    <nav class="tabs">
      {#each TABS as t}
        {@const Icon = t.icon}
        <button
          class="tab"
          class:active={tab === t.key}
          onclick={() => { haptic(4); tab = t.key; }}
        >
          <Icon size={15} strokeWidth={2} aria-hidden="true" />
          {t.label}
        </button>
      {/each}
    </nav>

    <div class="who">
      <UserRound size={14} strokeWidth={2} aria-hidden="true" />
      <span>{me.roll_no ?? me.name}</span>
      <button class="icon-btn" onclick={onlogout} aria-label="Log out" title="Log out">
        <LogOut size={17} strokeWidth={2} />
      </button>
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
    gap: 20px;
    flex-wrap: wrap;
    min-height: 64px;
    padding: 8px 24px;
    background: rgba(255, 255, 255, .94);
    backdrop-filter: blur(12px);
    border-bottom: 1px solid #ececec;
  }

  .brand { font-size: 1.02rem; font-weight: 800; letter-spacing: -.02em; }
  .brand span { color: #2e8b57; }

  .tabs { display: flex; gap: 4px; flex: 1; flex-wrap: wrap; }
  .tab {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    border: none;
    background: transparent;
    color: #777;
    font-weight: 600;
    font-size: .8rem;
    padding: 8px 12px;
    border-radius: 999px;
    transition: background-color 160ms ease, color 160ms ease, transform 160ms var(--ease-out);
  }
  .tab:hover:not(.active) { background: #f2f2f2; color: #333; }
  .tab.active { background: #e8f5ee; color: #1f6f45; }

  .who { display: flex; align-items: center; gap: 8px; font-size: .77rem; color: #777; }
  .icon-btn {
    display: inline-grid;
    place-items: center;
    width: 34px;
    height: 34px;
    padding: 0;
    border: none;
    border-radius: 10px;
    background: transparent;
    color: #888;
  }
  .icon-btn:hover:not(:disabled) { background: #f1f1f1; color: #1b1b1b; }

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
    padding: 8px 11px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, .18);
    word-break: break-word;
    animation: toast-in 220ms var(--ease-out) both;
  }
  @keyframes toast-in {
    from { opacity: 0; transform: translateY(8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @media (max-width: 640px) {
    .topbar { gap: 8px; padding: 8px 16px; }
    .tabs { order: 3; flex-basis: 100%; }
  }
</style>
