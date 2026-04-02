<script lang="ts">
  import { persistentStore } from "../stores/persistent";
  import { reloadConfig } from "../stores/configApi";
  import ModelList from "../components/config/ModelList.svelte";
  import MacroEditor from "../components/config/MacroEditor.svelte";
  import FileBrowser from "../components/config/FileBrowser.svelte";

  type TabId = "models" | "macros" | "files";

  const activeTabStore = persistentStore<TabId>("config-active-tab", "models");
  let reloading = $state(false);
  let toast = $state<{ message: string; type: "success" | "error" } | null>(null);

  function setTab(tab: TabId) {
    activeTabStore.set(tab);
  }

  async function handleReload() {
    reloading = true;
    try {
      await reloadConfig();
      showToast("Config reloaded.", "success");
    } catch (err) {
      showToast(err instanceof Error ? err.message : "Reload failed", "error");
    } finally {
      reloading = false;
    }
  }

  function showToast(message: string, type: "success" | "error") {
    toast = { message, type };
    setTimeout(() => (toast = null), 3000);
  }

  const tabs: { id: TabId; label: string }[] = [
    { id: "models", label: "Models" },
    { id: "macros", label: "Macros" },
    { id: "files", label: "Files" },
  ];
</script>

<div class="card h-full flex flex-col">
  <div class="shrink-0">
    <div class="flex items-center justify-between mb-4">
      <div class="flex border-b border-border">
        {#each tabs as tab}
          <button
            class="px-4 py-2 text-sm {$activeTabStore === tab.id
              ? 'border-b-2 border-blue-500 text-blue-600 dark:text-blue-400 font-semibold'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'}"
            onclick={() => setTab(tab.id)}
          >
            {tab.label}
          </button>
        {/each}
      </div>
      <button
        class="btn text-sm"
        onclick={handleReload}
        disabled={reloading}
      >
        {reloading ? "Reloading..." : "Reload Config"}
      </button>
    </div>

    {#if toast}
      <div class="mb-4 px-3 py-2 rounded text-sm {toast.type === 'success'
        ? 'bg-success/10 text-success border border-success/20'
        : 'bg-error/10 text-error border border-error/20'}">
        {toast.message}
      </div>
    {/if}
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if $activeTabStore === "models"}
      <ModelList />
    {:else if $activeTabStore === "macros"}
      <MacroEditor />
    {:else if $activeTabStore === "files"}
      <FileBrowser />
    {/if}
  </div>
</div>
