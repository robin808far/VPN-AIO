/* nodes-sysbar.js - the live system status strip shown above every panel page.
 *
 * One shared Vue mixin. Only the page-root instance (el === '#app') polls, so
 * the sidebar and theme-switch components stay out of it. It reads the same
 * /panel/nodes/list the Nodes page uses, silently (raw axios, no toast), and
 * hides itself when the caller is not a super admin or the request fails.
 */
(function () {
  'use strict';
  const POLL_MS = 60000;

  async function fetchNodes() {
    try {
      const res = await axios.get(basePath + 'panel/nodes/list');
      const msg = res && res.data;
      if (!msg || !msg.success || !Array.isArray(msg.obj)) return null;
      return msg.obj;
    } catch (e) {
      return null;
    }
  }

  Vue.mixin({
    data() {
      return {
        sysbar: { visible: false, total: 0, online: 0, down: 0, inbounds: 0, lastSeen: 0 },
      };
    },
    computed: {
      sysbarLabel() {
        const i18n = window.BO_SYSBAR_I18N || {};
        if (!this.sysbar.total) return i18n.none || '';
        let label = this.sysbar.online + ' ' + (i18n.online || '');
        if (this.sysbar.down) label += ' · ' + this.sysbar.down + ' ' + (i18n.down || '');
        return label;
      },
      sysbarHeartbeat() {
        const i18n = window.BO_SYSBAR_I18N || {};
        if (!this.sysbar.lastSeen) return '';
        return (i18n.heartbeat || '') + ' ' + new Date(this.sysbar.lastSeen * 1000).toLocaleTimeString();
      },
    },
    async mounted() {
      if (this.$options.el !== '#app') return;
      // Pages without the shell (login) have no strip to feed; skip so the
      // 401 interceptor's reload is never triggered from them.
      if (!document.querySelector('.bo-sysbar')) return;
      const apply = (nodes) => {
        if (!nodes) {
          this.sysbar.visible = false;
          return;
        }
        this.sysbar.total = nodes.length;
        this.sysbar.online = nodes.filter((n) => n.status === 'online').length;
        this.sysbar.down = nodes.filter((n) => n.status === 'down').length;
        this.sysbar.inbounds = nodes.reduce((sum, n) => sum + (n.inbounds || 0), 0);
        this.sysbar.lastSeen = nodes.reduce((max, n) => Math.max(max, n.lastSeen || 0), 0);
        this.sysbar.visible = true;
      };
      apply(await fetchNodes());
      const timer = setInterval(async () => apply(await fetchNodes()), POLL_MS);
      this.$once('hook:beforeDestroy', () => clearInterval(timer));
    },
  });
})();
