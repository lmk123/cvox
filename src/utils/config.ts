import * as fs from "fs";
import * as path from "path";
import * as os from "os";

export interface HookEventConfig {
  enabled: boolean;
  message: string;
}

export interface TtsConfig {
  enabled: boolean;
  engine: "auto" | "say" | "espeak" | "sapi";
  voice: string;
  rate: number;
}

export interface DesktopConfig {
  enabled: boolean;
}

export interface CvoxConfig {
  project: string;
  hooks: {
    notification: HookEventConfig;
    stop: HookEventConfig;
  };
  tts: TtsConfig;
  desktop: DesktopConfig;
}

export const DEFAULT_CONFIG: CvoxConfig = {
  project: "",
  hooks: {
    notification: {
      enabled: true,
      message: "{project} 需要确认权限",
    },
    stop: {
      enabled: true,
      message: "{project} 任务已完成",
    },
  },
  tts: {
    enabled: true,
    engine: "auto",
    voice: "",
    rate: 0,
  },
  desktop: {
    enabled: false,
  },
};

function deepMerge(target: any, source: any): any {
  const result = { ...target };
  for (const key of Object.keys(source)) {
    if (
      source[key] &&
      typeof source[key] === "object" &&
      !Array.isArray(source[key]) &&
      target[key] &&
      typeof target[key] === "object"
    ) {
      result[key] = deepMerge(target[key], source[key]);
    } else {
      result[key] = source[key];
    }
  }
  return result;
}

function tryReadJson(filePath: string): Partial<CvoxConfig> | null {
  try {
    const content = fs.readFileSync(filePath, "utf-8");
    return JSON.parse(content);
  } catch {
    return null;
  }
}

export function loadConfig(cwd: string): CvoxConfig {
  const projectConfig = tryReadJson(path.join(cwd, ".cvox.json"));
  const globalConfig = tryReadJson(
    path.join(os.homedir(), ".cvox.json")
  );

  let config: CvoxConfig = { ...DEFAULT_CONFIG };
  if (globalConfig) {
    config = deepMerge(config, globalConfig);
  }
  if (projectConfig) {
    config = deepMerge(config, projectConfig);
  }

  if (!config.project) {
    config.project = path.basename(cwd);
  }

  return config;
}
