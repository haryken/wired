#!/usr/bin/env python3
"""Generate the standalone hotspot UI from selected wired sections."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "webroot" / "index.html"
OUTPUT = ROOT / "hotspot-webroot" / "index.html"


def section(source: str, start: str, end: str) -> str:
    begin = source.index(start)
    finish = source.index(end, begin)
    return source[begin:finish].rstrip()


source = SOURCE.read_text(encoding="utf-8")
bot = section(source, '    <div id="botsettings"', '    <div id="aud"')
xiaozhi = section(source, '    <div id="xiaozhi"', '    <div id="voice"')
control = section(source, '    <div id="control"', '    <div id="freetime"')

bot = bot.replace(
    'class="container mod-section tab-content active"',
    'class="container mod-section tab-content active"',
    1,
)

scripts = [
    "freq.js",
    "wake-engine.js",
    "wakeword-pv.js",
    "sensitivity-pv.js",
    "jdoc-settings.js",
    "faces.js",
    "photos.js",
    "xiaozhi.js",
    "alexa.js",
    "control.js",
    "stim-stats.js",
    "petting.js",
    "wifi-mode.js",
    "listen-mode.js",
    "music-settings.js",
]
script_tags = "\n".join(
    f'    <script src="/portal-assets/js/{name}"></script>' for name in scripts
)

document = f"""<!doctype html>
<html lang="vi">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <title>Vector · Hotspot</title>
    <link rel="icon" type="image/png" href="/portal-assets/vector-logo-32.png">
    <link rel="stylesheet" href="/portal-assets/style.css">
    <link rel="stylesheet" href="/hotspot/portal.css">
</head>
<body>
    <header class="site-header">
        <div class="header-brand-row">
            <div class="brand">
                <img class="brand-logo" src="/portal-assets/vector-logo.png" width="44" height="44" alt="Vector">
            </div>
            <button type="button" id="portalBurger" class="nav-burger" aria-label="Mở menu">
                <span class="nav-burger-line"></span><span class="nav-burger-line"></span><span class="nav-burger-line"></span>
            </button>
        </div>
        <div class="header-title-row"><h1>Vector · Hotspot</h1></div>
    </header>
    <div id="portalBackdrop" class="nav-backdrop" hidden></div>
    <aside id="portalDrawer" class="nav-drawer" aria-hidden="true">
        <div class="nav-drawer-head">
            <div class="nav-drawer-brand">
                <img class="brand-logo" src="/portal-assets/vector-logo.png" width="40" height="40" alt="Vector">
                <div class="nav-drawer-hello"><span class="nav-drawer-hello-sub">Xin chào!</span><span class="nav-drawer-hello-name">Vector</span></div>
            </div>
            <button type="button" id="portalClose" class="nav-drawer-close" aria-label="Đóng menu">&times;</button>
        </div>
        <nav class="nav-drawer-list">
            <a href="/wifi">📶 Bắt WiFi</a>
            <button type="button" data-target="#botsettings">⚙️ Cài đặt bot</button>
            <button type="button" data-target="#xiaozhi">💜 Xiaozhi</button>
            <button type="button" data-target="#control">🎮 Điều khiển</button>
        </nav>
    </aside>
    <nav class="site-nav portal-tabs" aria-label="Chức năng hotspot">
        <a href="/wifi">📶 Bắt WiFi</a>
        <button type="button" class="active" data-target="#botsettings">⚙️ Cài đặt bot</button>
        <button type="button" data-target="#xiaozhi">💜 Xiaozhi</button>
        <button type="button" data-target="#control">🎮 Điều khiển</button>
    </nav>

{bot}

{xiaozhi}

{control}

    <div id="showDuringVicRestart" class="container" style="display:none"><h2>Robot đang khởi động lại, vui lòng chờ…</h2></div>
    <div id="restartNeeded" class="container" style="display:none"><h2>Cần khởi động lại để áp dụng thay đổi.</h2><button id="restartButton" onclick="RestartVic()">Khởi động lại</button></div>
    <script src="/hotspot/portal.js"></script>
{script_tags}
</body>
</html>
"""

OUTPUT.parent.mkdir(parents=True, exist_ok=True)
OUTPUT.write_text(document, encoding="utf-8")
print(f"generated {OUTPUT}")
