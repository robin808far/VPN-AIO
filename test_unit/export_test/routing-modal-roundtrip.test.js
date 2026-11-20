"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const assert = require("node:assert/strict");

const root = path.resolve(__dirname, "../..");
const html = fs.readFileSync(path.join(root, "web/html/modals/xray_rule_modal.html"), "utf8");
const script = html.match(/<script>([\s\S]*?)<\/script>/)[1]
    .replace(/\{\{\s*i18n\s+"[^"]+"\s*\}\}/g, "test");
const context = vm.createContext({
    Vue: class { constructor() {} },
    ObjectUtil: {isEmpty: value => value == null || value === ""},
    app: {
        templateSettings: {inbounds: [], outbounds: [], routing: {balancers: []}},
        inboundTags: [], enableDNS: false, dnsTag: "",
    },
});
vm.runInContext(script + "\nglobalThis.testRuleModal = ruleModal;", context);

const original = {
    type: "field", ruleTag: "operator-rule", domain: ["domain:old.example"],
    localIP: ["127.0.0.1"], localPort: "443", process: ["/usr/bin/curl"],
    localOS: ["linux"], webhook: {url: "https://hook.example", deduplication: 60, headers: {"X-Test": "ok"}},
    user: ["alice"], outboundTag: "direct", futureSelector: {mode: "keep"},
};
context.testRuleModal.show({rule: original, isEdit: true});
context.testRuleModal.rule.domain = "domain:new.example";
const edited = JSON.parse(JSON.stringify(context.testRuleModal.getResult()));
assert.deepEqual(edited.domain, ["domain:new.example"]);
for (const key of ["ruleTag", "localIP", "localPort", "process", "localOS", "webhook", "futureSelector"]) {
    assert.deepEqual(edited[key], original[key], `routing editor dropped ${key}`);
}

// Core-era aliases are surfaced through their current controls and normalized
// when the edited rule is saved, rather than keeping a stale alias that wins.
const legacy = {type: "field", domains: ["domain:legacy.example"], source: ["10.0.0.0/8"], outboundTag: "direct"};
context.testRuleModal.show({rule: legacy, isEdit: true});
context.testRuleModal.rule.domain = "domain:edited.example";
const migrated = JSON.parse(JSON.stringify(context.testRuleModal.getResult()));
assert.deepEqual(migrated.domain, ["domain:edited.example"]);
assert.deepEqual(migrated.sourceIP, ["10.0.0.0/8"]);
assert.equal(migrated.domains, undefined);
assert.equal(migrated.source, undefined);

console.log("PASS routing-rule editor preserves unmodeled Xray fields and normalizes legacy aliases");
