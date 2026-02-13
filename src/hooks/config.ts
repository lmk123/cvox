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
    Notification: CvoxHookMatcher[];
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
      Notification: [
        {
          matcher: "permission_prompt",
          hooks: [notifyHook],
        },
      ],
      Stop: [
        {
          matcher: "", // empty matcher matches all Stop events
          hooks: [{ ...notifyHook }],
        },
      ],
    },
  };
}
