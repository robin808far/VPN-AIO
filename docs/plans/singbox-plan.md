# sing-box as a second core in VPN-AIO

## Model (revised after studying s-ui)

The core is a **per-inbound CHOICE**, not a protocol. s-ui runs every inbound as
a sing-box object with a per-protocol form (Vless.vue, Hysteria2.vue, ...);
VortexUI and Zagros do the same split: vless/vmess/trojan/shadowsocks run under
either engine, hysteria2/tuic/anytls/shadowtls are sing-box-only. The earlier
draft that made "singbox" a protocol was wrong and has been removed.

- `model.Inbound.Core` (string, default ""): "" = Xray, "singbox" = sing-box.
- `isSingboxInbound`: sing-box-only protocols always; the Xray family by Core.
- One sing-box process per inbound: `singbox-server-<id>` under
  `/etc/vpn-ui-singbox/server-<id>`, config rendered from the RAW sing-box
  inbound object stored in the inbound's Settings, validated with
  `sing-box check` before every start.
- CRUD restarts/stops only that inbound's process; subscription links dispatch
  per protocol + Core.

## Shipped

- Catalog row `singbox` + `sing-box` daemon in the backend bundle; release
  script downloads the pinned upstream static binary (amd64/arm64).
- `web/service/singbox.go` (render/check/restart/stop/ensure/init).
- CRUD wiring in Add/Update/DelInbound; core status/restart/stop/logs.
- Frontend: Core dropdown (Xray | sing-box) in the inbound form for the shared
  protocols; sing-box-only protocols auto-select it; raw JSON editor
  (`form/singbox`) posts the block verbatim as Settings.

## Remaining

- Per-protocol structured forms for the sing-box block (like s-ui's Vless.vue
  et al.) — the raw JSON editor is the interim.
- gRPC traffic stats via sing-box's `with_v2ray_api` build; per-client accounting.
- Subscription URL presets per sing-box type in clients page.
