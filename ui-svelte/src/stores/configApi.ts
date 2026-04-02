import type { RawModelConfig, FileEntry } from "../lib/configTypes";

export async function fetchRawConfig(): Promise<string> {
  const response = await fetch("/api/config");
  if (!response.ok) throw new Error(`Failed to fetch config: ${response.status}`);
  return response.text();
}

export async function saveRawConfig(yaml: string): Promise<void> {
  const response = await fetch("/api/config", {
    method: "PUT",
    body: yaml,
  });
  if (!response.ok) throw new Error(`Failed to save config: ${response.status}`);
}

export async function validateConfig(yaml: string): Promise<{ valid: boolean; error?: string }> {
  const response = await fetch("/api/config/validate", {
    method: "POST",
    body: yaml,
  });
  if (!response.ok) throw new Error(`Failed to validate config: ${response.status}`);
  return response.json();
}

export async function reloadConfig(): Promise<void> {
  const response = await fetch("/api/config/reload", {
    method: "POST",
  });
  if (!response.ok) throw new Error(`Failed to reload config: ${response.status}`);
}

export async function fetchConfigModels(): Promise<Record<string, RawModelConfig>> {
  const response = await fetch("/api/config/models");
  if (!response.ok) throw new Error(`Failed to fetch models: ${response.status}`);
  const data = await response.json();
  return data.models || {};
}

export async function upsertModel(name: string, config: RawModelConfig): Promise<void> {
  const response = await fetch(`/api/config/models/${encodeURIComponent(name)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
  if (!response.ok) throw new Error(`Failed to save model: ${response.status}`);
}

export async function deleteModel(name: string): Promise<void> {
  const response = await fetch(`/api/config/models/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
  if (!response.ok) throw new Error(`Failed to delete model: ${response.status}`);
}

export async function renameModel(oldName: string, newName: string): Promise<void> {
  const response = await fetch("/api/config/models/rename", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ old: oldName, new: newName }),
  });
  if (!response.ok) throw new Error(`Failed to rename model: ${response.status}`);
}

export async function saveMacros(macros: Record<string, unknown>): Promise<void> {
  const response = await fetch("/api/config/macros", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(macros),
  });
  if (!response.ok) throw new Error(`Failed to save macros: ${response.status}`);
}

export async function fetchFiles(path?: string): Promise<{ entries: FileEntry[]; current: string }> {
  const params = path ? `?path=${encodeURIComponent(path)}` : "";
  const response = await fetch(`/api/config/files${params}`);
  if (!response.ok) {
    if (response.status === 501) {
      throw new Error("File browsing not configured");
    }
    throw new Error(`Failed to fetch files: ${response.status}`);
  }
  return response.json();
}

export async function deleteFile(path: string): Promise<void> {
  const response = await fetch("/api/config/files", {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
  if (!response.ok) throw new Error(`Failed to delete file: ${response.status}`);
}
