"use strict";
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('node:assert/strict');
const root = path.resolve(__dirname, '../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8').replace(/\r\n/g, '\n');
const timers = new Set();
const listeners = new Map();
const context = vm.createContext({
    console, crypto: require('crypto').webcrypto,
    document: { addEventListener() {} },
    location: { hostname: 'example.test', protocol: 'https:' },
    localStorage: { getItem: () => 'false', setItem() {} },
    themeSwitcher: {}, persianDatepicker: {}, ClientActionsMixin: {}, RESELLER: {},
    FormData, btoa: value => Buffer.from(value, 'latin1').toString('base64'),
    setTimeout: callback => { timers.add(callback); return callback; },
    clearTimeout: callback => timers.delete(callback),
});
context.window = context;
context.wsClient = {
    connect() {},
    on(event, callback) { if (!listeners.has(event)) listeners.set(event, new Set()); listeners.get(event).add(callback); },
    off(event, callback) { listeners.get(event)?.delete(callback); },
};
context.Vue = class {
    constructor(options) {
        this.$options = options;
        Object.defineProperties(this, Object.getOwnPropertyDescriptors(options.data || {}));
        for (const [key, fn] of Object.entries(options.methods || {})) this[key] = fn.bind(this);
        for (const [key, fn] of Object.entries(options.computed || {})) Object.defineProperty(this, key, { get: fn.bind(this) });
        this.$message = { error: message => { throw new Error(message); } };
        this.$nextTick = () => {};
    }
};
for (const file of ['web/assets/js/util/index.js', 'web/assets/js/model/inbound.js', 'web/assets/js/model/dbinbound.js']) {
    vm.runInContext(read(file), context, { filename: file });
}
function inline(file) {
    return [...read(file).matchAll(/<script>([\s\S]*?)<\/script>/g)].map(match => match[1])
        .join('\n').replace(/\{\{ if \.legacyInboundForm \}\}true\{\{ else \}\}false\{\{ end \}\}/g, 'false')
        .replace(/\{\{[\s\S]*?\}\}/g, '');
}

