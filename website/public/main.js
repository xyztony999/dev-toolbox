(() => {
  const I18N = {
    zh: {
      "meta.title": "DevToolbox — 项目环境与 Git 身份工具箱",
      "meta.description": "DevToolbox 帮你管好多技术栈项目的运行时和 Git 身份：先看清，再确认，再改。",
      "meta.og": "这个项目用哪套运行时、哪个 Git 身份，一眼看清。",
      "nav.capabilities": "功能",
      "nav.how": "怎么用",
      "hero.eyebrow": "开发环境 · Git 身份",
      "hero.lede": "先看清这个项目该用哪套运行时、哪个 Git 身份，确认后再改。不会悄悄动你的全局配置。",
      "hero.download": "下载",
      "hero.source": "源码",
      "chip.git": "切换 Git 身份",
      "chip.path": "不改系统 PATH",
      "cap.title": "能做什么",
      "cap.lede": "命令行和桌面共用同一套 Go 核心，界面只负责展示和发起操作。",
      "cap.project.title": "项目",
      "cap.project.body": "登记、扫描本机项目，识别 Git 仓库和构建模块，读出版本要求。只读扫描，不会跑项目脚本。",
      "cap.runtime.title": "运行时",
      "cap.runtime.body": "支持 Node、Java、Maven、Gradle、Android SDK、Go、Python：发现、检查、安装。先预览，再确认。",
      "cap.git.title": "Git 身份",
      "cap.git.body": "多个身份档案，可按目录绑定；应用时完整覆盖，凭据不落盘。可先预览再执行，也支持批量仓库。",
      "how.title": "怎么用",
      "how.lede": "安装和切换都要你点确认。环境主要作用在新开的终端和子进程上。",
      "how.s1.title": "选项目",
      "how.s1.body": "打开已登记的项目，直接看到运行时要求和当前 Git 身份。",
      "how.s2.title": "先预览",
      "how.s2.body": "把版本、校验、目标路径和体积列清楚，你点确认后再动手。",
      "how.s3.title": "再生效",
      "how.s3.body": "对新开终端和子进程生效，不会悄悄改全局默认。",
      "dl.title": "下载 Windows / Linux 版本",
      "dl.lede": "安装包在 GitHub Releases。",
      "dl.get": "下载最新版",
      "dl.readme": "看 README",
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
