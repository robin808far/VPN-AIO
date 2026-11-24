"use strict";

// Client-side mirror of service.ProjectHost for synchronous panel links and QR codes.
var ManagedHosts = class ManagedHosts {    static splitAddress(value, fallbackPort) {
        const text = (value || '').trim();
        const bracket = text.match(/^\[([^\]]+)\]:(\d+)$/);
        const simple = text.match(/^([^:]*):(\d+)$/);
        const match = bracket || simple;
        return match ? [match[1], Number(match[2]) || fallbackPort] : [text.replace(/^\[|\]$/g, ''), fallbackPort];
    }

    static randomIndex(length) {
        const values = new Uint32Array(1);
        window.crypto.getRandomValues(values);
        return Math.floor(values[0] / 4294967296 * length);
    }

    static cidrAddresses(value, requested) {
        const parts = value.split('/');
        if (parts.length !== 2) return [value];
        const ip = parts[0], prefix = Number(parts[1]), v6 = ip.includes(':');
        const width = v6 ? 128 : 32;
        if (!Number.isInteger(prefix) || prefix < 0 || prefix > width) throw new Error('Invalid CIDR');
        let bytes;
        if (v6) {
            const halves = ip.split('::');
            if (halves.length > 2) throw new Error('Invalid IPv6 CIDR');
            const left = halves[0] ? halves[0].split(':') : [];
            const right = halves.length > 1 && halves[1] ? halves[1].split(':') : [];
            const groups = halves.length === 1 ? left : [...left, ...Array(8-left.length-right.length).fill('0'), ...right];
            if (groups.length !== 8) throw new Error('Invalid IPv6 CIDR');
            bytes = groups.flatMap(g => { const n = parseInt(g,16); if (!/^[a-f\d]{1,4}$/i.test(g)) throw new Error('Invalid IPv6 CIDR'); return [n >> 8,n & 255]; });
        } else {
            bytes = ip.split('.').map(Number);
            if (bytes.length !== 4 || bytes.some(n => !Number.isInteger(n) || n < 0 || n > 255)) throw new Error('Invalid IPv4 CIDR');
        }
        let base = bytes.reduce((n,b) => n*256n+BigInt(b),0n);
        const hostBits = BigInt(width-prefix), total = 1n << hostBits;
        base = (base >> hostBits) << hostBits;
        const start = hostBits > 1n ? 1n : 0n;
        const usable = total-start-(!v6 && hostBits > 1n ? 1n : 0n);
        const count = Number(usable < BigInt(Math.min(Math.max(requested || 1,1),100)) ? usable : BigInt(Math.min(Math.max(requested || 1,1),100)));
        const result = [], seen = new Set();
        for (let attempts=0; result.length<count && attempts<count*100; attempts++) {
            const random = new Uint8Array(bytes.length);
            window.crypto.getRandomValues(random);
            const offset = random.reduce((n,b)=>n*256n+BigInt(b),0n) % usable + start;
            let number = base+offset;
            const out = Array(bytes.length);
            for (let i=out.length-1;i>=0;i--) { out[i]=Number(number & 255n); number >>= 8n; }
            const address = v6 ? Array.from({length:8},(_,i)=>((out[2*i]<<8)|out[2*i+1]).toString(16)).join(':') : out.join('.');
            if (!seen.has(address)) { seen.add(address); result.push(address); }
        }
        if (result.length !== count) throw new Error('Could not select CIDR addresses');
        return result;
    }

    static project(inbound, row, fallback) {
        const json = JSON.parse(JSON.stringify(inbound.toJson()));
        const h = {...row};
        const wildcard = !json.listen || ['0.0.0.0','::','::0'].includes(json.listen) || json.listen.startsWith('@');
        const address = h.address || (wildcard ? fallback : json.listen), port = h.port || json.port;
        json.listen = address; json.port = port;
        const stream = json.streamSettings || (json.streamSettings = {});
        if (h.security === 'tls' || h.security === 'none') stream.security = h.security;
        if (h.security === 'reality' && stream.security !== 'reality') throw new Error('Reality hosts require a Reality inbound');
        h.sni = h.keepSniBlank ? '' : h.overrideSniFromAddress ? address : h.sni || '';
        if (stream.security === 'none') { delete stream.tlsSettings; delete stream.realitySettings; }
        if (stream.security === 'tls') delete stream.realitySettings;
        if (stream.security === 'reality') delete stream.tlsSettings;
        if (stream.security === 'tls' || stream.security === 'reality') {
            const tlsKey = stream.security === 'reality' ? 'realitySettings' : 'tlsSettings';
            const tls = stream[tlsKey] || (stream[tlsKey] = {}), settings = tls.settings || (tls.settings = {});
            if (h.sni || h.keepSniBlank) { tls.serverName = h.sni; if (stream.security === 'reality') tls.serverNames = [h.sni]; }
            if (h.fingerprint) settings.fingerprint = h.fingerprint;
            if (h.alpn?.length) tls.alpn = h.alpn;
        }
        for (const key of ['wsSettings','httpupgradeSettings','xhttpSettings']) {
            const transport = stream[key];
            if (!transport) continue;
            if (h.hostHeader) transport.host = h.hostHeader;
            if (h.path) transport.path = h.path;
            if (key === 'xhttpSettings' && h.xhttpExtraParams) this.merge(transport,JSON.parse(h.xhttpExtraParams));
        }
        if (h.finalMask) {
            const masks=stream.finalmask || {},extra=JSON.parse(h.finalMask);
            for(const [key,value] of Object.entries(extra)) {
                if(['__proto__','constructor','prototype'].includes(key))continue;
                if(key==='tcp'||key==='udp')masks[key]=[...(masks[key]||[]),...value];
                else if(key!=='quicParams'||!Object.prototype.hasOwnProperty.call(masks,key))masks[key]=value;
            }
            stream.finalmask=masks;
        }
        const ep = {dest:address,port,forceTls:'same',remark:h.remark || ''};
        const uriAddress=address.includes(':')&&json.protocol!=='vmess'?'['+address.replace(/^\[|\]$/g,'')+']':address;
        stream.externalProxy = [{...ep,dest:uriAddress}];
        if (['wg-c','awg','ssh','mtproto','openvpn','l2tp','pptp','openconnect','sstp','ikev2'].includes(json.protocol)) json.settings.externalProxy = [ep];
        for (const client of json.settings.clients || []) {
            if (json.protocol === 'mtproto') client.externalProxy = [ep];
            if (json.protocol === 'vless' && h.vlessRoute) {
                const id = client.id.replace(/-/g,'');
                if (!/^[a-f\d]{32}$/i.test(id)) throw new Error('VLESS route requires a valid UUID');
                const route = Number(h.vlessRoute).toString(16).padStart(4,'0');
                const next = id.slice(0,12)+route+id.slice(16);
                client.id = [next.slice(0,8),next.slice(8,12),next.slice(12,16),next.slice(16,20),next.slice(20)].join('-');
            }
        }
        return {inbound:Inbound.fromJson(json),host:h};
    }

    static merge(base, overlay) {
        for (const [key,value] of Object.entries(overlay)) {
            if (['__proto__','constructor','prototype'].includes(key)) continue;
            base[key] = value && typeof value === 'object' && !Array.isArray(value) ? this.merge(base[key] && typeof base[key] === 'object' && !Array.isArray(base[key]) ? base[key] : {},value) : value;
        }
        return base;
    }

    static decorate(link, host) {
        if (link.startsWith('vmess://')) {
            const obj = JSON.parse(Base64.decode(link.slice(8)));
            if (obj.tls && obj.tls !== 'none') Object.assign(obj,this.tlsValues(host));
            if (host.serverDescription) obj.serverDescription=host.serverDescription;
            return 'vmess://'+Base64.encode(JSON.stringify(obj));
        }
        const u = new URL(link);
        if (['vless:','trojan:','hysteria:','hysteria2:','hy2:','anytls:','tuic:'].includes(u.protocol)) {
            if (host.security !== 'none' && u.searchParams.get('security') !== 'none') for (const [k,v] of Object.entries(this.tlsValues(host))) u.searchParams.set(k,v);
            if (host.mihomoX25519) u.searchParams.delete('support-x25519mlkem768');
            if (host.serverDescription) u.hash=decodeURIComponent(u.hash.slice(1))+'?serverDescription='+Base64.encode(host.serverDescription);
            return u.toString();
        }
        return link;
    }

    static tlsValues(h) {
        const values = {};
        if (h.sni || h.keepSniBlank) values.sni = h.sni;
        if (h.fingerprint) values.fp = h.fingerprint;
        if (h.alpn?.length) values.alpn = h.alpn.join(',');
        if (h.cipherSuites) values.cipherSuites = h.cipherSuites;
        if (h.echConfigList) values.ech = h.echConfigList;
        if (h.pinnedPeerCertSha256?.length) values.pcs = h.pinnedPeerCertSha256.join(',');
        if (h.verifyPeerCertByName) values.verifyPeerCertByName = h.verifyPeerCertByName;
        if (h.allowInsecure) values.allowInsecure = '1';
        return values;
    }

    static links(inbound, remark, remarkModel, client) {
        const rows = (inbound._managedHostRows || []).filter(h => !h.isDisabled && !(h.excludeFromSubTypes || []).includes('raw'));
        const email = client?.email || '';
        const rules = inbound._managedHostRules || {};
        const bound = Object.prototype.hasOwnProperty.call(rules,email);
        const selected = bound ? rows.filter(h => rules[email].includes(h.id)) : rows;
        if (bound && selected.length === 0) return [];
        if (!rows.length) return null;
        const results = [], shared = {}, names = new Set();
        for (const source of selected) {
            const addresses = this.cidrAddresses(source.address || '',source.randomCount);
            for (let i=0;i<addresses.length;i++) {
                const host = {...source,address:addresses[i]};
                if (addresses.length > 1) host.remark = (source.remark || '')+' - '+(i+1);
                if (host.snis?.length) {
                    if (host.sniSharedRandom) host.sni = shared[host.groupId] || (shared[host.groupId] = host.snis[this.randomIndex(host.snis.length)]);
                    else host.sni = host.snis[this.randomIndex(host.snis.length)];
                }
                const view = this.project(inbound,host,location.hostname);
                const targetClient = (view.inbound.clients || []).find(c => c.email === email) || client;
                for (const item of view.inbound.genAllLinks(remark,remarkModel,targetClient)) {
                    if (!item.link) continue;
                    item.link = this.decorate(item.link,view.host);
                    const open = String.fromCharCode(123,123), close = String.fromCharCode(125,125);
                    if ((host.remark || '').includes(open)) {
                        let title = host.remark;
                        const stats = (inbound.clientStats || []).find(s=>s.email===email) || {};
                        const total = Number(stats.total || client?.totalGB || 0), used = Number(stats.up||0)+Number(stats.down||0), expiry = Number(stats.expiryTime || client?.expiryTime || 0);
                        const traffic = bytes => {const units=['B','KB','MB','GB','TB','PB'];let n=Math.max(0,bytes),i=0;while(n>=1024&&i<units.length-1){n/=1024;i++}return n.toFixed(2)+units[i]};
                        const tokens = {INBOUND:remark,EMAIL:email,USERNAME:email,ADDRESS:view.inbound.listen,HOST:view.inbound.listen,PORT:view.inbound.port,
                            TRAFFIC_LEFT:total>0?traffic(Math.max(0,total-used)):'∞',TRAFFIC_USED:traffic(used),TRAFFIC_TOTAL:traffic(total),DAYS_LEFT:expiry>0?Math.max(0,Math.floor((expiry-Date.now())/86400000)):'∞'};
                        for (const [key,value] of Object.entries(tokens)) title = title.split(open+key+close).join(String(value));
                        item.remark = title;
                        if (item.link.startsWith('vmess://')) { const obj=JSON.parse(Base64.decode(item.link.slice(8))); obj.ps=title; item.link='vmess://'+Base64.encode(JSON.stringify(obj)); }
                        else { const u=new URL(item.link); u.hash=title+(host.serverDescription?'?serverDescription='+Base64.encode(host.serverDescription):''); item.link=u.toString(); }
                    }
                    if (names.has(item.remark)) {
                        const separator=(remarkModel || '-')[0],base=item.remark+separator+(inbound.tag || '#'+(inbound._managedInboundId||0));
                        let name=base,index=2;while(names.has(name))name=base+separator+index++;
                        item.remark=name;
                        if(item.link.startsWith('vmess://')){const obj=JSON.parse(Base64.decode(item.link.slice(8)));obj.ps=name;item.link='vmess://'+Base64.encode(JSON.stringify(obj))}
                        else {const u=new URL(item.link);if(u.protocol!=='tg:'){u.hash=name+(host.serverDescription?'?serverDescription='+Base64.encode(host.serverDescription):'');item.link=u.toString()}}
                    }
                    names.add(item.remark);
                    results.push(item);
                }
                if (results.length > 4096) throw new Error('Host expansion exceeds 4096 endpoints');
            }
        }
        return results;
    }
}
