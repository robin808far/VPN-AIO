"use strict";
const assert = require('node:assert/strict');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const root = path.resolve(__dirname, '../..');
const code = fs.readFileSync(path.join(root, 'web/assets/js/softnav.js'), 'utf8');

function script(src, textContent = '', type = '') {
    return { src, textContent, getAttribute: name => name === 'src' ? src : name === 'type' ? type : null };
}
const loaded = [script('https://panel.test/assets/js/softnav.js'),
    script('https://panel.test/assets/otpauth/otpauth.umd.min.js')];
const elements = new Map();
const makeElement = id => ({ id, style: {}, classList: { add() {}, remove() {} },
    remove() { elements.delete(this.id); }, appendChild() {} });
let nextScripts = [];
let fetchedAssets = [];
let destroyed = 0;
const context = vm.createContext({
    URL, console,
    location: { href: 'https://panel.test/panel/inbounds', origin: 'https://panel.test' },
    history: { pushState() {} },
    document: {
        title: '', addEventListener() {},
        getElementById: id => elements.get(id),
        createElement: () => makeElement(''),
        querySelector: () => null,
        querySelectorAll: () => loaded,
        body: { innerHTML: '', appendChild(el) { elements.set(el.id, el); } },
    },
    DOMParser: class { parseFromString() {
        return { title: 'fixture', body: { innerHTML: '' },
            getElementById: () => ({ removeAttribute() {} }),
            querySelectorAll: () => nextScripts };
    } },
    fetch: async url => {
        if (url.includes('/assets/')) {
            fetchedAssets.push(url);
            return { ok: true, text: async () => 'window.MODEL_LOADS = (window.MODEL_LOADS || 0) + 1;' };
        }
        return { ok: true, redirected: false, text: async () => '<fixture>' };
    },
    addEventListener() {}, scrollTo() {},
});
context.window = context;
context.BO_PAGE_ROOTS = new Set([{ $destroy() { destroyed++; } }, { $destroy() { destroyed++; } }]);
vm.runInContext(code, context);
(async () => {
    nextScripts = [script('https://panel.test/assets/otpauth/otpauth.umd.min.js'),
        script('https://panel.test/assets/js/model/inbound.js'),
        script('', 'var INBOUND_SECTIONS = []; window.PAGE_LOADS = (window.PAGE_LOADS || 0) + 1;'),
        script('', 'not executable JSON', 'application/json')];
    await context.BO_SOFTNAV.navigate('https://panel.test/panel/inbounds');
    await context.BO_SOFTNAV.navigate('https://panel.test/panel/inbounds');
    assert.equal(context.PAGE_LOADS, 2);
    assert.equal(context.MODEL_LOADS, 2, 'page models must restore globals shared by inbound/outbound editors');
    assert.equal(destroyed, 4, 'all page roots, including separate modals, must be destroyed');
    assert.ok(fetchedAssets.every(url => !url.includes('/otpauth/')), 'initial vendor asset was evaluated again');
    console.log('PASS soft navigation initial vendor deduplication, page scripts and complete root teardown');
})().catch(error => { console.error(error); process.exitCode = 1; });
