(() => {
  const I18N = {
    zh: {
      "meta.title": "DevToolbox — 项目环境与 Git 身份工具箱",
      "meta.description": "DevToolbox：多技术栈项目的开发环境与 Git 身份管理工具箱。以项目为入口，一眼可见、一键确认。",
      "meta.og": "项目运行时与 Git 身份，收进同一个钥匙盒。",
      "nav.capabilities": "能力",
      "nav.how": "怎么用",
      "hero.eyebrow": "铜钥匙 · 项目入口",
      "hero.lede": "把「这个项目用哪套运行时、哪个 Git 身份」收进同一个钥匙盒——可见、可确认、可撤销。",
      "hero.download": "下载 Release",
      "hero.source": "查看源码",
      "chip.git": "Git 身份全量覆盖",
      "chip.path": "不改父终端 PATH",
      "cap.title": "三件事，一套工具",
      "cap.lede": "CLI、桌面客户端与托盘共用同一 Go 应用服务；前端只展示与发出结构化请求。",
      "cap.project.title": "项目上下文",
      "cap.project.body": "登记与扫描本机项目，识别 Git 工作树与构建模块，提取版本要求。只读，不跑项目脚本。",
      "cap.runtime.title": "用户级运行时",
      "cap.runtime.body": "Node / Java / Maven / Gradle / Android SDK / Go / Python：发现、检查、安全安装，预览后显式确认。",
      "cap.git.title": "Git 身份",
      "cap.git.body": "多档案全量覆盖、目录绑定、凭据不落盘；dry-run 与真跑同一算法，批量出结果。",
      "how.title": "显式动作，可预期副作用",
      "how.lede": "安装与切换属于明确用户动作；环境优先作用于启动的会话与子进程。",
      "how.s1.title": "选中项目",
      "how.s1.body": "从登记列表进入，一眼看到运行时要求与当前 Git 身份。",
      "how.s2.title": "预览计划",
      "how.s2.body": "下载、校验、目标路径与体积先列出来，再确认执行。",
      "how.s3.title": "会话生效",
      "how.s3.body": "作用于新开的会话与子进程；不偷偷改写全局默认。",
      "dl.title": "Windows 与 Linux 预构建",
      "dl.lede": "从 GitHub Releases 下载安装包，开箱即用。",
      "dl.get": "获取最新版本",
      "dl.readme": "阅读 README",
      "foot.copy": '开源 · MIT · <a href="https://github.com/xyztony999/dev-toolbox">xyztony999/dev-toolbox</a>',
      "ctrl.lang": "EN",
      "ctrl.langLabel": "切换到英文",
      "ctrl.themeLabel": "切换主题",
    },
    en: {
      "meta.title": "DevToolbox — Project runtimes & Git identity",
      "meta.description": "DevToolbox manages multi-stack project runtimes and Git identities from a project-first entry point—visible, confirmable, reversible.",
      "meta.og": "Project runtimes and Git identity, kept in one toolbox.",
      "nav.capabilities": "Capabilities",
      "nav.how": "How it works",
      "hero.eyebrow": "Copper key · Project-first",
      "hero.lede": "Put “which runtime and which Git identity this project uses” in one toolbox—visible, confirmable, reversible.",
      "hero.download": "Download release",
      "hero.source": "View source",
      "chip.git": "Full Git identity overwrite",
      "chip.path": "Leaves parent PATH alone",
      "cap.title": "Three jobs, one tool",
      "cap.lede": "CLI, desktop, and tray share one Go application service. The UI only presents state and sends structured requests.",
      "cap.project.title": "Project context",
      "cap.project.body": "Register and scan local projects, detect Git worktrees and build modules, extract version requirements. Read-only—never runs project scripts.",
      "cap.runtime.title": "User-level runtimes",
      "cap.runtime.body": "Node / Java / Maven / Gradle / Android SDK / Go / Python: discover, inspect, and install safely after an explicit preview-and-confirm step.",
      "cap.git.title": "Git identity",
      "cap.git.body": "Multi-profile full overwrite, directory binding, credentials never on disk; dry-run and apply share the same algorithm with per-repo results.",
      "how.title": "Explicit actions, predictable side effects",
      "how.lede": "Install and switch are deliberate user actions. Environment changes apply to launched sessions and child processes first.",
      "how.s1.title": "Pick a project",
      "how.s1.body": "Open from the registry and see runtime requirements and the current Git identity at a glance.",
      "how.s2.title": "Preview the plan",
      "how.s2.body": "Downloads, checksums, target paths, and size are listed before you confirm.",
      "how.s3.title": "Session takes effect",
      "how.s3.body": "Applies to newly opened sessions and child processes—never silently rewrites global defaults.",
      "dl.title": "Windows & Linux builds",
      "dl.lede": "Grab installers from GitHub Releases and start immediately.",
      "dl.get": "Get the latest release",
      "dl.readme": "Read the README",
      "foot.copy": 'Open source · MIT · <a href="https://github.com/xyztony999/dev-toolbox">xyztony999/dev-toolbox</a>',
      "ctrl.lang": "中文",
      "ctrl.langLabel": "Switch to Chinese",
      "ctrl.themeLabel": "Toggle theme",
    },
  };

  const THEME_KEY = "dt.theme";
  const LANG_KEY = "dt.lang";

  function resolveLang() {
    const saved = localStorage.getItem(LANG_KEY);
    if (saved === "zh" || saved === "en") return saved;
    return (navigator.language || "").toLowerCase().startsWith("zh") ? "zh" : "en";
  }

  function resolveTheme() {
    const saved = localStorage.getItem(THEME_KEY);
    return saved === "light" || saved === "dark" || saved === "system" ? saved : "system";
  }

  function paintTheme(pref) {
    if (pref === "light" || pref === "dark") {
      document.documentElement.dataset.theme = pref;
    } else {
      document.documentElement.removeAttribute("data-theme");
    }
  }

  function applyLang(lang) {
    const dict = I18N[lang] || I18N.en;
    document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
    document.documentElement.dataset.lang = lang;

    document.querySelectorAll("[data-i18n]").forEach((el) => {
      const key = el.getAttribute("data-i18n");
      if (key && dict[key] != null) el.textContent = dict[key];
    });
    document.querySelectorAll("[data-i18n-html]").forEach((el) => {
      const key = el.getAttribute("data-i18n-html");
      if (key && dict[key] != null) el.innerHTML = dict[key];
    });

    const title = dict["meta.title"];
    if (title) document.title = title;
    const desc = document.querySelector('meta[name="description"]');
    if (desc && dict["meta.description"]) desc.setAttribute("content", dict["meta.description"]);
    const og = document.querySelector('meta[property="og:description"]');
    if (og && dict["meta.og"]) og.setAttribute("content", dict["meta.og"]);

    const langBtn = document.getElementById("lang-toggle");
    if (langBtn) {
      langBtn.textContent = dict["ctrl.lang"];
      langBtn.setAttribute("aria-label", dict["ctrl.langLabel"]);
    }
    const themeBtn = document.getElementById("theme-toggle");
    if (themeBtn) themeBtn.setAttribute("aria-label", dict["ctrl.themeLabel"]);
  }

  function cycleTheme(current) {
    if (current === "system") return "light";
    if (current === "light") return "dark";
    return "system";
  }

  const lang = resolveLang();
  const theme = resolveTheme();
  paintTheme(theme);
  applyLang(lang);
  localStorage.setItem(LANG_KEY, lang);
  localStorage.setItem(THEME_KEY, theme);

  document.getElementById("lang-toggle")?.addEventListener("click", () => {
    const next = (document.documentElement.dataset.lang === "zh" ? "en" : "zh");
    localStorage.setItem(LANG_KEY, next);
    applyLang(next);
  });

  document.getElementById("theme-toggle")?.addEventListener("click", () => {
    const next = cycleTheme(resolveTheme());
    localStorage.setItem(THEME_KEY, next);
    paintTheme(next);
  });

  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const stage = document.querySelector(".hero-stage");
  const key = document.querySelector(".key-orbit");
  if (!stage || !key || reduce) return;

  let raf = 0;
  let targetX = 0;
  let targetY = 0;
  let currentX = 0;
  let currentY = 0;
  const started = performance.now();

  const tick = (now) => {
    const bob = Math.sin((now - started) / 900) * 10;
    currentX += (targetX - currentX) * 0.08;
    currentY += (targetY - currentY) * 0.08;
    key.style.transform = `translate3d(${currentX}px, ${currentY + bob}px, 0)`;
    raf = requestAnimationFrame(tick);
  };

  stage.addEventListener("pointermove", (event) => {
    const rect = stage.getBoundingClientRect();
    const nx = (event.clientX - rect.left) / rect.width - 0.5;
    const ny = (event.clientY - rect.top) / rect.height - 0.5;
    targetX = nx * 16;
    targetY = ny * 12;
  });

  stage.addEventListener("pointerleave", () => {
    targetX = 0;
    targetY = 0;
  });

  raf = requestAnimationFrame(tick);
  window.addEventListener("pagehide", () => cancelAnimationFrame(raf), { once: true });
})();
