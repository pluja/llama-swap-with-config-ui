<script lang="ts">
  import { onMount } from "svelte";
  import { fetchRawConfig, saveMacros } from "../../stores/configApi";

  interface MacroRow {
    key: string;
    value: string;
  }

  let rows = $state<MacroRow[]>([]);
  let loading = $state(true);
  let saving = $state(false);
  let error = $state("");
  let success = $state("");

  async function loadMacros() {
    loading = true;
    error = "";
    try {
      const yaml = await fetchRawConfig();
      const macros = parseMacrosFromYaml(yaml);
      rows = Object.entries(macros).map(([key, value]) => ({
        key,
        value: String(value),
      }));
    } catch (err) {
      if (err instanceof Error && err.message.includes("501")) {
        error = "Config management not enabled on the server.";
      } else {
        error = err instanceof Error ? err.message : "Failed to load macros";
      }
    } finally {
      loading = false;
    }
  }

  function parseMacrosFromYaml(yaml: string): Record<string, string> {
    const macros: Record<string, string> = {};
    const lines = yaml.split("\n");
    let inMacros = false;

    for (const line of lines) {
      if (/^macros:\s*$/.test(line)) {
        inMacros = true;
        continue;
      }
      if (inMacros) {
        if (/^\S/.test(line) && !/^macros:/.test(line)) break;
        const match = line.match(/^\s+(\S+):\s*(.*)$/);
        if (match) {
          macros[match[1]] = match[2].trim();
        }
      }
    }
    return macros;
  }

  function addRow() {
    rows = [...rows, { key: "", value: "" }];
  }

  function removeRow(index: number) {
    rows = rows.filter((_, i) => i !== index);
  }

  function updateRowKey(index: number, value: string) {
    rows = rows.map((row, i) => (i === index ? { ...row, key: value } : row));
  }

  function updateRowValue(index: number, value: string) {
    rows = rows.map((row, i) => (i === index ? { ...row, value } : row));
  }

  async function handleSave() {
    saving = true;
    error = "";
    success = "";

    const macros: Record<string, unknown> = {};
    for (const row of rows) {
      const key = row.key.trim();
      if (key) {
        macros[key] = row.value;
      }
    }

    try {
      await saveMacros(macros);
      success = "Macros saved.";
      setTimeout(() => (success = ""), 3000);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to save macros";
    } finally {
      saving = false;
    }
  }

  onMount(loadMacros);
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <p class="text-sm text-txtsecondary">Reusable snippets for model commands.</p>
    <div class="flex gap-2">
      <button class="btn" onclick={loadMacros} disabled={loading} title="Refresh">
        ↻
      </button>
      <button
        class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 rounded text-sm"
        onclick={handleSave}
        disabled={saving}
      >
        {saving ? "Saving..." : "Save All"}
      </button>
    </div>
  </div>

  {#if error}
    <div class="bg-error/10 text-error border border-error/20 rounded px-3 py-2 text-sm">{error}</div>
  {/if}

  {#if success}
    <div class="bg-success/10 text-success border border-success/20 rounded px-3 py-2 text-sm">{success}</div>
  {/if}

  {#if loading}
    <p class="text-sm text-txtsecondary">Loading macros...</p>
  {:else}
    {#if rows.length === 0}
      <p class="text-sm text-txtsecondary">No macros defined.</p>
    {:else}
      <div class="space-y-2">
        {#each rows as row, i}
          <div class="flex gap-2 items-start">
            <input
              type="text"
              class="bg-surface border border-border rounded px-3 py-1.5 text-sm font-mono w-48 shrink-0"
              value={row.key}
              oninput={(e) => updateRowKey(i, (e.target as HTMLInputElement).value)}
              placeholder="MACRO_NAME"
            />
            <textarea
              class="bg-surface border border-border rounded px-3 py-1.5 text-sm w-full font-mono resize-y"
              rows="1"
              value={row.value}
              oninput={(e) => updateRowValue(i, (e.target as HTMLTextAreaElement).value)}
              placeholder="value"
            ></textarea>
            <button
              class="text-red-500 hover:text-red-700 text-sm px-2 py-1.5 shrink-0"
              onclick={() => removeRow(i)}
            >
              ✕
            </button>
          </div>
        {/each}
      </div>
    {/if}

    <button
      class="border border-border hover:bg-gray-100 dark:hover:bg-gray-700 px-3 py-1.5 rounded text-sm"
      onclick={addRow}
    >
      + Add Macro
    </button>
  {/if}
</div>
