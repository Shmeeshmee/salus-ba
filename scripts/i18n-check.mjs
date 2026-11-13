// The catalog check (docs/i18n/SPEC.md §3 "Catalogs"), run by scripts/i18n-check.sh in the node service.
// Every language of an app has the source language's keys and no other, each message is valid ICU
// MessageFormat with the source's placeholders, and every plural has exactly the categories its language
// uses (CLDR, through Intl.PluralRules). Catalogs that do not exist yet are skipped.
//   web:    web/src/i18n/<area>/<language>.json against en.json; a value is an ICU string or
//           {"defaultMessage": "<ICU>", "description": "legal"}; the legal tag matches en's;
//   mobile: mobile/lib/l10n/app_<language>.arb against app_en.arb ("@" keys are metadata); app_sr.arb (the
//           base gen-l10n needs for the script variants) is the same text as app_sr_Cyrl.arb;
//   e-mail: internal/notification/templates/<language>/ against en/: the same files, the same {{…}} fields;
//   link:   internal/appointment/templates/link/<language>.json against en.json, in the ten patient languages, as web's.
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join, relative } from 'node:path';
import { pathToFileURL } from 'node:url';

const STAFF = ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en'];
const ALL = [...STAFF, 'de', 'ar', 'tr', 'it', 'fr'];
const ARB = ['en', 'bs', 'hr', 'sr_Latn', 'sr_Cyrl', 'sr', 'de', 'ar', 'tr', 'it', 'fr'];

// loadParser returns @formatjs/icu-messageformat-parser's parse, installed in web/ (react-intl brings it).
export function loadParser(repo) {
  const require = createRequire(join(repo, 'web', 'package.json'));
  return require('@formatjs/icu-messageformat-parser').parse;
}

// check returns the problems of the catalogs under root, one line each; none means the catalogs pass.
// parse is called only when there is a message to check.
export function check(root, parse) {
  const out = [];
  const report = (file, msg) => out.push(`${relative(root, file)}: ${msg}`);
  checkWeb(root, parse, report);
  checkMobile(root, parse, report);
  checkEmail(root, report);
  checkLink(root, parse, report);
  return out;
}

const dirs = (dir) => (existsSync(dir) ? readdirSync(dir).filter((d) => statSync(join(dir, d)).isDirectory()).sort() : []);

function readJSON(file, report) {
  try {
    return JSON.parse(readFileSync(file, 'utf8'));
  } catch {
    report(file, 'not valid JSON');
    return null;
  }
}

function checkWeb(root, parse, report) {
  const base = join(root, 'web', 'src', 'i18n');
  for (const area of dirs(base)) {
    const dir = join(base, area);
    const files = readdirSync(dir).filter((f) => f.endsWith('.json'));
    if (files.length === 0) continue;
    for (const f of files) {
      if (!STAFF.includes(f.slice(0, -'.json'.length))) report(join(dir, f), 'not a language of this app');
    }
    const read = (lang) => readCatalog(join(dir, `${lang}.json`), report);
    compare(read('en'), STAFF.filter((l) => l !== 'en').map((l) => [l, read(l)]), parse, report, false);
  }
}

// readCatalog reads a web-style catalog (a value is an ICU string or {defaultMessage, description}), or reports it
// missing or broken and returns undefined.
function readCatalog(file, report) {
  if (!existsSync(file)) {
    report(file, 'missing');
    return undefined;
  }
  const json = readJSON(file, report);
  if (json === null) return undefined;
  const messages = {};
  for (const [key, value] of Object.entries(json)) {
    if (typeof value === 'string') messages[key] = { text: value, legal: false };
    else if (value && typeof value.defaultMessage === 'string') messages[key] = { text: value.defaultMessage, legal: value.description === 'legal' };
    else report(file, `${key}: neither a string nor {defaultMessage, description}`);
  }
  return { file, messages };
}

// checkLink checks the one-time link's page (internal/appointment/templates/link/<language>.json, SAL-34) in the ten
// patient languages against en.json, as the web's catalogs.
function checkLink(root, parse, report) {
  const dir = join(root, 'internal', 'appointment', 'templates', 'link');
  if (!existsSync(join(dir, 'en.json'))) return;
  for (const f of readdirSync(dir).filter((f) => f.endsWith('.json'))) {
    if (!ALL.includes(f.slice(0, -'.json'.length))) report(join(dir, f), 'not a language of this app');
  }
  const read = (lang) => readCatalog(join(dir, `${lang}.json`), report);
  compare(read('en'), ALL.filter((l) => l !== 'en').map((l) => [l, read(l)]), parse, report, false);
}

function checkMobile(root, parse, report) {
  const dir = join(root, 'mobile', 'lib', 'l10n');
  if (!existsSync(join(dir, 'app_en.arb'))) return;
  for (const f of readdirSync(dir).filter((f) => f.endsWith('.arb'))) {
    if (!ARB.includes(f.slice('app_'.length, -'.arb'.length))) report(join(dir, f), 'not a language of this app');
  }
  const read = (lang) => {
    const file = join(dir, `app_${lang}.arb`);
    if (!existsSync(file)) {
      report(file, 'missing');
      return undefined;
    }
    const json = readJSON(file, report);
    if (json === null) return undefined;
    const messages = {};
    for (const [key, value] of Object.entries(json)) {
      if (key.startsWith('@')) continue;
      if (typeof value === 'string') messages[key] = { text: value, legal: false };
      else report(file, `${key}: not a string`);
    }
    return { file, messages };
  };
  const others = ARB.filter((l) => l !== 'en').map((l) => [l.replace('_', '-'), read(l)]);
  compare(read('en'), others, parse, report, true);
  const sr = others.find(([l]) => l === 'sr')[1];
  const cyrl = others.find(([l]) => l === 'sr-Cyrl')[1];
  if (sr && cyrl && JSON.stringify(sr.messages) !== JSON.stringify(cyrl.messages)) report(sr.file, 'differs from app_sr_Cyrl.arb');
}

