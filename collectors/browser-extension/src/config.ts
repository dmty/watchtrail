export interface Config {
  enabled: boolean;
  coreUrl: string;
  token: string;
  allowlist: string[];
}

export const DEFAULT_CONFIG: Config = {
  enabled: true,
  coreUrl: "http://127.0.0.1:8765",
  token: "",
  allowlist: [],
};

export function withDefaults(partial: Partial<Config>): Config {
  return { ...DEFAULT_CONFIG, ...partial };
}
