import * as fs from "fs";
import * as path from "path";
import * as os from "os";

export interface HookEventConfig {
  enabled: boolean;
  message: string;
}

export interface TtsConfig {
  enabled: boolean;
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
      message: "{project} needs permission",
    },
    stop: {
      enabled: true,
      message: "{project} task completed",
    },
  },
  tts: {
    enabled: true,
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

export const LOCALE_MESSAGES: Record<string, { notification: string; stop: string }> = {
  en: { notification: "{project} needs permission", stop: "{project} task completed" },
  zh: { notification: "{project} 需要权限", stop: "{project} 任务完成" },
  ja: { notification: "{project} は権限が必要です", stop: "{project} タスクが完了しました" },
  ko: { notification: "{project} 권한이 필요합니다", stop: "{project} 작업이 완료되었습니다" },
};

export function writeProjectConfig(cwd: string, config: Record<string, unknown>): void {
  const filePath = path.join(cwd, ".cvox.json");
  fs.writeFileSync(filePath, JSON.stringify(config, null, 2) + "\n", "utf-8");
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