// compare checks every language's catalog against the source's. The legal tag is compared only where the
// catalogs carry it per language (web); ARB metadata lives in the template alone.
function compare(source, others, parse, report, arb) {
  if (!source) return;
  const shapes = {};
  for (const [key, m] of Object.entries(source.messages)) shapes[key] = shape(source.file, key, m.text, 'en', parse, report, arb);
  for (const [lang, cat] of others) {
    if (!cat) continue;
    for (const key of Object.keys(source.messages)) {
      if (!(key in cat.messages)) report(cat.file, `missing key ${key}`);
    }
    for (const [key, m] of Object.entries(cat.messages)) {
      if (!(key in source.messages)) {
        report(cat.file, `extra key ${key}`);
        continue;
      }
      if (!arb && m.legal !== source.messages[key].legal) report(cat.file, `${key}: the legal tag differs from en`);
      const got = shape(cat.file, key, m.text, lang, parse, report, arb);
      const want = shapes[key];
      if (got !== undefined && want !== undefined && got !== want) {
        report(cat.file, `${key}: placeholders ${got || 'none'}, want ${want || 'none'}`);
      }
    }
  }
}

// shape parses one message and checks its plurals against lang's CLDR categories; it returns the message's
// placeholders by name ("name (kind)" for all but a plain argument), or undefined when it is not valid ICU.
function shape(file, key, text, lang, parse, report, arb) {
  let ast;
  try {
    ast = parse(text, { ignoreTag: arb, requiresOtherClause: true });
  } catch (e) {
    report(file, `${key}: invalid ICU (${e.message})`);
    return undefined;
  }
  const args = new Map();
  walk(ast, (el) => {
    const kind = KINDS[el.type] === 'plural' && el.pluralType === 'ordinal' ? 'selectordinal' : KINDS[el.type];
    if (!kind) return;
    args.set(el.value, kind);
    if (kind === 'plural' || kind === 'selectordinal') {
      const rules = new Intl.PluralRules(lang, { type: kind === 'plural' ? 'cardinal' : 'ordinal' });
      const want = rules.resolvedOptions().pluralCategories;
      const have = Object.keys(el.options).filter((k) => !k.startsWith('='));
      const lacking = want.filter((c) => !have.includes(c) && !OPTIONAL[lang]?.includes(c)).sort();
      const unused = have.filter((c) => !want.includes(c)).sort();
      if (lacking.length) report(file, `${key}: plural ${el.value} lacks ${lacking.join(', ')}`);
      if (unused.length) report(file, `${key}: plural ${el.value} has ${unused.join(', ')}, which ${lang} does not use`);
    }
  });
  return [...args].map(([name, kind]) => (kind === 'argument' ? name : `${name} (${kind})`)).sort().join(', ');
}

// Categories CLDR has that a message may leave to other: French and Italian "many" is for millions written
// in compact or exponent form ("1 million de", "1e6"), which the apps never format.
const OPTIONAL = { fr: ['many'], it: ['many'] };

// The parser's element types (TYPE in @formatjs/icu-messageformat-parser) that carry an argument name.
const KINDS = { 1: 'argument', 2: 'number', 3: 'date', 4: 'time', 5: 'select', 6: 'plural', 8: 'tag' };

function walk(elements, visit) {
  for (const el of elements) {
    visit(el);
    if (el.options) for (const opt of Object.values(el.options)) walk(opt.value, visit);
    if (el.children) walk(el.children, visit);
  }
}

function checkEmail(root, report) {
  const base = join(root, 'internal', 'notification', 'templates');
  const en = join(base, 'en');
  if (!existsSync(en)) return;
  const fields = (file) => [...new Set(readFileSync(file, 'utf8').match(/\{\{[^}]*\}\}/g) ?? [])].map((f) => f.replace(/\s+/g, ' ')).sort();
  const files = readdirSync(en).sort();
  for (const lang of dirs(base)) {
    if (!ALL.includes(lang)) report(join(base, lang), 'not a language of this app');
  }
  for (const lang of ALL.filter((l) => l !== 'en')) {
    const dir = join(base, lang);
    const have = existsSync(dir) ? readdirSync(dir) : [];
    for (const f of have) if (!files.includes(f)) report(join(dir, f), 'not in en/');
    for (const f of files) {
      const file = join(dir, f);
      if (!existsSync(file)) {
        report(file, 'missing');
        continue;
      }
      const got = fields(file).join(' ');
      const want = fields(join(en, f)).join(' ');
      if (got !== want) report(file, `template fields ${got || 'none'}, want ${want || 'none'}`);
    }
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const repo = process.argv[2] ?? process.cwd();
  let parser;
  const problems = check(repo, (text, opts) => {
    parser ??= loadParser(repo);
    return parser(text, opts);
  });
  if (problems.length) {
    console.error(`i18n-check: ${problems.length} problem(s)\n${problems.join('\n')}`);
    process.exit(1);
  }
  console.log('i18n-check: catalogs pass');
}
