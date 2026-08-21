/* Shared WireOS confirm modal — replaces browser confirm() for dark UI. */
(function () {
    'use strict';

    function ensureModal() {
        let root = document.getElementById('wireosConfirmModal');
        if (root) return root;
        root = document.createElement('div');
        root.id = 'wireosConfirmModal';
        root.className = 'wireos-modal';
        root.hidden = true;
        root.innerHTML =
            '<div class="wireos-modal-backdrop" data-wireos-confirm-cancel></div>' +
            '<div class="wireos-modal-panel" role="dialog" aria-modal="true" aria-labelledby="wireosConfirmTitle">' +
            '  <h3 id="wireosConfirmTitle" class="wireos-modal-title"></h3>' +
            '  <p id="wireosConfirmBody" class="wireos-modal-body"></p>' +
            '  <div class="wireos-modal-actions">' +
            '    <button type="button" class="wireos-modal-btn secondary" data-wireos-confirm-cancel>Hủy</button>' +
            '    <button type="button" class="wireos-modal-btn primary" data-wireos-confirm-ok>Xác nhận</button>' +
            '  </div>' +
            '</div>';
        document.body.appendChild(root);
        return root;
    }

    /**
     * @param {string| {title?:string, message:string, ok?:string, cancel?:string}} opts
     * @returns {Promise<boolean>}
     */
    function wireosConfirm(opts) {
        const o = (typeof opts === 'string') ? { message: opts } : (opts || {});
        const title = o.title || (typeof t === 'function' ? t('confirm.title', 'Xác nhận') : 'Xác nhận');
        const message = o.message || '';
        const okLabel = o.ok || (typeof t === 'function' ? t('confirm.ok', 'Xác nhận') : 'Xác nhận');
        const cancelLabel = o.cancel || (typeof t === 'function' ? t('confirm.cancel', 'Hủy') : 'Hủy');

        const root = ensureModal();
        const titleEl = document.getElementById('wireosConfirmTitle');
        const bodyEl = document.getElementById('wireosConfirmBody');
        const okBtn = root.querySelector('[data-wireos-confirm-ok]');
        const cancelBtns = root.querySelectorAll('[data-wireos-confirm-cancel]');

        titleEl.textContent = title;
        bodyEl.textContent = message;
        okBtn.textContent = okLabel;
        cancelBtns.forEach((b) => {
            if (b.tagName === 'BUTTON') b.textContent = cancelLabel;
        });

        root.hidden = false;
        document.body.classList.add('wireos-modal-open');

        return new Promise((resolve) => {
            function cleanup(result) {
                root.hidden = true;
                document.body.classList.remove('wireos-modal-open');
                okBtn.removeEventListener('click', onOk);
                cancelBtns.forEach((b) => b.removeEventListener('click', onCancel));
                document.removeEventListener('keydown', onKey);
                resolve(result);
            }
            function onOk() { cleanup(true); }
            function onCancel() { cleanup(false); }
            function onKey(e) {
                if (e.key === 'Escape') onCancel();
                if (e.key === 'Enter') onOk();
            }
            okBtn.addEventListener('click', onOk);
            cancelBtns.forEach((b) => b.addEventListener('click', onCancel));
            document.addEventListener('keydown', onKey);
            okBtn.focus();
        });
    }

    window.wireosConfirm = wireosConfirm;
})();
