import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';

// Read manifests and filenames only. Never execute discovered project scripts.
const root = path.resolve(process.argv[2] || 'D:/Project');
const out = path.resolve(process.argv[3] || path.join(root, 'dev-toolbox/docs/project-inventory.json'));
const excluded = ['.git', 'node_modules', 'vendor', 'dist', 'build', 'target', '.venv', 'venv', '.gradle', '.idea', '.next', '__pycache__', '.claude', '.codex', '.mimosa', '.zcode', '.local', '.cache', 'verify-libs', 'profiles', 'payload', 'workspace', 'site-packages', 'local_modules'];
const globs = ['package.json', 'go.mod', 'pom.xml', 'build.gradle*', 'pyproject.toml', 'requirements*.txt', 'Cargo.toml', '*.csproj', 'pubspec.yaml', 'CMakeLists.txt', 'Makefile', '*.uvproj*', '.nvmrc', '.node-version', '.python-version', '.tool-versions', 'rust-toolchain*', 'gradle-wrapper.properties', 'pnpm-lock.yaml', 'yarn.lock', 'package-lock.json', 'poetry.lock', 'uv.lock', 'Dockerfile*', '*compose*.yml', '*compose*.yaml', '*.java', '*.kt', '*.py', '*.vue', '*.tsx', '*.ts', '*.rs', '*.c', '*.cpp', '*.html', '*.js', '*.php'];
// Put exclusions last: ripgrep resolves overlapping globs in command-line order.
const args = ['--files', '--hidden', '--no-ignore', ...globs.flatMap(x => ['-g', x]), ...excluded.flatMap(x => ['-g', `!**/${x}/**`]), root];
const names = execFileSync('rg', args, { encoding: 'utf8', maxBuffer: 32 * 1024 * 1024 }).split(/\r?\n/).filter(Boolean).sort();
const dirs = fs.readdirSync(root, { withFileTypes: true }).filter(x => x.isDirectory() && x.name !== 'dev-toolbox').map(x => x.name).sort();
const projects = dirs.map(name => ({ name, path: path.join(root, name), sourceExtensions: {}, manifests: [], markers: [] }));
const byName = new Map(projects.map(x => [x.name, x]));
const manifestPattern = /^(package\.json|go\.mod|pom\.xml|build\.gradle(?:\.kts)?|pyproject\.toml|requirements[^/]*\.txt|Cargo\.toml|.*\.csproj|pubspec\.yaml|CMakeLists\.txt|Makefile|.*\.uvprojx?)$/;
const archive = /(?:^|\/)(归档|archive[^/]*|[^/]+\.old)(?:\/|$)/i;
for (const filename of names) {
  const relative = path.relative(root, filename).replaceAll('\\', '/');
  const [projectName, ...parts] = relative.split('/');
  if (parts.slice(0, -1).some(x => excluded.includes(x))) continue;
  const project = byName.get(projectName);
  if (!project) continue;
  const subpath = parts.join('/');
  const base = path.basename(filename);
  const ext = path.extname(filename).toLowerCase();
  if (/^\.(java|kt|py|vue|tsx|ts|rs|c|cpp|html|js|php)$/.test(ext)) project.sourceExtensions[ext] = (project.sourceExtensions[ext] || 0) + 1;
  if (!manifestPattern.test(base)) {
    if (!/^\.(java|kt|py|vue|tsx|ts|rs|c|cpp|html|js|php)$/.test(ext)) project.markers.push(subpath);
    continue;
  }
  const record = { path: subpath, archived: archive.test(subpath), bundled: /(?:^|\/)(pymavlink|mavlink)(?:\/|$)/i.test(subpath) };
  try {
    const text = fs.readFileSync(filename, 'utf8').replace(/^\uFEFF/, '');
    if (base === 'package.json') {
      const j = JSON.parse(text);
      Object.assign(record, { kind: 'node', name: j.name, engines: j.engines, packageManager: j.packageManager, dependencies: j.dependencies, devDependencies: j.devDependencies, workspaces: j.workspaces });
    } else {
      record.kind = base === 'pom.xml' || base.startsWith('build.gradle') ? 'java' : base === 'go.mod' ? 'go' : base === 'Cargo.toml' ? 'rust' : base === 'pyproject.toml' || base.startsWith('requirements') ? 'python' : 'native-or-other';
      const signal = /python\s*=|requires-python|java\.version|maven\.compiler|sourceCompatibility|targetCompatibility|VERSION_\d|jvmToolchain|languageVersion|compileSdk|targetSdk|com\.android|springframework\.boot|spring-boot-starter-parent|<version>|^go |^toolchain |^module |tauri|rust-version|^edition\s*=|flask|fastapi|django|poetry|uvicorn|pymongo|motor\s*=|chromadb|mysql|postgres|redis|ARMCC|arm-none-eabi|mcpu|^PREFIX|^image\s*:/i;
      record.signals = text.split(/\r?\n/).filter(x => signal.test(x)).map(x => x.trim().slice(0, 200)).slice(0, 30);
      if (record.kind === 'python') record.dependencyNames = text.split(/\r?\n/).map(x => x.match(/^([A-Za-z][A-Za-z0-9_.-]*)\s*(?:=|[<>=!~])/ )?.[1]).filter(Boolean);
    }
  } catch (e) { record.readError = e.message; }
  project.manifests.push(record);
}
const data = { schemaVersion: 1, scannedAt: new Date().toISOString(), timezone: 'Asia/Shanghai', root, method: 'ripgrep filename enumeration; static manifests; no project commands executed', excludedDirectories: excluded, limitations: ['Directories are project groups, not guaranteed independent repositories.', 'Declared versions are requirements or compile targets, not proof of installed or working runtimes.', 'Archived and bundled manifests are retained and labeled; source extension counts may include static templates and bundled source.', 'No inference of activity from modification time; no environment files, credentials, database URLs, or private configuration read.', 'Lockfile and container markers record presence only, not complete dependency or service inventories.'], projects };
fs.mkdirSync(path.dirname(out), { recursive: true });
fs.writeFileSync(out, JSON.stringify(data, null, 2) + '\n');
for (const p of projects) {
  console.log(`${p.name}: ${p.manifests.length} manifests; ${Object.entries(p.sourceExtensions).map(([k,v]) => `${k}:${v}`).join(' ')}`);
  for (const m of p.manifests.filter(x => !x.bundled)) console.log(`  ${m.path}: ${m.kind} ${m.engines ? JSON.stringify(m.engines) : ''} ${m.dependencies ? Object.keys(m.dependencies).join(', ') : (m.signals || []).slice(0, 5).join(' | ')}`);
}
console.log(`Saved ${out}`);
