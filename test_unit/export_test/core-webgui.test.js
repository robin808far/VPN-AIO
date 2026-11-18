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
    }
    console.log(`PASS ${model}: XDRIVE settings survive edits without losing S3/XMUX`);
}
