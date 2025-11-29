<script lang="ts">
  import Router, { wrap } from 'svelte-spa-router/wrap';
  import Welcome from './lib/Welcome.svelte';
  import AppManagement from './lib/AppManagement.svelte';
  import WebManagement from './lib/WebManagement.svelte';
  import Settings from './lib/Settings.svelte';
  import Login from './lib/Login.svelte';
  import { onMount } from 'svelte';
  import { isAuthenticated } from './lib/authStore';
  import Toast from './lib/Toast.svelte';
  import {
    isUninstallModalOpen,
    uninstallPassword,
    uninstallError,
    handleUninstallSubmit,
  } from './lib/modalStore';

  const routes = {
    // Exact path
    '/': wrap({ asyncComponent: () => import('./lib/Welcome.svelte') }),
    '/apps': wrap({ asyncComponent: () => import('./lib/AppManagement.svelte') }),
    '/web': wrap({ asyncComponent: () => import('./lib/WebManagement.svelte') }),
    '/settings': wrap({ asyncComponent: () => import('./lib/Settings.svelte') }),
    '/login': wrap({ asyncComponent: () => import('./lib/Login.svelte') }),
    // Catch-all for 404
    '*': wrap({ asyncComponent: () => import('./lib/Welcome.svelte') }), // Default to Welcome
  };

  import { checkExtension } from './lib/extensionStore';

  let uninstallPasswordValue = '';
  $: uninstallPassword.set(uninstallPasswordValue);

  async function handleStop() {
    if (confirm('Bạn có chắc chắn muốn dừng ProcGuard không?')) {
      try {
        await window.go.main.App.Stop();
        alert('ProcGuard đã được dừng.');
      } catch (error) {
        console.error('Lỗi khi dừng ProcGuard:', error);
        alert('Đã có lỗi xảy ra khi cố gắng dừng ProcGuard.');
      }
    }
  }

  async function handleLogout() {
    await window.go.main.App.Logout();
    window.location.hash = '#/login'; // Use hash navigation
  }

  onMount(async () => {
    checkExtension();
    const authenticated = await window.go.main.App.GetIsAuthenticated();
    isAuthenticated.set(authenticated);

    if (!authenticated && window.location.hash !== '#/login') {
      window.location.hash = '#/login'; // Use hash navigation
    }
  });
</script>

{#if isAuthenticated}
  <nav class="navbar navbar-expand-lg navbar-light">
    <div class="container-fluid">
      <a
        class="navbar-brand"
        href="#/"
        >ProcGuard</a
      >
      <button
        class="navbar-toggler"
        type="button"
        data-bs-toggle="collapse"
        data-bs-target="#navbarNav"
        aria-controls="navbarNav"
        aria-expanded="false"
        aria-label="Toggle navigation"
      >
        <span class="navbar-toggler-icon"></span>
      </button>
      <div class="collapse navbar-collapse" id="navbarNav">
        <ul class="navbar-nav me-auto">
          <li class="nav-item">
            <a
              class="nav-link"
              href="#/"
              >Trang chủ</a
            >
          </li>
          <li class="nav-item">
            <a
              class="nav-link"
              href="#/settings"
              on:click|preventDefault={() => (window.location.hash = '#/settings')}
              >Cài đặt</a
            >
          </li>
        </ul>
        <ul class="navbar-nav">
          <li class="nav-item">
            <button class="nav-link btn" on:click={handleStop}
              >Dừng ProcGuard</button
            >
          </li>
          <li class="nav-item">
            <button class="nav-link btn" on:click={handleLogout}>Đăng xuất</button>
          </li>
        </ul>
      </div>
    </div>
  </nav>
{/if}

<main class="container mt-4">
  <Router {routes} />
</main>

<Toast />

<!-- Uninstall Modal -->
{#if isUninstallModalOpen}
  <div class="modal-backdrop fade show"></div>
  <div
    class="modal fade show"
    style="display: block;"
    id="uninstall-modal"
    tabindex="-1"
    aria-labelledby="uninstallModalLabel"
    aria-hidden="false"
  >
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title" id="uninstallModalLabel">
            Xác nhận gỡ cài đặt
          </h5>
          <button
            type="button"
            class="btn-close"
            on:click={() => isUninstallModalOpen.set(false)}
            aria-label="Close"
          ></button>
        </div>
        <div class="modal-body">
          <p>Vui lòng nhập mật khẩu của bạn để tiếp tục.</p>
          <form
            on:submit|preventDefault={() => {
              handleUninstallSubmit();
            }}
          >
            <div class="mb-3">
              <input
                type="password"
                class="form-control"
                id="uninstall-password"
                placeholder="Mật khẩu"
                required
                bind:value={uninstallPasswordValue}
              />
            </div>
            {#if uninstallError}
              <p class="text-danger" style="display: block">
                {uninstallError}
              </p>
            {/if}
            <button type="submit" class="btn btn-danger w-100">
              Xác nhận
            </button>
          </form>
        </div>
      </div>
    </div>
  </div>
{/if}
