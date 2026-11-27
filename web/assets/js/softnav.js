// Soft navigation for the panel shell.
//
// Clicking a panel link used to hand the whole document to location.href, so
// the browser tore down the shell (sidebar, topbar), painted its default white
// page and rebuilt everything from scratch. The shell markup of every panel
// page is identical, so instead we fetch the target page (the sidebar already
// prefetches every tab, and the responses carry a short private max-age), swap
// the body in place and re-run the page's own scripts. The sidebar and topbar
// never visibly leave, and while the fetch is in flight a spinner sits over
// the content area instead of a white page.
(function () {
  'use strict';

  const PANEL_RE = /\/panel(\/|$)/;

  function isPanelUrl(url) {
    if (!url) return false;
    const abs = new URL(url, location.href);
    if (abs.origin !== location.origin) return false;
    if (!PANEL_RE.test(abs.pathname)) return false;
    if (/logout/.test(abs.pathname)) return false;
    return true;
  }

  let navigating = false;

  function ensureLoader() {
    let loader = document.getElementById('bo-page-loader');
    if (loader) return loader;
    loader = document.createElement('div');
    loader.id = 'bo-page-loader';
    loader.innerHTML = '<span class="bo-page-loader-spin"></span>';
    document.body.appendChild(loader);
    return loader;
  }

  function showLoader() {
    ensureLoader().classList.add('is-visible');
  }

  function hideLoader() {
    const loader = document.getElementById('bo-page-loader');
    if (loader) loader.classList.remove('is-visible');
  }

  // Vendor libraries run once, including OTPAuth and CodeMirror. Page models
  // run on every visit: inbound/outbound models share global class names and
  // must restore the owning page's classes after visiting the other editor.
  // Their global declarations are var in the sources, including on initial load;
  // rewriting a later script cannot undo an existing global lexical binding.
  const loadedScripts = new Set();
  const SHARED_ASSET_RE = /\/assets\/(?!js\/)|\/assets\/js\/(axios-init|nodes-sysbar|websocket|softnav)\.js/;

  function runInline(code, where) {
    try {
      (0, eval)(code);
    } catch (e) {
      console.error('softnav: script error in', where, e);
      throw e; // navigate falls back to a complete, working document.
    }
  }

  // Re-run a fetched page's scripts. Page-level external scripts are fetched
  // and appended as classic scripts; inline scripts are run in document
  // order via runInline, which tolerates a harmless redeclaration of
  // top-level const/let/class bindings. The shell's `const basePath`
  // bootstrap is skipped — its value never changes between pages.
  async function runScripts(doc) {
    let n = 0;
    for (const old of doc.querySelectorAll('script')) {
      const type = old.getAttribute('type');
      if (type && !/^(text|application)\/javascript$/i.test(type)) continue;
      const src = old.getAttribute('src');
      if (src) {
        const abs = new URL(src, location.href).href;
        if (SHARED_ASSET_RE.test(abs) && loadedScripts.has(abs)) continue;
        const res = await fetch(abs, { credentials: 'same-origin' });
        if (!res.ok) throw new Error('HTTP ' + res.status + ' loading ' + abs);
        runInline(await res.text(), abs);
        loadedScripts.add(abs);
        continue;
      }
      const code = old.textContent || '';
      if (!code.trim()) continue;
      n += 1;
      runInline(code, 'inline#' + n);
    }
  }

  async function navigate(url, push) {
    if (navigating) return;
    navigating = true;
    showLoader();
    try {
      const res = await fetch(url, { credentials: 'same-origin' });
      const finalUrl = res.redirected ? res.url : url;
      const text = await res.text();
      const doc = new DOMParser().parseFromString(text, 'text/html');
      const nextApp = doc.getElementById('app');
      if (!nextApp || !res.ok) {
        location.href = url;
        return;
      }
      // Remember the initial document's assets before the body swap. Destroy
      // every page root, including separately mounted/portaled modal instances.
      for (const script of document.querySelectorAll('script[src]')) loadedScripts.add(script.src);
      for (const root of Array.from(window.BO_PAGE_ROOTS || [])) root.$destroy();

      // Freeze the current shell chrome (side rail + top bar) as fixed clones
      // BEFORE the body swap. The new page's chrome only comes back when its
      // Vue app mounts (after the scripts run); without the clones that gap
      // shows a mostly blank page — the flash on some navigations. The clones
      // render pixel-identically (same markup, same CSS) and are removed once
      // the real chrome is mounted, so the handover is invisible.
      const rail = document.querySelector('.bo-rail');
      const topbar = document.querySelector('.bo-topbar');
      const railWidth = rail ? rail.getBoundingClientRect().width : 0;
      const topbarHeight = topbar ? topbar.getBoundingClientRect().height : 0;
      const railClone = rail ? rail.cloneNode(true) : null;
      const topbarClone = topbar ? topbar.cloneNode(true) : null;
      const mountGhosts = () => {
        if (railClone) {
          const g = document.createElement('div');
          g.id = 'bo-rail-ghost';
          g.style.width = railWidth + 'px';
          g.appendChild(railClone.cloneNode(true));
          document.body.appendChild(g);
        }
        if (topbarClone) {
          const g = document.createElement('div');
          g.id = 'bo-topbar-ghost';
          g.style.height = topbarHeight + 'px';
          g.style.insetInlineStart = railWidth + 'px';
          g.appendChild(topbarClone.cloneNode(true));
          document.body.appendChild(g);
        }
      };
      mountGhosts();

      if (push !== false) history.pushState({}, '', finalUrl);
      document.title = doc.title;

      // The fetched #app carries v-cloak; the boot splash must not take over
      // an already-running shell, so drop it and swap immediately.
      nextApp.removeAttribute('v-cloak');
      document.body.innerHTML = doc.body.innerHTML;
      // The body swap just wiped the loader and the ghost clones. Recreate
      // them in the SAME synchronous task (no paint can happen in between) so
      // the frozen chrome and the spinner keep covering the raw, unmounted
      // markup while the page's scripts (external fetches included) finish
      // running; they are removed in the finally below.
      showLoader();
      mountGhosts();

      await runScripts(doc);
      window.scrollTo(0, 0);
    } catch (e) {
      console.error('softnav:', e);
      location.href = url;
    } finally {
      // The real chrome is mounted by now; drop the frozen clones so hover
      // and the new page's active-item highlight come back.
      for (const id of ['bo-rail-ghost', 'bo-topbar-ghost']) {
        const ghost = document.getElementById(id);
        if (ghost) ghost.remove();
      }
      hideLoader();
      navigating = false;
    }
  }

  document.addEventListener('click', (ev) => {
    if (ev.defaultPrevented || ev.button !== 0 || ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey) return;
    const a = ev.target.closest && ev.target.closest('a[href]');
    if (!a || a.target || a.hasAttribute('download')) return;
    const href = a.getAttribute('href');
    if (!href || href.startsWith('#')) return;
    if (!isPanelUrl(href)) return;
    ev.preventDefault();
    navigate(a.href, true);
  });

  window.addEventListener('popstate', () => {
    if (isPanelUrl(location.href)) navigate(location.href, false);
  });

  window.BO_SOFTNAV = { navigate, isPanelUrl };
})();
