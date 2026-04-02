<script lang="ts">
  import { onMount } from "svelte";
  import type { FileEntry } from "../../lib/configTypes";
  import { fetchFiles, deleteFile } from "../../stores/configApi";

  let entries = $state<FileEntry[]>([]);
  let currentPath = $state("");
  let loading = $state(true);
  let error = $state("");
  let notConfigured = $state(false);
  let confirmDeletePath = $state<string | null>(null);

  let breadcrumbs = $derived(buildBreadcrumbs(currentPath));

  function buildBreadcrumbs(path: string): { label: string; path: string }[] {
    if (!path) return [{ label: "root", path: "" }];
    const parts = path.split("/").filter(Boolean);
    const crumbs = [{ label: "root", path: "" }];
    let accumulated = "";
    for (const part of parts) {
      accumulated = accumulated ? `${accumulated}/${part}` : part;
      crumbs.push({ label: part, path: accumulated });
    }
    return crumbs;
  }

  async function navigateTo(path: string) {
    loading = true;
    error = "";
    try {
      const result = await fetchFiles(path || undefined);
      entries = result.entries || [];
      currentPath = result.current || path;
    } catch (err) {
      if (err instanceof Error && err.message.includes("not configured")) {
        notConfigured = true;
      } else {
        error = err instanceof Error ? err.message : "Failed to load files";
      }
    } finally {
      loading = false;
    }
  }

  async function handleDelete(path: string) {
    try {
      await deleteFile(path);
      confirmDeletePath = null;
      await navigateTo(currentPath);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to delete";
    }
  }

  function formatSize(entry: FileEntry): string {
    if (entry.is_dir) return "—";
    if (entry.size) return entry.size;
    if (entry.size_bytes == null) return "—";
    const bytes = entry.size_bytes;
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  }

  onMount(() => navigateTo(""));
</script>

<div class="space-y-4">
  {#if notConfigured}
    <div class="bg-surface border border-border rounded-lg p-8 text-center">
      <p class="text-txtsecondary">
        File browsing is not configured. Start llama-swap with the <code class="font-mono bg-secondary px-1.5 py-0.5 rounded text-sm">-models-dir</code> flag to enable.
      </p>
    </div>
  {:else}
    <div class="flex items-center gap-2 text-sm">
      {#each breadcrumbs as crumb, i}
        {#if i > 0}
          <span class="text-txtsecondary">/</span>
        {/if}
        {#if i === breadcrumbs.length - 1}
          <span class="font-medium">{crumb.label}</span>
        {:else}
          <button
            class="text-blue-600 dark:text-blue-400 hover:underline"
            onclick={() => navigateTo(crumb.path)}
          >
            {crumb.label}
          </button>
        {/if}
      {/each}
      <button class="btn ml-auto" onclick={() => navigateTo(currentPath)} disabled={loading} title="Refresh">
        ↻
      </button>
    </div>

    {#if error}
      <div class="bg-error/10 text-error border border-error/20 rounded px-3 py-2 text-sm">{error}</div>
    {/if}

    {#if loading}
      <p class="text-sm text-txtsecondary">Loading files...</p>
    {:else if entries.length === 0}
      <p class="text-sm text-txtsecondary">This directory is empty.</p>
    {:else}
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left border-b border-border">
            <th class="py-2 pr-4 font-medium">Name</th>
            <th class="py-2 pr-4 font-medium w-28 text-right">Size</th>
            <th class="py-2 font-medium w-20"></th>
          </tr>
        </thead>
        <tbody>
          {#each entries as entry (entry.path)}
            <tr class="border-b border-border hover:bg-secondary-hover">
              <td class="py-2 pr-4">
                {#if entry.is_dir}
                  <button
                    class="text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
                    onclick={() => navigateTo(entry.path)}
                  >
                    <span>📁</span>
                    {entry.name}/
                  </button>
                {:else}
                  <span class="flex items-center gap-1">
                    <span>📄</span>
                    {entry.name}
                  </span>
                {/if}
              </td>
              <td class="py-2 pr-4 text-right text-txtsecondary tabular-nums">
                {formatSize(entry)}
              </td>
              <td class="py-2 text-right">
                {#if confirmDeletePath === entry.path}
                  <span class="flex items-center gap-1 justify-end">
                    <button
                      class="bg-red-600 hover:bg-red-700 text-white px-2 py-0.5 rounded text-xs"
                      onclick={() => handleDelete(entry.path)}
                    >
                      Yes
                    </button>
                    <button class="btn btn--sm" onclick={() => (confirmDeletePath = null)}>No</button>
                  </span>
                {:else}
                  <button
                    class="text-red-500 hover:text-red-700 text-xs"
                    onclick={() => (confirmDeletePath = entry.path)}
                  >
                    Delete
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  {/if}
</div>
