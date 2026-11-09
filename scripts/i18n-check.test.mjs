// Tests of scripts/i18n-check.mjs (node --test), on catalogs written to a temporary folder.
import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { after, test } from 'node:test';
import { fileURLToPath } from 'node:url';

import { check, loadParser } from './i18n-check.mjs';

const repo = join(dirname(fileURLToPath(import.meta.url)), '..');
const parse = loadParser(repo);
const roots = [];
after(() => roots.forEach((r) => rmSync(r, { recursive: true, force: true })));

// tree writes files ({path: content}, objects as JSON) into a new temporary root and returns it.
function tree(files) {
  const root = mkdtempSync(join(tmpdir(), 'i18n-check-'));
  roots.push(root);
  for (const [path, content] of Object.entries(files)) {
    mkdirSync(dirname(join(root, path)), { recursive: true });
    writeFileSync(join(root, path), typeof content === 'string' ? content : JSON.stringify(content));
  }
  return root;
}

const staff = ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en'];
const patient = ['en', 'bs', 'hr', 'sr_Latn', 'sr_Cyrl', 'sr', 'de', 'ar', 'tr', 'it', 'fr'];
const email = ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr'];

// web is one area's catalogs: the same messages in every staff language, then the overrides.
function web(messages, overrides = {}) {
  const files = {};
  for (const lang of staff) files[`web/src/i18n/shell/${lang}.json`] = messages;
  for (const [lang, m] of Object.entries(overrides)) files[`web/src/i18n/shell/${lang}.json`] = m;
  return files;
}

// mobile is the ARB catalogs, as web.
function mobile(messages, overrides = {}) {
  const files = {};
  for (const lang of patient) files[`mobile/lib/l10n/app_${lang}.arb`] = { '@@locale': lang, ...messages };
  for (const [lang, m] of Object.entries(overrides)) files[`mobile/lib/l10n/app_${lang}.arb`] = { '@@locale': lang, ...m };
  return files;
}

const problems = (files) => check(tree(files), parse);

test('passes with no catalogs at all', () => {
  assert.deepEqual(check(tree({ 'README.md': 'x' }), parse), []);
});

test('passes on empty catalogs and on matching ones in both value shapes', () => {
  assert.deepEqual(problems(web({})), []);
  const plural = '{count, plural, one {# file} few {# files} other {# files}}';
  const files = web(
    { title: 'Settings', files: { defaultMessage: plural, description: 'legal' } },
    {
      en: { title: 'Settings', files: { defaultMessage: '{count, plural, one {# file} other {# files}}', description: 'legal' } },
      'sr-Cyrl': { title: 'Подешавања', files: { defaultMessage: '{count, plural, one {# фајл} few {# фајла} other {# фајлова}}', description: 'legal' } },
    },
  );
  assert.deepEqual(problems(files), []);
});

test('fails on a missing or an extra key, and a missing language file', () => {
  const files = web({ a: 'A', b: 'B' }, { hr: { a: 'A' }, bs: { a: 'A', b: 'B', c: 'C' } });
  delete files['web/src/i18n/shell/sr-Latn.json'];
  const got = problems(files).join('\n');
  assert.match(got, /shell\/hr\.json: missing key b/);
  assert.match(got, /shell\/bs\.json: extra key c/);
  assert.match(got, /shell\/sr-Latn\.json: missing/);
});

test('fails on a language the app does not have', () => {
  const files = web({}, { de: {} });
  assert.match(problems(files).join('\n'), /shell\/de\.json: not a language of this app/);
});

test('fails on invalid ICU, a placeholder mismatch, and a legal tag only in English', () => {
  const files = web(
    { broken: 'Hello', name: 'Hello {name}', legal: { defaultMessage: 'Consent', description: 'legal' } },
    { bs: { broken: 'Zdravo {', name: 'Zdravo {ime}', legal: 'Saglasnost' } },
  );
  const got = problems(files).join('\n');
  assert.match(got, /shell\/bs\.json: broken: invalid ICU/);
  assert.match(got, /shell\/bs\.json: name: placeholders ime, want name/);
  assert.match(got, /shell\/bs\.json: legal: the legal tag differs from en/);
});

test('fails on plural categories a language lacks or does not have', () => {
  const english = '{count, plural, one {# day} other {# days}}';
  const files = web({ days: english }, { bs: { days: english }, hr: { days: '{count, plural, one {#} few {#} many {#} other {#}}' } });
  const got = problems(files).join('\n');
  assert.match(got, /shell\/bs\.json: days: plural count lacks few/);
  assert.match(got, /shell\/hr\.json: days: plural count has many, which hr does not use/);
});

test('checks the ARB catalogs against app_en.arb, with Arabic plurals and app_sr.arb as app_sr_Cyrl.arb', () => {
  const english = { tryAgain: 'Try again', '@tryAgain': { description: 'Button' }, n: '{n, plural, one {#} other {#}}' };
  const ar = { tryAgain: 'إعادة المحاولة', n: '{n, plural, zero {#} one {#} two {#} few {#} many {#} other {#}}' };
  const slavic = { tryAgain: 'x', n: '{n, plural, one {#} few {#} other {#}}' };
  const ok = mobile(english, { ar, bs: slavic, hr: slavic, sr_Latn: slavic, sr_Cyrl: slavic, sr: slavic });
  assert.deepEqual(problems(ok), []);

  const bad = mobile(english, { ar: { tryAgain: 'x', n: '{n, plural, one {#} other {#}}' }, bs: slavic, hr: slavic, sr_Latn: slavic, sr_Cyrl: slavic, sr: { ...slavic, tryAgain: 'y' } });
  const got = problems(bad).join('\n');
  assert.match(got, /app_ar\.arb: n: plural n lacks few, many, two, zero/);
  assert.match(got, /app_sr\.arb: differs from app_sr_Cyrl\.arb/);
});

test('checks the e-mail templates against en/: the same files and the same template fields', () => {
  const files = {};
  for (const lang of email) {
    files[`internal/notification/templates/${lang}/invite.txt`] = 'Hello {{.Clinic}}';
  }
  assert.deepEqual(problems(files), []);
  files['internal/notification/templates/de/invite.txt'] = 'Hallo {{.Klinik}}';
  files['internal/notification/templates/fr/extra.txt'] = 'x';
  delete files['internal/notification/templates/ar/invite.txt'];
  const got = problems(files).join('\n');
  assert.match(got, /templates\/de\/invite\.txt: template fields \{\{\.Klinik\}\}, want \{\{\.Clinic\}\}/);
  assert.match(got, /templates\/fr\/extra\.txt: not in en\//);
  assert.match(got, /templates\/ar\/invite\.txt: missing/);
});
