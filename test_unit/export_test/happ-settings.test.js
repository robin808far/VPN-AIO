"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const assert = require("node:assert/strict");

const root = path.resolve(__dirname, "../..");
const html = fs.readFileSync(path.join(root, "web/html/settings.html"), "utf8").replace(/\r\n/g, "\n");
const start = html.indexOf("<script>\n  var app = new Vue");
assert.ok(start >= 0, "settings Vue script not found");
const script = html.slice(start + "<script>".length, html.indexOf("</script>", start))
    .replace(/\{\{\s*i18n\s+"[^"]+"\s*\}\}/g, "test")
    .replace(/\{\{\s*\.two_factor\s*\}\}/g, "false");
const context = vm.createContext({
    Vue: class { constructor(options) { context.settingsOptions = options; } },
    AllSetting: class {},
    themeSwitcher: {}, MediaQueryMixin: {}, SectionNavMixin: {},
    LanguageManager: { getLanguage: () => "en-US" },
    btoa: value => Buffer.from(value, "latin1").toString("base64"),
});
vm.runInContext(script, context);
const apply = context.settingsOptions.methods.applyHappRoutePreset;

function applyPreset(preset, adblock = false, routingEnabled = true) {
    const state = { happRoutePreset: preset, happIncludeAdblock: adblock, allSetting: { subEnableRouting: routingEnabled } };
    apply.call(state);
    return state.allSetting;
}

const off = applyPreset("off");
assert.equal(off.subRoutingRules, "happ://routing/off");
assert.equal(off.subEnableRouting, false, "Disable-routing preset left the routing switch enabled");

const iran = applyPreset("iran-bypass", true, false);
assert.equal(iran.subEnableRouting, true, "Active preset failed to enable routing");
assert.ok(iran.subRoutingRules.startsWith("happ://routing/onadd/"));
const profile = JSON.parse(Buffer.from(iran.subRoutingRules.slice("happ://routing/onadd/".length), "base64").toString("utf8"));
assert.equal(profile.Name, "Iran Bypass");
assert.deepEqual(profile.DirectSites, ["geosite:private", "domain:ir", "geosite:category-ir"]);
assert.deepEqual(profile.BlockSites, ["geosite:category-ads-all"]);

const global = applyPreset("global");
const globalProfile = JSON.parse(Buffer.from(global.subRoutingRules.slice("happ://routing/onadd/".length), "base64").toString("utf8"));
assert.deepEqual(globalProfile.BlockSites, [], "Ad blocking must be opt-in");
assert.deepEqual(globalProfile.DirectIp, []);
console.log("PASS Happ settings presets generate routing profiles and synchronize the routing switch");
