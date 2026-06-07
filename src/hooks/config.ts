export interface CvoxHook {
  type: "command";
  command: string;
  async: boolean;
}

export interface CvoxHookMatcher {
  matcher: string;
  hooks: CvoxHook[];
}

export interface CvoxHooksConfig {
  hooks: {
    PermissionRequest: CvoxHookMatcher[];
    Stop: CvoxHookMatcher[];
  };
}

export function generateHooksConfig(): CvoxHooksConfig {
  const notifyHook: CvoxHook = {
    type: "command",
    command: "cvox notify",
    async: true,
  };

  return {
    hooks: {
      // Permission prompts fire PermissionRequest on both the Claude Code CLI
      // and the Claude Desktop app, so this single hook covers both. (The CLI
      // also fires a legacy Notification hook that Desktop does not; cvox no
      // longer uses it, since PermissionRequest alone is enough.)
      PermissionRequest: [
        {
          matcher: "", // empty matcher matches all permission requests
          hooks: [notifyHook],
        },
      ],
      Stop: [
        {
          matcher: "", // empty matcher matches all Stop events
          hooks: [notifyHook],
        },
      ],
    },
  };
}
