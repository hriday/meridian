// stubWails is injected into the page before any app script runs. It replaces
// the Wails bridge with a recursive, permissive mock so every bound Go method
// resolves to a value that renders without throwing: it is an empty iterable, is
// callable, indexes to itself, has length 0, and stringifies to "". That lets
// each tool mount (and any onMount backend call return) without a real backend.
// It also opens every sidebar group and picks a default tool so the smoke test
// can reach them all.
export function stubWails() {
  try {
    localStorage.setItem("meridian.closedGroups", "{}");
    localStorage.setItem("meridian.activeTool", "lookup");
  } catch {
    /* private mode */
  }

  const perm = (): any =>
    new Proxy(function () {}, {
      get(_t, p) {
        if (p === Symbol.iterator) return function* () {};
        if (p === "then") return undefined; // not a thenable — await yields this proxy
        if (p === "length") return 0;
        if (p === Symbol.toPrimitive || p === "toString" || p === "valueOf") return () => "";
        return perm();
      },
      apply: () => perm(),
    });

  // A recursive, callable bridge: any property chain (go.app.App.Method) returns
  // the same bridge, and calling it invokes `call`. So every bound Go method,
  // at any depth, is a function that returns the mock.
  const makeBridge = (call: () => any): any => {
    const b: any = new Proxy(function () {}, {
      get: (_t, p) => (p === "then" ? undefined : b),
      apply: () => call(),
    });
    return b;
  };

  (window as any).go = makeBridge(() => Promise.resolve(perm())); // Go methods are async
  (window as any).runtime = makeBridge(() => perm()); // runtime calls are sync
}
