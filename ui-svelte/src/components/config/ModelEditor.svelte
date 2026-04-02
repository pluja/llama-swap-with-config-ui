<script lang="ts">
  import type { RawModelConfig } from "../../lib/configTypes";
  import { upsertModel } from "../../stores/configApi";

  interface Props {
    modelName: string;
    modelConfig: RawModelConfig;
    isNew?: boolean;
    onSave: () => void;
    onClose: () => void;
  }

  let { modelName, modelConfig, isNew = false, onSave, onClose }: Props = $props();

  let saving = $state(false);
  let error = $state("");

  let name = $state(modelName);
  let cmd = $state(modelConfig.cmd ?? "");
  let cmdStop = $state(modelConfig.cmdStop ?? "");
  let proxy = $state(modelConfig.proxy ?? "");
  let checkEndpoint = $state(modelConfig.checkEndpoint ?? "");
  let ttl = $state(modelConfig.ttl ?? 0);
  let displayName = $state(modelConfig.name ?? "");
  let description = $state(modelConfig.description ?? "");
  let unlisted = $state(modelConfig.unlisted ?? false);
  let useModelName = $state(modelConfig.useModelName ?? "");
  let concurrencyLimit = $state(modelConfig.concurrencyLimit ?? 0);
  let envItems = $state<string[]>([...(modelConfig.env ?? [])]);
  let aliasItems = $state<string[]>([...(modelConfig.aliases ?? [])]);

  function addEnvItem() {
    envItems = [...envItems, ""];
  }

  function removeEnvItem(index: number) {
    envItems = envItems.filter((_, i) => i !== index);
  }

  function updateEnvItem(index: number, value: string) {
    envItems = envItems.map((item, i) => (i === index ? value : item));
  }

  function addAlias() {
    aliasItems = [...aliasItems, ""];
  }

  function removeAlias(index: number) {
    aliasItems = aliasItems.filter((_, i) => i !== index);
  }

  function updateAlias(index: number, value: string) {
    aliasItems = aliasItems.map((item, i) => (i === index ? value : item));
  }

  async function handleSave() {
    const modelId = isNew ? name.trim() : modelName;
    if (!modelId) {
      error = "Model ID is required";
      return;
    }
    if (!cmd.trim()) {
      error = "Command is required";
      return;
    }

    saving = true;
    error = "";

    const config: RawModelConfig = { cmd: cmd.trim() };

    if (cmdStop.trim()) config.cmdStop = cmdStop.trim();
    if (proxy.trim()) config.proxy = proxy.trim();
    if (checkEndpoint.trim()) config.checkEndpoint = checkEndpoint.trim();
    if (ttl > 0) config.ttl = ttl;
    if (displayName.trim()) config.name = displayName.trim();
    if (description.trim()) config.description = description.trim();
    if (unlisted) config.unlisted = true;
    if (useModelName.trim()) config.useModelName = useModelName.trim();
    if (concurrencyLimit > 0) config.concurrencyLimit = concurrencyLimit;
    const filteredEnv = envItems.filter((e) => e.trim());
    if (filteredEnv.length > 0) config.env = filteredEnv;
    const filteredAliases = aliasItems.filter((a) => a.trim());
    if (filteredAliases.length > 0) config.aliases = filteredAliases;

    try {
      await upsertModel(modelId, config);
      onSave();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to save model";
    } finally {
      saving = false;
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") onClose();
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 bg-black/50 flex items-start justify-center z-50 overflow-y-auto p-4"
  onclick={handleBackdropClick}
>
  <div class="bg-surface border border-border rounded-lg w-full max-w-2xl my-8">
    <div class="flex items-center justify-between p-4 border-b border-border">
      <h3 class="text-lg font-semibold pb-0">{isNew ? "New Model" : `Edit: ${modelName}`}</h3>
      <button class="btn" onclick={onClose}>✕</button>
    </div>

    <div class="p-4 space-y-4 max-h-[70vh] overflow-y-auto">
      {#if error}
        <div class="bg-error/10 text-error border border-error/20 rounded px-3 py-2 text-sm">{error}</div>
      {/if}

      <div>
        <label class="block text-sm font-medium mb-1" for="model-id">Model ID</label>
        {#if isNew}
          <input
            id="model-id"
            type="text"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
            bind:value={name}
            placeholder="my-model"
          />
        {:else}
          <input
            id="model-id"
            type="text"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full opacity-60"
            value={modelName}
            disabled
          />
        {/if}
      </div>

      <div>
        <label class="block text-sm font-medium mb-1" for="model-cmd">Command *</label>
        <textarea
          id="model-cmd"
          class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full font-mono resize-y"
          rows="3"
          bind:value={cmd}
          placeholder="llama-server --port $&#123;PORT&#125; --model /path/to/model.gguf"
        ></textarea>
      </div>

      <div>
        <label class="block text-sm font-medium mb-1" for="model-cmdstop">Stop Command</label>
        <textarea
          id="model-cmdstop"
          class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full font-mono resize-y"
          rows="2"
          bind:value={cmdStop}
          placeholder="Optional stop command"
        ></textarea>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium mb-1" for="model-proxy">Proxy URL</label>
          <input
            id="model-proxy"
            type="text"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
            bind:value={proxy}
            placeholder="http://localhost:$&#123;PORT&#125;"
          />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1" for="model-check">Check Endpoint</label>
          <input
            id="model-check"
            type="text"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
            bind:value={checkEndpoint}
            placeholder="/health"
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium mb-1" for="model-ttl">TTL (seconds)</label>
          <input
            id="model-ttl"
            type="number"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
            bind:value={ttl}
            min="0"
          />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1" for="model-concurrency">Concurrency Limit</label>
          <input
            id="model-concurrency"
            type="number"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
            bind:value={concurrencyLimit}
            min="0"
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium mb-1" for="model-displayname">Display Name</label>
          <input
            id="model-displayname"
            type="text"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
            bind:value={displayName}
          />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1" for="model-usemodelname">Use Model Name</label>
          <input
            id="model-usemodelname"
            type="text"
            class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
            bind:value={useModelName}
          />
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium mb-1" for="model-description">Description</label>
        <input
          id="model-description"
          type="text"
          class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
          bind:value={description}
        />
      </div>

      <div class="flex items-center gap-2">
        <input id="model-unlisted" type="checkbox" bind:checked={unlisted} />
        <label class="text-sm" for="model-unlisted">Unlisted</label>
      </div>

      <div>
        <div class="flex items-center justify-between mb-1">
          <label class="text-sm font-medium">Environment Variables</label>
          <button class="text-xs text-blue-600 dark:text-blue-400 hover:underline" onclick={addEnvItem}>+ Add</button>
        </div>
        {#each envItems as item, i}
          <div class="flex gap-2 mb-1">
            <input
              type="text"
              class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full font-mono"
              value={item}
              oninput={(e) => updateEnvItem(i, (e.target as HTMLInputElement).value)}
              placeholder="KEY=value"
            />
            <button class="text-red-500 hover:text-red-700 text-sm px-2" onclick={() => removeEnvItem(i)}>✕</button>
          </div>
        {/each}
      </div>

      <div>
        <div class="flex items-center justify-between mb-1">
          <label class="text-sm font-medium">Aliases</label>
          <button class="text-xs text-blue-600 dark:text-blue-400 hover:underline" onclick={addAlias}>+ Add</button>
        </div>
        {#each aliasItems as item, i}
          <div class="flex gap-2 mb-1">
            <input
              type="text"
              class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full"
              value={item}
              oninput={(e) => updateAlias(i, (e.target as HTMLInputElement).value)}
              placeholder="alias-name"
            />
            <button class="text-red-500 hover:text-red-700 text-sm px-2" onclick={() => removeAlias(i)}>✕</button>
          </div>
        {/each}
      </div>
    </div>

    <div class="flex items-center justify-end gap-2 p-4 border-t border-border">
      <button class="btn" onclick={onClose}>Cancel</button>
      <button
        class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 rounded text-sm"
        onclick={handleSave}
        disabled={saving}
      >
        {saving ? "Saving..." : "Save"}
      </button>
    </div>
  </div>
</div>
