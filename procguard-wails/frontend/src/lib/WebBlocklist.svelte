<script lang="ts">
  import { onMount } from 'svelte';
  import { writable } from 'svelte/store';
  import { showToast } from './toastStore';

  interface WebBlocklistItem {
    domain: string;
    title: string;
    iconUrl: string;
  }

  let webBlocklistItems = writable<WebBlocklistItem[]>([]);
  let selectedWebsites: string[] = [];

  async function loadWebBlocklist(): Promise<void> {
    try {
      const data = await window.go.main.App.GetWebBlocklist();
      if (data && data.length > 0) {
        webBlocklistItems.set(data);
      } else {
        webBlocklistItems.set([]);
      }
    } catch (error) {
      showToast(`Lỗi tải danh sách chặn web: ${error}`, 'error');
      webBlocklistItems.set([]);
    }
  }

  async function removeWebBlocklist(domain: string): Promise<void> {
    if (confirm(`Bạn có chắc chắn muốn bỏ chặn ${domain} không?`)) {
      try {
        await window.go.main.App.RemoveWebBlocklist(domain);
        showToast(`Đã bỏ chặn ${domain}`, 'success');
        loadWebBlocklist();
      } catch (error) {
        showToast(`Lỗi bỏ chặn ${domain}: ${error}`, 'error');
      }
    }
  }

  async function unblockSelectedWebsites(): Promise<void> {
    if (selectedWebsites.length === 0) {
      showToast('Vui lòng chọn các trang web để bỏ chặn.', 'info');
      return;
    }

    try {
      // Wails allows calling the Go method for each selected domain.
      for (const domain of selectedWebsites) {
        await window.go.main.App.RemoveWebBlocklist(domain);
      }
      showToast(`Đã bỏ chặn: ${selectedWebsites.join(', ')}`, 'success');
    } catch (error) {
      showToast(`Lỗi bỏ chặn trang web: ${error}`, 'error');
      return;
    }

    loadWebBlocklist();
    selectedWebsites = [];
  }

  async function clearWebBlocklist(): Promise<void> {
    if (
      confirm('Bạn có chắc chắn muốn xóa toàn bộ danh sách chặn web không?')
    ) {
      try {
        await window.go.main.App.ClearWebBlocklist();
        showToast('Đã xóa toàn bộ danh sách chặn web.', 'success');
        loadWebBlocklist();
      } catch (error) {
        showToast(`Lỗi xóa danh sách chặn: ${error}`, 'error');
      }
    }
  }

  async function saveWebBlocklist(): Promise<void> {
    try {
      // The Go method returns a byte array which can be used to create a Blob
      const blobBytes = await window.go.main.App.SaveWebBlocklist();
      const blob = new Blob([new Uint8Array(blobBytes)], {
        type: 'application/json',
      });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.style.display = 'none';
      a.href = url;
      a.download = 'procguard_web_blocklist.json';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (error) {
      showToast(`Lỗi lưu danh sách chặn: ${error}`, 'error');
    }
  }

  async function loadWebBlocklistFile(event: Event): Promise<void> {
    const file = (event.target as HTMLInputElement).files?.[0];
    if (!file) {
      return;
    }
    const reader = new FileReader();
    reader.onload = async () => {
      try {
        // We pass the file content as a Uint8Array to the Go method
        await window.go.main.App.LoadWebBlocklist(
          new Uint8Array(reader.result as ArrayBuffer)
        );
        showToast('Đã tải lên và hợp nhất danh sách chặn web.', 'success');
        loadWebBlocklist();
      } catch (error) {
        showToast(`Lỗi tải lên danh sách chặn: ${error}`, 'error');
      }
    };
    reader.readAsArrayBuffer(file);
  }

  onMount(() => {
    loadWebBlocklist();
  });
</script>

<div class="card mt-3">
  <div class="card-body">
    <h5 class="card-title">Các trang web bị chặn</h5>
    <div class="btn-toolbar" role="toolbar">
      <div class="btn-group me-2" role="group">
        <button
          type="button"
          class="btn btn-primary"
          on:click={unblockSelectedWebsites}
        >
          Bỏ chặn mục đã chọn
        </button>
        <button
          type="button"
          class="btn btn-danger"
          on:click={clearWebBlocklist}
        >
          Xóa toàn bộ
        </button>
      </div>
      <div class="btn-group" role="group">
        <button
          type="button"
          class="btn btn-outline-secondary"
          on:click={saveWebBlocklist}
        >
          Lưu danh sách
        </button>
        <button
          type="button"
          class="btn btn-outline-secondary"
          on:click={() => document.getElementById('load-web-input')?.click()}
        >
          Tải lên danh sách
        </button>
      </div>
    </div>
    <input
      type="file"
      id="load-web-input"
      style="display: none"
      on:change={loadWebBlocklistFile}
    />
    <div id="web-blocklist-items" class="list-group mt-3">
      {#if $webBlocklistItems.length > 0}
        {#each $webBlocklistItems as item (item.domain)}
          <div
            class="list-group-item d-flex justify-content-between align-items-center"
          >
            <label class="flex-grow-1 mb-0 d-flex align-items-center">
              <input
                class="form-check-input me-2"
                type="checkbox"
                name="blocked-website"
                value={item.domain}
                bind:group={selectedWebsites}
              />
              {#if item.iconUrl}
                <img
                  src={item.iconUrl}
                  class="me-2"
                  style="width: 24px; height: 24px;"
                  alt="Website Icon"
                />
              {:else}
                <div class="me-2" style="width: 24px; height: 24px;"></div>
              {/if}
              <span class="fw-bold me-2">{item.title || item.domain}</span>
            </label>
            <button
              class="btn btn-sm btn-outline-danger"
              on:click={() => removeWebBlocklist(item.domain)}>&times;</button
            >
          </div>
        {/each}
      {:else}
        <div class="list-group-item">Hiện không có trang web nào bị chặn.</div>
      {/if}
    </div>
  </div>
</div>
