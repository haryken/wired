# Wired UI i18n: bold / links must render as HTML

## Symptom

On the page you see literal tags instead of formatting:

```text
Chọn mốc <b>10–50 giây</b>; một nút…
```

instead of:

**Chọn mốc 10–50 giây; một nút…**

## Cause

`applyI18n()` in `webroot/js/i18n.js` defaults to `textContent`, which escapes markup.

Strings in `STRINGS` (vi/en/it/zh) that contain tags such as `<b>`, `<a>`, `<br>` must be applied with `innerHTML`.

## Required rule (when editing HTML)

If the translation (or the Vietnamese fallback inside the element) contains HTML tags, mark the element:

```html
<li data-i18n="news.silence_p2" data-i18n-html="1">
  Chọn mốc <b>10–50 giây</b>; …
</li>
```

**Always pair:**

| Attribute | When |
|-----------|------|
| `data-i18n="some.key"` | every translatable node |
| `data-i18n-html="1"` | whenever the string has `<b>`, `<strong>`, `<i>`, `<em>`, `<br>`, `<a …>`, `<span>`, `<code>` |

Plain text keys (no tags) must **not** set `data-i18n-html`.

## Safety net (runtime)

`applyI18n` also auto-detects allowlisted tags in the resolved translation and uses `innerHTML` even if `data-i18n-html` was forgotten.

Still set `data-i18n-html="1"` explicitly:

1. Makes intent obvious in reviews
2. Matches the Vietnamese fallback before JS runs
3. Documents which nodes are trusted markup

Do **not** put untrusted / user-supplied text into these keys. Keep the tag allowlist small; never add `<script>` or inline event handlers.

## Checklist before commit

1. Grep new/changed keys for tags:

   ```bash
   rg "<b>|<a |<br|<strong>" webroot/js/i18n.js
   ```

2. For each key used in HTML, confirm the element has `data-i18n-html="1"`:

   ```bash
   rg 'data-i18n="news.silence_p2"' webroot/index.html
   ```

3. Hard-refresh the page and switch language (vi/en) — bold and links must render, tags must not show as text.

4. Rebuild hotspot after `index.html` / `i18n.js` changes:

   ```bash
   python3 tools/build-hotspot-webroot.py
   ```

## Examples

Good:

```html
<p data-i18n="vol.help1" data-i18n-html="1">Có <b>hai chỉnh riêng</b>…</p>
```

```js
'vol.help1': 'Có <b>hai chỉnh riêng</b>: hệ thống (trên) và giao tiếp AI (dưới).',
```

Bad (missing `data-i18n-html` — tags show as text until/unless auto-detect runs; fallback still wrong before i18n):

```html
<li data-i18n="news.silence_p2">Chọn mốc <b>10–50 giây</b>; …</li>
```

## Related files

- `webroot/js/i18n.js` — `applyI18n`, `STRINGS`
- `webroot/index.html` — `data-i18n` / `data-i18n-html`
- `hotspot-webroot/index.html` — generated; do not edit by hand
