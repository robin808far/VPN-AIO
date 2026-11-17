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
        assert.equal(awg.toJson().randomTrailers, true);
        assert.equal(awg.toJson().headerProtectionKey, 'abc');
    }
    console.log(`PASS ${model}: XDRIVE settings survive edits without losing S3/XMUX`);
}
