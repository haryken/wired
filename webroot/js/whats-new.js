/* "Tính năng mới" tab: buttons jump to the matching setting. */

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
