export interface RawModelConfig {
  cmd?: string;
  cmdStop?: string;
  proxy?: string;
  aliases?: string[];
  env?: string[];
  checkEndpoint?: string;
  ttl?: number;
  unlisted?: boolean;
  useModelName?: string;
  name?: string;
  description?: string;
  concurrencyLimit?: number;
  filters?: {
    stripParams?: string;
    setParams?: Record<string, unknown>;
    setParamsByID?: Record<string, Record<string, unknown>>;
  };
  macros?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  sendLoadingState?: boolean;
}

export interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: string | null;
  size_bytes: number | null;
}

export interface ConfigState {
  models: Record<string, RawModelConfig>;
  macros: Record<string, unknown>;
}
