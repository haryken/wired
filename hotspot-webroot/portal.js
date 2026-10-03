function portalActivate(target) {
    if (!target || !document.querySelector(target)) target = '#botsettings';
    document.querySelectorAll('.tab-content').forEach((panel) => {
        panel.classList.toggle('active', '#' + panel.id === target);
    });
    document.querySelectorAll('[data-target]').forEach((button) => {
        button.classList.toggle('active', button.dataset.target === target);
    });
    history.replaceState(null, '', '/portal/' + target);
    portalCloseDrawer();

    if (target === '#xiaozhi' && typeof xzLoad === 'function') xzLoad();
    if (target === '#control' && typeof ctrlStatus === 'function') ctrlStatus();
}

function portalOpenDrawer() {
    const drawer = document.getElementById('portalDrawer');
    const backdrop = document.getElementById('portalBackdrop');
    if (drawer) {
        drawer.classList.add('is-open');
        drawer.setAttribute('aria-hidden', 'false');
    }
    if (backdrop) backdrop.hidden = false;
}

function portalCloseDrawer() {
    const drawer = document.getElementById('portalDrawer');
    const backdrop = document.getElementById('portalBackdrop');
    if (drawer) {
        drawer.classList.remove('is-open');
        drawer.setAttribute('aria-hidden', 'true');
    }
    if (backdrop) backdrop.hidden = true;
}

function hide(id) {
    const element = document.getElementById(id);
    if (element) element.style.display = 'none';
}

function show(id) {
    const element = document.getElementById(id);
    if (element) element.style.display = 'block';
}

async function GetCurrent(mod) {
    const response = await fetch('/api/mods/' + encodeURIComponent(mod) + '/get');
    return response.text();
}

function SetModStatus(message) {
    const element = document.getElementById('modStatus');
    if (!element) return;
    element.innerHTML = '<h3>' + message + '</h3>';
    element.style.display = message ? 'block' : 'none';
}

function HideModStatus() {
    hide('modStatus');
}

document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-target]').forEach((button) => {
        button.addEventListener('click', () => portalActivate(button.dataset.target));
    });
    document.getElementById('portalBurger')?.addEventListener('click', portalOpenDrawer);
    document.getElementById('portalClose')?.addEventListener('click', portalCloseDrawer);
    document.getElementById('portalBackdrop')?.addEventListener('click', portalCloseDrawer);

    const initial = ['#botsettings', '#xiaozhi', '#control'].includes(location.hash)
        ? location.hash
        : '#botsettings';
    portalActivate(initial);

    GetCurrent('FreqChange').then((value) => {
        document.getElementsByName('frequency').forEach((radio) => {
            radio.checked = radio.value === value;
        });
    }).catch(() => {});
    if (typeof setSensitivity === 'function') setSensitivity();
    if (typeof getTimezone === 'function') getTimezone();
    if (typeof getLocation === 'function') getLocation();
    if (typeof getTempUnits === 'function') getTempUnits();
    if (typeof getMasterVolume === 'function') getMasterVolume();
    if (typeof getEyePreset === 'function') getEyePreset();
});