(async () => {
    const raw = { type: 'vless', listen: '127.0.0.1', listen_port: 34567,
        users: [{ name: 'alice', uuid: '11111111-1111-4111-8111-111111111111' }],
        transport: { type: 'ws', path: '/original' }, future_option: { keep: true } };
    context.fixture = { id: 7, core: 'singbox', protocol: 'vless', port: 34567,
        listen: '127.0.0.1', settings: JSON.stringify(raw), clientStats: [] };
    const roundtrip = vm.runInContext('new DBInbound(fixture).toInbound().toJson()', context);
    assert.equal(roundtrip.core, 'singbox');
    assert.deepEqual(JSON.parse(JSON.stringify(roundtrip.settings)), raw);
    assert.equal(vm.runInContext('new DBInbound(fixture).isMultiUser()', context), true);
    const clientView = vm.runInContext('new DBInbound(fixture).toInbound().clients[0]', context);
    assert.equal(clientView.email, 'alice');
    assert.equal(clientView.id, raw.users[0].uuid);

    const modalScript = inline('web/html/modals/inbound_modal.html');
    for (let visit = 0; visit < 3; visit++) vm.runInContext(modalScript, context);
    vm.runInContext('inModal.show({inbound: new DBInbound(fixture).toInbound(), dbInbound: new DBInbound(fixture), isEdit: true})', context);
    const modal = context.inboundModalVueInstance;
    assert.deepEqual(JSON.parse(modal.singboxDraft), raw);
    modal.singboxSet(['transport', 'path'], '/edited');
    assert.equal(context.inModal.inbound.settings.transport.path, '/edited');
    context.inModal.show({});
    assert.equal(modal.singboxDraft, '', 'a new modal retained the previous raw draft');

    const defaults = [];
    for (const protocol of ['vless', 'vmess', 'shadowsocks', 'tuic', 'hysteria2', 'anytls']) {
        context.inModal.inbound.protocol = protocol;
        context.inModal.inbound.core = 'singbox';
        modal.singboxSeed(context.inModal.inbound);
        const block = JSON.parse(modal.singboxDraft);
        defaults.push(block);
        if (protocol === 'vmess') assert.equal('flow' in block.users[0], false);
        if (protocol === 'shadowsocks') assert.equal(Buffer.from(block.password, 'base64').length, 16);
        if (['tuic', 'hysteria2', 'anytls'].includes(protocol)) {
            assert.ok(modal.inboundProblems.some(problem => /certificate/.test(problem.why)), 'incomplete TLS was presented as valid');
        } else assert.equal(modal.inboundProblems.length, 0);
        if (protocol === 'tuic') assert.match(block.users[0].uuid, /^[0-9a-f-]{36}$/);
        if (protocol === 'tuic') {
            for (const key of ['congestionControl', 'authTimeout', 'zeroRttHandshake', 'heartbeat', 'udpTimeout']) {
                assert.equal(key in block, false, `Xray TUIC key ${key} leaked into native sing-box settings`);
            }
        }
        if (protocol === 'anytls') {
            assert.equal('paddingScheme' in block, false, 'Xray AnyTLS key leaked into native sing-box settings');
        }
    }

    context.inModal.inbound.protocol = 'shadowtls';
    context.inModal.inbound.core = 'singbox';
    modal.singboxSeed(context.inModal.inbound);
    assert.equal(JSON.parse(modal.singboxDraft).version, 3);
    modal.singboxSet(['handshake', 'server'], 'example.test');
    assert.equal(modal.inboundProblems.length, 0, 'version 3 ShadowTLS form should be valid');
    modal.singboxSet(['version'], 2);
    assert.ok(modal.inboundProblems.some(problem => /managed accounts require version 3/.test(problem.why)),
        'version 2 ShadowTLS managed users were presented as valid');
    modal.singboxSet(['users'], []);
    assert.ok(modal.inboundProblems.some(problem => /managed accounts require version 3/.test(problem.why)),
        'version 2 ShadowTLS with no current users was presented as valid');
    modal.singboxSet(['version'], 3);
    assert.equal(modal.inboundProblems.length, 0, 'selecting version 3 should clear the validation error');

    const singboxForm = read('web/html/form/protocol/singbox.html');
    assert.match(singboxForm, /label="Managed-user version"[\s\S]*?singboxSet\(\['version'\], v\)[\s\S]*?a-select-option :value="3"/,
        'the graphical ShadowTLS form cannot select managed-user version 3');
    assert.match(singboxForm, /Detour target \(optional\)[\s\S]*?singboxSetDetourTarget[\s\S]*?singboxDetourTargets\(\)/,
        'the graphical ShadowTLS form cannot select a managed detour target');
    assert.match(singboxForm, /pages\.inbounds\.singboxUnmanaged[\s\S]*?:href="basePath \+ 'panel\/clients'"[\s\S]*?pages\.clients\.title/,
        'the sing-box form must direct account-policy edits to Clients');
    assert.doesNotMatch(singboxForm, /Managed accounts, subscriptions, traffic statistics[\s\S]*?unavailable/,
        'the sing-box form must not claim managed-account support is unavailable');
    assert.match(singboxForm, /Clear unsupported inbound-level limits/,
        'the sing-box form must distinguish unsupported inbound limits from account policies');
    const inboundModalTemplate = read('web/html/modals/inbound_modal.html');
    assert.match(inboundModalTemplate, /clear unsupported inbound-level limits before saving/);
    assert.doesNotMatch(inboundModalTemplate, /clear unsupported account limits before saving/);

    const membershipTemplate = read('web/html/modals/client_membership_modal.html');
    vm.runInContext(inline('web/html/modals/client_membership_modal.html'), context);
    const membershipModal = context.clientMembershipModal;
    membershipModal.assignable = [
        { inboundId: 1, core: 'singbox', protocol: 'vless' },
        { inboundId: 2, core: 'xray', protocol: 'vless' },
    ];
    membershipModal.selected = [1];
    assert.equal(membershipModal.hasSingboxSelection(), true,
        'a selected sing-box membership must expose its stream-limit control');
    membershipModal.selected = [2];
    assert.equal(membershipModal.hasSingboxSelection(), false,
        'an Xray-only VLESS account must not be treated as a sing-box selection');
    assert.match(membershipTemplate, /v-if="clientMembershipModal\.hasSingboxSelection\(\)"/,
        'the stream-limit control must call the reactive selection predicate');

    const inboundsTemplate = read('web/html/inbounds.html');
    assert.equal((inboundsTemplate.match(/pages\.inbounds\.singboxNoStats/g) || []).length, 2);
    assert.equal((inboundsTemplate.match(/href="\{\{ \.base_path \}\}panel\/clients" class="bo-il-sub"/g) || []).length, 2,
        'both desktop and mobile sing-box usage notices must link to account-level traffic');

    if (process.env.VPN_AIO_SINGBOX_DEFAULTS_FILE) fs.writeFileSync(process.env.VPN_AIO_SINGBOX_DEFAULTS_FILE, JSON.stringify(defaults));

    vm.runInContext(inline('web/html/inbounds.html'), context);
    let submitted;
    const page = context.app;
    page.dbInbounds = [
        { id: 7, core: 'singbox', protocol: 'shadowtls', tag: 'current-front' },
        { id: 8, core: 'singbox', protocol: 'shadowsocks', tag: 'managed-target' },
        { id: 9, core: 'xray', protocol: 'vless', tag: 'xray-inbound' },
    ];
    page.submit = async (url, data) => { submitted = { url, data }; };
    vm.runInContext('inModal.show({inbound: new DBInbound(fixture).toInbound(), dbInbound: new DBInbound(fixture)})', context);
    assert.equal(context.inModal.inbound.id, 7, 'editing the current inbound must preserve its id');
    const detourTargets = modal.singboxDetourTargets();
    assert.deepEqual(detourTargets.map(row => row.tag), ['managed-target'],
        'the detour selector must exclude itself and Xray-only targets');
    modal.singboxSetDetourTarget('managed-target');
    assert.equal(modal.singboxDetourTarget(), 'managed-target');
    assert.equal(context.inModal.inbound.settings.detour, 'managed-target',
        'selecting a detour target must update the raw sing-box settings');
    modal.singboxSetDetourTarget(undefined);
    assert.equal(modal.singboxDetourTarget(), undefined,
        'clearing the selector must remove the raw detour');
    modal.singboxSet(['transport', 'path'], '/submitted');
    await page.addInbound(context.inModal.inbound, context.inModal.dbInbound);
    assert.equal(submitted.data.core, 'singbox');
    assert.equal(JSON.parse(submitted.data.settings).transport.path, '/submitted');
    assert.equal('clients' in JSON.parse(submitted.data.settings), false);
    await page.updateInbound(context.inModal.inbound, context.inModal.dbInbound);
    assert.equal(JSON.parse(submitted.data.get('settings')).users[0].uuid, raw.users[0].uuid);

    page.loading = () => {};
    page.getDefaultSettings = () => {};
    page.getDBInbounds = async () => {};
    for (let visit = 0; visit < 3; visit++) {
        page.$options.mounted.call(page);
        assert.equal(listeners.get('inbounds').size, 1);
        for (const callback of listeners.get('invalidate')) callback({ type: 'inbounds' });
        page.$options.beforeDestroy.call(page);
        for (const callbacks of listeners.values()) assert.equal(callbacks.size, 0, 'old websocket callback retained');
        assert.equal(timers.size, 0, 'navigation retained an invalidation timer');
    }
    console.log('PASS sing-box model/modal/payload/defaults and page listener cleanup');
})().catch(error => { console.error(error); process.exitCode = 1; });
