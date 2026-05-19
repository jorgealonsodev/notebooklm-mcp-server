package browser

// stealthScript is injected into every page via AddInitScript to neutralize
// bot-detection signals. It covers: navigator.webdriver, chrome.runtime,
// plugins, languages, permissions, iframe detection, and userAgent consistency.
//
// This is functionally equivalent to puppeteer-extra-plugin-stealth and
// Patchright's built-in evasion — but applied through CDP init script
// instead of binary patching.
const stealthScript = `(() => {
  // 1. Delete navigator.webdriver
  const proto = Object.getPrototypeOf(navigator);
  delete proto.webdriver;
  Object.defineProperty(navigator, 'webdriver', {
    get: () => false,
    configurable: true,
  });

  // 2. Mock chrome.runtime if absent
  if (typeof chrome === 'undefined' || !chrome.runtime) {
    window.chrome = {
      runtime: { onConnect: { addListener: () => {} }, onMessage: { addListener: () => {} } },
      loadTimes: () => {},
      csi: () => {},
      app: {},
    };
  }

  // 3. Populate plugins array (headless often reports 0 plugins)
  const originalPlugins = navigator.plugins;
  if (!originalPlugins || originalPlugins.length === 0) {
    const makePlugin = (name, filename, description) => ({
      name, filename, description,
      length: 1,
      item: () => null,
      namedItem: () => null,
    });
    const mimetypes = {
      length: 3,
      item: () => null,
      namedItem: () => null,
    };
    Object.defineProperty(navigator, 'plugins', {
      get: () => ({
        ...Object.create(Plugin.prototype || {}),
        0: makePlugin('Chrome PDF Plugin', 'internal-pdf-viewer', 'Portable Document Format'),
        1: makePlugin('Chrome PDF Viewer', 'mhjfbmdgcfjbbpaeojofohoefgiehjai', ''),
        2: makePlugin('Native Client', 'internal-nacl-plugin', ''),
        length: 3,
        item: (i) => [makePlugin('Chrome PDF Plugin', 'internal-pdf-viewer', 'PDF'),
                       makePlugin('Chrome PDF Viewer', 'mhjfbmdgcfjbbpaeojofohoefgiehjai', ''),
                       makePlugin('Native Client', 'internal-nacl-plugin', '')][i] || null,
        namedItem: () => null,
      }),
    });
  }

  // 4. Override languages to match locale
  Object.defineProperty(navigator, 'languages', {
    get: () => ['en-US', 'en'],
  });

  // 5. Normalize permissions API
  const originalQuery = window.navigator.permissions?.query;
  if (originalQuery) {
    window.navigator.permissions.query = (parameters) => (
      parameters.name === 'notifications'
        ? Promise.resolve({ state: Notification.permission || 'default' })
        : originalQuery(parameters)
    );
  }

  // 6. Fix iframe contentWindow detection
  try {
    const originalAttachShadow = Element.prototype.attachShadow;
    if (originalAttachShadow) {
      Element.prototype.attachShadow = function(init) {
        init = init || {};
        init.mode = 'open';
        return originalAttachShadow.call(this, init);
      };
    }
  } catch (_) {}

  // 7. Override userAgent to remove "HeadlessChrome" if present
  const ua = navigator.userAgent;
  if (/HeadlessChrome/.test(ua)) {
    const cleanUA = ua.replace(/HeadlessChrome/, 'Chrome');
    Object.defineProperty(navigator, 'userAgent', {
      get: () => cleanUA,
    });
  }

  // 8. Remove automation CDP traces
  delete window.__playwright__binding__;
  delete window.__pw_manual__;
  delete window.__PW_inspect__;
})();
`