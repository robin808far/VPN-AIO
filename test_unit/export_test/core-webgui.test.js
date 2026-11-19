"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const assert = require("node:assert/strict");
const root = path.resolve(__dirname, "../..");
for (const model of ["inbound", "outbound"]) {
    const context = vm.createContext({
        console,
        window: {location: {hostname: "localhost"}, crypto: require("crypto").webcrypto},
        document: {addEventListener() {}},
    });
    const source = fs.readFileSync(path.join(root, "web/assets/js/util/index.js"), "utf8") + "\n" +
        fs.readFileSync(path.join(root, `web/assets/js/model/${model}.js`), "utf8");
    vm.runInContext(source + "\nglobalThis.Stream = StreamSettings;", context);
    const input = {network: "xdrive", xdriveSettings: {
        service: "s3", remoteFolder: "folder", secrets: ["access", "secret"],
        s3: {endpoint: "https://s3.example", bucket: "vpn", region: "eu", pathStyle: false},
        segmentBytes: 65536, pollIntervalMs: 10, xmux: {maxConnections: 1},
    }};
    const stream = context.Stream.fromJson(input);
    const quicInput = {
        congestion: 'bbr', bbrProfile: 'aggressive',
        initStreamReceiveWindow: 32768, maxStreamReceiveWindow: 65536,
        initConnectionReceiveWindow: 131072, maxConnectionReceiveWindow: 262144,
        maxIdleTimeout: 45, keepAlivePeriod: 10, maxIncomingStreams: 32,
        disablePathMTUDiscovery: true, disableChromeParrot: true,
        disableGSO: true, disableStatelessReset: true,
        brutalDisableLossCompensation: true,
    };
    const quicStream = context.Stream.fromJson({network: 'xhttp', finalmask: {quicParams: quicInput}});
    for (const [key, value] of Object.entries(quicInput)) {
        assert.equal(quicStream.toJson().finalmask.quicParams[key], value, `${model} lost QUIC ${key}`);
    }
    quicStream.finalmask.quicParams.disableGSO = false;
    assert.notEqual(quicStream.toJson().finalmask.quicParams.disableGSO, true);
    stream.xdrive.pollIntervalMs = 20;
    const expected = structuredClone(input.xdriveSettings);
    expected.pollIntervalMs = 20;
    assert.deepEqual(JSON.parse(JSON.stringify(stream.toJson().xdriveSettings)), expected);
    assert.equal(context.Stream.fromJson({network: "tcp"}).toJson().xdriveSettings, undefined);
    const fresh = context.Stream.fromJson({network: "xdrive"});
    assert.equal(fresh.xdrive.s3.pathStyle, true);
    fresh.xdrive.service = "s3";
    fresh.xdrive.s3.bucket = "new-bucket";
    assert.equal(fresh.toJson().xdriveSettings.s3.bucket, "new-bucket");
    if (model === "inbound") {
        vm.runInContext("globalThis.Awg = Inbound.AwgSettings;", context);
        const awg = context.Awg.fromJson({s3: 12, s4: 24, randomTrailers:true, headerProtectionKey:'abc', clients: []});
        assert.equal(awg.toJson().s3, 12);
        assert.equal(awg.toJson().s4, 24);
        awg.awg31.randomTrailers=true; awg.awg31.headerProtectionKey='abc';
        assert.equal(awg.toJson().awg31.randomTrailers, true);
        assert.equal(awg.toJson().awg31.headerProtectionKey, 'abc');
        vm.runInContext("globalThis.Masque = Inbound.MasqueSettings;", context);
        const masque = context.Masque.fromJson({address:['198.18.70.1/24'], mtu:1280, clients:[{email:'alice',password:'secret',enable:true}]});
        assert.deepEqual(JSON.parse(JSON.stringify(masque.toJson().address)), ['198.18.70.1/24']);
        assert.equal(masque.toJson().clients[0].password, 'secret');
        const newMasque = new context.Masque('masque');
        assert.deepEqual(JSON.parse(JSON.stringify(newMasque.toJson().address)), ['10.14.0.1/24', 'fd14::1/64']);
        vm.runInContext('globalThis.MasqueInbound = Inbound;', context);
        const inbound = context.MasqueInbound.fromJson({protocol:'masque', settings:{address:['10.14.0.1/24'],clients:[]}, streamSettings:{network:'masque',security:'tls',masqueSettings:{path:'/connect'},tlsSettings:{alpn:['h2']}}});
        const restored = JSON.parse(JSON.stringify(inbound.toJson()));
        assert.equal(restored.streamSettings.network, 'masque');
        assert.equal(restored.streamSettings.masqueSettings.path, '/connect');
        assert.deepEqual(restored.streamSettings.tlsSettings.alpn, ['h2']);
        const maskInbound = context.MasqueInbound.fromJson({protocol:'hysteria',settings:{clients:[]},streamSettings:{network:'hysteria',security:'tls',hysteriaSettings:{version:2},tlsSettings:{alpn:['h3']},finalmask:{tcp:[{type:'fragment',settings:{length:'100-200',delay:'5-10'}},{type:'header-custom',settings:{clients:[],servers:[],errors:[[{type:'str',packet:'fallback'}]]}}],udp:[{type:'mkcp-aes128gcm',settings:{password:'legacy-key'}},{type:'xdns',settings:{domains:['example.test:a'],resolvers:['example.test:a+udp://1.1.1.1:53']}}]}}});
        const maskJson = JSON.parse(JSON.stringify(maskInbound.toJson().streamSettings.finalmask));
        assert.deepEqual(maskJson.tcp[0].settings.lengths, ['100-200']);
        assert.deepEqual(maskJson.tcp[0].settings.delays, ['5-10']);
        assert.equal(maskJson.tcp[1].settings.errors[0][0].packet, 'fallback');
        assert.equal(maskJson.udp[0].type, 'mkcp-legacy');
        assert.equal(maskJson.udp[0].settings.value, 'legacy-key');
        assert.deepEqual(maskJson.udp[1].settings.domains[0].types, [1]);
        assert.deepEqual(maskJson.udp[1].settings.resolvers[0].addrs, ['udp://1.1.1.1:53']);
    }
    if (model === "outbound") {
        vm.runInContext("globalThis.Out = Outbound; globalThis.P = Protocols;", context);
        const out = new context.Out();
        out.protocol = context.P.Masque;
        out.settings.address = 'proxy.example';
        out.stream.masque.user = 'alice';
        out.stream.masque.pass = 'test-password';
        const serialized = JSON.parse(JSON.stringify(out.toJson()));
        assert.equal(serialized.streamSettings.network, 'masque');
        assert.equal(serialized.streamSettings.security, 'tls');
        assert.equal(serialized.streamSettings.masqueSettings.user, 'alice');
        assert.equal(serialized.settings.address, 'proxy.example');
        const restored = context.Out.fromJson(serialized);
        assert.equal(restored.stream.masque.pass, 'test-password');
        assert.equal(restored.canEnableTls(), true);
        assert.equal(restored.hasAddressPort(), true);
        const warp = context.Out.fromJson(serialized);
        warp.stream.masque.user = 'must-not-be-sent';
        warp.stream.masque.pass = 'must-not-be-sent';
        warp.stream.setMasqueWarp(true);
        warp.stream.masque.warp.privateKey = 'warp-private';
        const warpJson = JSON.parse(JSON.stringify(warp.toJson()));
        assert.equal(warpJson.streamSettings.masqueSettings.user, undefined);
        assert.equal(warpJson.streamSettings.masqueSettings.pass, undefined);
        assert.equal(warpJson.streamSettings.masqueSettings.warp.privateKey, 'warp-private');
        assert.equal(warpJson.mux, undefined, 'generic Xray Mux must not wrap MASQUE');
        const warpRestored = context.Out.fromJson(warpJson);
        assert.equal(warpRestored.stream.masque.warp.privateKey, 'warp-private');
        assert.equal(warpRestored.stream.masque.user, '');
        const badLegacyMux = {...serialized, mux: {enabled:true, concurrency:8}};
        assert.equal(context.Out.fromJson(badLegacyMux).toJson().mux, undefined);
        const masks = context.Stream.fromJson({network:'hysteria',finalmask:{tcp:[{type:'xmc',settings:{hostname:'mc.example',profiles:[{username:'Alex',uuid:'069a79f4-44e9-4726-a5be-fca90e38aaf5',texturesValue:'v',texturesSignature:'s'}],password:'secret'}},{type:'header-custom',settings:{clients:[],servers:[],errors:[[{type:'str',packet:'fallback'}]]}}],udp:[{type:'mkcp-original'},{type:'realm',settings:{url:'realm://token@host:443/id',stunServers:['stun.example:3478']}},{type:'udphop',settings:{mode:'intervalRemote',interval:'5-10',remoteIPs:['192.0.2.1'],remotePorts:'20000-30000'}}]}});
        const maskJson = JSON.parse(JSON.stringify(masks.toJson().finalmask));
        assert.equal(maskJson.tcp[0].settings.profiles[0].texturesSignature, 's');
        assert.equal(maskJson.tcp[1].settings.errors[0][0].packet, 'fallback');
        assert.equal(maskJson.udp[0].type, 'mkcp-legacy');
        assert.equal(maskJson.udp[1].settings.portMapping, undefined);
        assert.equal(maskJson.udp[2].settings.remotePorts, '20000-30000');
        const scripted = context.Stream.fromJson({network:'hysteria',finalmask:{udp:[{type:'noise',settings:{script:'return "noise"',noise:[{rand:'1-3',type:'array',packet:[1]}]}}]}});
        assert.equal(JSON.parse(JSON.stringify(scripted.toJson().finalmask.udp[0].settings)).noise, undefined);
    }
    console.log(`PASS ${model}: XDRIVE settings survive edits without losing S3/XMUX`);
}
