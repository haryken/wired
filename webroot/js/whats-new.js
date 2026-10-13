/* "Tính năng mới" tab: buttons jump to the matching setting. */

const NEWS_RECENT_DAYS = 7;
const NEWS_LOCALES = { vi: 'vi-VN', en: 'en-US', it: 'it-IT', zh: 'zh-CN' };

function newsFormatDate(iso, lang) {
    const parts = iso.split('-').map(Number);
    const d = new Date(parts[0], parts[1] - 1, parts[2]);
    const opts = (lang === 'en' || lang === 'zh')
        ? { year: 'numeric', month: (lang === 'en' ? 'short' : 'long'), day: 'numeric' }
        : { year: 'numeric', month: '2-digit', day: '2-digit' };
    try {
        return new Intl.DateTimeFormat(NEWS_LOCALES[lang] || 'vi-VN', opts).format(d);
    } catch (e) {
        return iso;
    }
}

function newsRenderDates() {
    const lang = (window.I18N && I18N.getLang) ? I18N.getLang() : 'vi';
    const label = (typeof xzT === 'function') ? xzT('news.added', 'Thêm ngày') : 'Thêm ngày';
    const now = Date.now();
    document.querySelectorAll('#news .news-card[data-added]').forEach(function (card) {
        const iso = card.getAttribute('data-added');
        const timeEl = card.querySelector('.news-date');
        if (timeEl) timeEl.textContent = '🗓 ' + label + ' ' + newsFormatDate(iso, lang);
        const badge = card.querySelector('.news-new');
        const parts = iso.split('-').map(Number);
        const ageDays = (now - new Date(parts[0], parts[1] - 1, parts[2]).getTime()) / 86400000;
        if (badge) badge.hidden = !(ageDays >= 0 && ageDays < NEWS_RECENT_DAYS);
    });
}

document.addEventListener('DOMContentLoaded', newsRenderDates);
document.addEventListener('wireos-i18n', newsRenderDates);

function newsGo(target, sub) {
    if (typeof activateSection === 'function') activateSection(target);
    if (target === '#botsettings' && sub && typeof showBotSection === 'function') {
        showBotSection(sub);
        const sec = document.getElementById(sub);
        if (sec) sec.scrollIntoView({ behavior: 'smooth', block: 'start' });
        return;
    }
    if (target === '#xiaozhi' && sub === 'course') {
        const block = document.getElementById('xzConfigBlock');
        const xzOn = block && block.style.display !== 'none';
        if (xzOn && typeof xzShowPresetUI === 'function') {
            // Only shows the form; nothing is applied until "Lưu & áp dụng".
            xzShowPresetUI('course');
            if (!(typeof xzLastCfg !== 'undefined' && xzLastCfg && xzLastCfg.identity_mode === 'course')
                && typeof xzSetElStatus === 'function') {
                xzSetElStatus('xzCourseStatus', xzT('xz.course_pick', 'Chọn khóa học, giọng, bài rồi bấm Lưu & áp dụng.'), false);
            }
            const fields = document.getElementById('xzCourseFields');
            if (fields) fields.scrollIntoView({ behavior: 'smooth', block: 'start' });
            return;
        }
        if (typeof xzSetModeStatus === 'function') {
            xzSetModeStatus(xzT('news.course_need_xz', 'Chọn "Xiaozhi (cloud chat)" trước, sau đó chọn "Dạy tiếng Anh (khóa học)".'), false);
        }
        const radios = document.getElementById('xzModeXiaozhi');
        if (radios) radios.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
}
