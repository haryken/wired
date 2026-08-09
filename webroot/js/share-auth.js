/* Keep remote-share auth working when cookies are dropped (Messenger/in-app browsers). */
(function () {
    'use strict';
    var m = location.pathname.match(/^\/r\/([a-f0-9]+)(?=\/|$)/i);
    var token = m ? m[1] : '';
    if (!token) {
        try {
            token = new URLSearchParams(location.search).get('share') || '';
        } catch (_) {}
    }
    if (!token) return;

    try {
        sessionStorage.setItem('wired_share', token);
    } catch (_) {}

    var origFetch = window.fetch;
    window.fetch = function (input, init) {
        init = init ? Object.assign({}, init) : {};
        var headers = new Headers(init.headers || (typeof input !== 'string' && input && input.headers) || undefined);
        headers.set('X-Wired-Share', token);
        init.headers = headers;
        return origFetch.call(this, input, init);
    };

    var OrigWS = window.WebSocket;
    function WrappedWS(url, protocols) {
        try {
            var u = new URL(url, location.href);
            if (!u.searchParams.get('share')) {
                u.searchParams.set('share', token);
            }
            url = u.toString();
        } catch (_) {}
        if (protocols !== undefined) {
            return new OrigWS(url, protocols);
        }
        return new OrigWS(url);
    }
    WrappedWS.prototype = OrigWS.prototype;
    WrappedWS.CONNECTING = OrigWS.CONNECTING;
    WrappedWS.OPEN = OrigWS.OPEN;
    WrappedWS.CLOSING = OrigWS.CLOSING;
    WrappedWS.CLOSED = OrigWS.CLOSED;
    window.WebSocket = WrappedWS;
})();
