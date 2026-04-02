<script lang="ts">
  import { onMount } from "svelte";
  import type { RawModelConfig } from "../../lib/configTypes";
  import { fetchConfigModels, deleteModel, upsertModel, renameModel } from "../../stores/configApi";
  import ModelEditor from "./ModelEditor.svelte";

  let models = $state<Record<string, RawModelConfig>>({});
  let loading = $state(true);
  let error = $state("");
  let searchQuery = $state("");
  let expandedModel = $state<string | null>(null);

  let editingModel = $state<{ name: string; config: RawModelConfig; isNew: boolean } | null>(null);
  let confirmDelete = $state<string | null>(null);
  let renamingModel = $state<string | null>(null);
  let renameValue = $state("");

  let filteredModelEntries = $derived(
    Object.entries(models)
      .filter(([name]) => name.toLowerCase().includes(searchQuery.toLowerCase()))
      .sort(([a], [b]) => a.localeCompare(b, undefined, { numeric: true }))
  );

  async function loadModels() {
    loading = true;
    error = "";
    try {
      models = await fetchConfigModels();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load models";
    } finally {
      loading = false;
    }
  }

  async function handleDelete(name: string) {
    try {
      await deleteModel(name);
      confirmDelete = null;
      await loadModels();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to delete model";
    }
  }

  async function handleDuplicate(name: string, config: RawModelConfig) {
    const newName = `${name}-copy`;
    try {
      await upsertModel(newName, { ...config });
      await loadModels();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to duplicate model";
    }
  }

  async function handleRename(oldName: string) {
    const newName = renameValue.trim();
    if (!newName || newName === oldName) {
      renamingModel = null;
      return;
    }
    try {
      await renameModel(oldName, newName);
      renamingModel = null;
      renameValue = "";
      await loadModels();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to rename model";
    }
  }

  function startRename(name: string) {
    renamingModel = name;
    renameValue = name;
  }

  function toggleExpanded(name: string) {
    expandedModel = expandedModel === name ? null : name;
  }

  function truncateCmd(cmd: string | undefined, maxLen: number = 80): string {
    if (!cmd) return "";
    return cmd.length > maxLen ? cmd.substring(0, maxLen) + "…" : cmd;
  }

  onMount(loadModels);
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <input
      type="text"
      class="bg-surface border border-border rounded px-3 py-1.5 text-sm flex-1"
      placeholder="Filter models..."
      bind:value={searchQuery}
    />
    <button
      class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 rounded text-sm whitespace-nowrap"
      onclick={() => (editingModel = { name: "", config: {}, isNew: true })}
    >
      + New Model
    </button>
    <button class="btn" onclick={loadModels} disabled={loading} title="Refresh">
      ↻
    </button>
  </div>

  {#if error}
    <div class="bg-error/10 text-error border border-error/20 rounded px-3 py-2 text-sm">{error}</div>
  {/if}

  {#if loading}
    <p class="text-sm text-txtsecondary">Loading models...</p>
  {:else if filteredModelEntries.length === 0}
    <p class="text-sm text-txtsecondary">
      {searchQuery ? "No models match your filter." : "No models configured."}
    </p>
  {:else}
    <div class="space-y-2">
      {#each filteredModelEntries as [name, config] (name)}
        <div class="bg-surface border border-border rounded-lg overflow-hidden">
          <button
            class="w-full text-left px-4 py-3 flex items-center justify-between hover:bg-secondary-hover"
            onclick={() => toggleExpanded(name)}
          >
            <div class="min-w-0 flex-1">
              <span class="font-semibold text-sm">{name}</span>
              {#if config.cmd}
                <p class="text-xs text-txtsecondary font-mono truncate mt-0.5">{truncateCmd(config.cmd)}</p>
              {/if}
            </div>
            <span class="text-txtsecondary ml-2 flex-shrink-0">
              {expandedModel === name ? "▾" : "▸"}
            </span>
          </button>

          {#if expandedModel === name}
            <div class="border-t border-border px-4 py-3 space-y-3">
              <div class="flex flex-wrap gap-2 mb-3">
                <button
                  class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 rounded text-sm"
                  onclick={() => (editingModel = { name, config, isNew: false })}
                >
                  Edit
                </button>
                <button
                  class="border border-border hover:bg-gray-100 dark:hover:bg-gray-700 px-3 py-1.5 rounded text-sm"
                  onclick={() => handleDuplicate(name, config)}
                >
                  Duplicate
                </button>
                <button
                  class="border border-border hover:bg-gray-100 dark:hover:bg-gray-700 px-3 py-1.5 rounded text-sm"
                  onclick={() => startRename(name)}
                >
                  Rename
                </button>
                {#if confirmDelete === name}
                  <span class="flex items-center gap-1">
                    <span class="text-sm text-error">Delete?</span>
                    <button
                      class="bg-red-600 hover:bg-red-700 text-white px-3 py-1.5 rounded text-sm"
                      onclick={() => handleDelete(name)}
                    >
                      Yes
                    </button>
                    <button class="btn text-sm" onclick={() => (confirmDelete = null)}>No</button>
                  </span>
                {:else}
                  <button
                    class="bg-red-600 hover:bg-red-700 text-white px-3 py-1.5 rounded text-sm"
                    onclick={() => (confirmDelete = name)}
                  >
                    Delete
                  </button>
                {/if}
              </div>

              {#if renamingModel === name}
                <div class="flex gap-2 items-center mb-3">
                  <input
                    type="text"
                    class="bg-surface border border-border rounded px-3 py-1.5 text-sm flex-1"
                    bind:value={renameValue}
                    onkeydown={(e) => { if (e.key === "Enter") handleRename(name); if (e.key === "Escape") renamingModel = null; }}
                  />
                  <button class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 rounded text-sm" onclick={() => handleRename(name)}>
                    Save
                  </button>
                  <button class="btn text-sm" onclick={() => (renamingModel = null)}>Cancel</button>
                </div>
              {/if}

              <table class="w-full text-sm">
                <tbody>
                  {#if config.cmd}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary align-top w-36">cmd</td>
                      <td class="py-1.5 font-mono text-xs break-all whitespace-pre-wrap">{config.cmd}</td>
                    </tr>
                  {/if}
                  {#if config.cmdStop}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary align-top">cmdStop</td>
                      <td class="py-1.5 font-mono text-xs break-all">{config.cmdStop}</td>
                    </tr>
                  {/if}
                  {#if config.proxy}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">proxy</td>
                      <td class="py-1.5 text-xs">{config.proxy}</td>
                    </tr>
                  {/if}
                  {#if config.checkEndpoint}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">checkEndpoint</td>
                      <td class="py-1.5 text-xs">{config.checkEndpoint}</td>
                    </tr>
                  {/if}
                  {#if config.ttl}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">ttl</td>
                      <td class="py-1.5 text-xs">{config.ttl}s</td>
                    </tr>
                  {/if}
                  {#if config.name}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">name</td>
                      <td class="py-1.5 text-xs">{config.name}</td>
                    </tr>
                  {/if}
                  {#if config.description}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">description</td>
                      <td class="py-1.5 text-xs">{config.description}</td>
                    </tr>
                  {/if}
                  {#if config.unlisted}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">unlisted</td>
                      <td class="py-1.5 text-xs">true</td>
                    </tr>
                  {/if}
                  {#if config.useModelName}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">useModelName</td>
                      <td class="py-1.5 text-xs">{config.useModelName}</td>
                    </tr>
                  {/if}
                  {#if config.concurrencyLimit}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary">concurrencyLimit</td>
                      <td class="py-1.5 text-xs">{config.concurrencyLimit}</td>
                    </tr>
                  {/if}
                  {#if config.aliases && config.aliases.length > 0}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary align-top">aliases</td>
                      <td class="py-1.5 text-xs">{config.aliases.join(", ")}</td>
                    </tr>
                  {/if}
                  {#if config.env && config.env.length > 0}
                    <tr class="border-b border-border">
                      <td class="py-1.5 pr-4 font-medium text-txtsecondary align-top">env</td>
                      <td class="py-1.5 font-mono text-xs">{config.env.join("\n")}</td>
                    </tr>
                  {/if}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if editingModel}
  <ModelEditor
    modelName={editingModel.name}
    modelConfig={editingModel.config}
    isNew={editingModel.isNew}
    onSave={() => { editingModel = null; loadModels(); }}
    onClose={() => (editingModel = null)}
  />
{/if}
