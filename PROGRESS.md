# Project Progress & State

آخرین به‌روزرسانی: 2026-11-28 (ذخیرهٔ وضعیت؛ ثبت کارهای محلی sing-box بعد از v7.0.2 و هم‌ترازی اسناد وضعیت)

Project root: `C:\Users\mef\Documents\OpenCode Projects\VPN-AIO`

## وضعیت فعلی (Current Status)

- **شاخه/نسخهٔ محلی (مشاهدهٔ تازه):** `main` روی `b010e8be`، **۴ کامیت جلوتر از `origin/main`** (`884ad937`, `6607000c`, `951ab31b`, `b010e8be`)، working tree تمیز (commit شد؛ هنوز push نشده). `config/version` = `7.0.2`.
- **تگ‌های منتشرشده (مشاهدهٔ محلی/ریموت):** `v7.0.0`, `v7.0.1`, `v7.0.2` وجود دارند؛ تگ ریموت `v7.0.2` دیده شد (`refs/tags/v7.0.2`). در این ذخیرهٔ وضعیت، CI انتشار/اعتبارسنجی v7.0.2 دوباره بررسی نشده؛ PROGRESS قبلی ثبت کرده بود که CI v7.0.2 در زمان ذخیرهٔ قبلی هنوز در حال اجرا بوده است.
- **کار محلی بعد از v7.0.2 (کامیت‌شده، push نشده):** سه قابلیت sing-box اضافه شده‌اند: Snell inbound (`884ad937`)، فیلدهای مشترک Listen (`6607000c`)، فرم Cloudflared inbound (`951ab31b`). این‌ها منتشر/اعتبارسنجی runtime نشده‌اند.
- **تغییرات uncommitted محلی:** `PROGRESS.md`, `HANDOFF.md`, `docs/README.md`, `web/html/core.html` تغییر کرده‌اند؛ `docs/plans/tunnel-repositories-review-20261128.md` untracked است. در `web/html/core.html` دکمهٔ قالب OpenVPN Server endpoint (sing-box 1.14.0+) اضافه شده که یک قالب شروع را در `endpoints[]` درج می‌کند؛ این endpoint-editor همچنان کامل نیست (بدون فرم گرافیکی کامل برای tls/ciphers/push options)، اما باگ پسورد placeholder رفع شد — جزئیات در بخش «کارهای انجام‌شده».
- **میزبان زنده:** طبق شواهد قبلی، میزبان `sweden` (پورت 12301) قبلاً روی 6.6.0 دیده شده و بعداً کاربر 7.0.0 را دیپلوی کرده بود؛ نقص bundle قدیمی sing-box در 7.0.0 گزارش/رفع شد و v7.0.1/v7.0.2 برای آن منتشر شدند. در این ذخیرهٔ وضعیت میزبان زنده بررسی یا دستکاری نشده است.
- **درخواست‌های باز کاربر (از زمینهٔ قبلی):** ۱) rename کامل `vpn-ui` -> `vpn-aio` با مهاجرت خودکار (در 7.0.1 انجام شد؛ نقص مهاجرت لاگ/برخورد در 7.0.2 رفع شد)؛ ۲) انتشار 7.0.1/7.0.2 و بررسی هستهٔ تعبیه‌شده؛ ۳) پس از به‌روزرسانی میزبان زنده به 7.0.2 توسط کاربر، تست‌های sing-box؛ ۴) گزینه‌های باقی‌ماندهٔ بخش ۳ sing-box (Snell/Listen/cloudflared تا حدی در کامیت‌های محلی؛ OpenVPN server کامل، Hysteria2 Realm و موارد دیگر هنوز باز). این ذخیرهٔ وضعیت کار اجرایی جدیدی انجام نمی‌دهد.

## کارهای انجام‌شده (Completed)

### انتشار/رفع نقص — v7.0.1 / v7.0.2 (شواهد قبلی این نشست)

- **v7.0.1** (`7d688912`): rename کامل `vpn-ui` -> `vpn-aio` با مهاجرت خودکار (`web/service/legacy_migrate.go`, `config/legacy.go`)؛ رفع cache bundle قدیمی sing-box با بررسی نسخه/marker `vpn-aio-limits` و حذف `restore-keys` در workflow انتشار. CI انتشار و اعتبارسنجی Ubuntu طبق PROGRESS قبلی موفق بودند. بررسی روی `sweden`: `SHA256SUMS` برای amd64/arm64 OK؛ هستهٔ sing-box تعبیه‌شده `1.14.5` با تگ `with_v2ray_api` و marker `vpn-aio-limits`.
- **نقص مشاهده‌شده در 7.0.1:** مهاجرت rename در حالت برخورد مسیر جدید، مسیر legacy را باقی می‌گذاشت (`/var/log/VPN-AIO/vpn-ui.log`, `/etc/vpn-ui-singbox` کنار `/etc/vpn-aio-singbox`). در کانتینر یک‌بارمصرف `vpn-aio-migrate-test` بازتولید شد.
- **v7.0.2** (`9b4378e8`): در برخورد، مسیر legacy به `<new-name>.legacy-backup` (یا `-2`, `-3`, ...) تغییر نام می‌دهد؛ `MigrateLegacyLogFile()` قبل از `initLogger()` لاگ قدیمی `vpn-ui.log` را به `vpn-aio.log` ادغام می‌کند. تست‌های Go اضافه شد (`TestMigrateLegacyIn`, `TestMigrateLegacyInBackupNameIsUnique`) و در کانتینر یک‌بارمصرف تأیید شد؛ `find / -xdev -iname '*vpn-ui*'` خالی بود.
- **رفع عملیاتی روی میزبان زنده (بدون تغییر کد محصول):** پنل دستی باقی‌ماندهٔ دستیار در `/home/ubuntu/v701-extract` پورت Xray API `127.0.0.1:21111` را گرفته بود؛ فقط آن پروسه متوقف و دایرکتوری حذف شد؛ `VPN-AIO.service` دست‌نخورده ماند و بعداً Xray بدون خطای bind شروع شد.
- **پاکسازی موقت:** کانتینرهای `vpn-aio-ui-check-20261128`, `vpn-aio-migrate-test` و فایل‌های scratch مرتبط در `sweden` حذف شدند. کانتینرهای pre-existing audit/review دست‌نخورده باقی گذاشته شدند.

### کار محلی sing-box بعد از v7.0.2 (کامیت‌شده، push نشده؛ مشاهدهٔ git)

- `884ad937 feat(singbox): add Snell inbound (sing-box 1.14.0+), managed per-account users` — فایل‌ها: `database/model/model.go`, `web/html/form/protocol/singbox.html`, `web/html/modals/inbound_modal.html`, `web/service/singbox_accounts.go`.
- `6607000c feat(singbox): expose shared Listen Fields (keep-alive, TFO, multipath, UDP) in the inbound form` — فایل: `web/html/form/protocol/singbox.html`.
- `951ab31b feat(singbox): add Cloudflared inbound form (admin-config, no managed users)` — فایل‌ها: `web/html/form/protocol/singbox.html`, `web/html/modals/inbound_modal.html`.
- `def29152 ci: align build-cache key with the release key so tag runs reuse the prebuilt arm64 bundle` — در `origin/main..v7.0.2?` نه؛ در `v7.0.2..HEAD` هست و روی `origin/main` هم هست (چون شاخه ۳ کامیت جلوتر از origin/main است).

### کار محلی uncommitted (مشاهدهٔ تازه)

- `web/html/core.html`: دکمهٔ «OpenVPN Server» در بخش Endpoints ویرایشگر base-config اضافه شده؛ با کلیک، قالب endpoint `openvpn-server` را به `endpoints[]` JSON اضافه و تب JSON را باز می‌کند. کامنت کد تأکید می‌کند endpoint یک inbound نیست، users/tls/ciphers/push options باید در JSON تکمیل شوند و endpointهای موجود حفظ می‌شوند.
- **رفع شده (این نشست):** مقدار پسورد placeholder در `addOpenvpnServerEndpoint()` (خط ~1043) که رشتهٔ ثابت بود، به `RandomUtil.randomSeq(...)` تغییر کرد تا هر بار کلیک دکمه یک پسورد تصادفی امن (مطابق الگوی موجود در `client.html`/`inbound_modal.html`/`client_bulk_modal.html`) تولید شود.
- **تست runtime (این نشست، کامل شده):** چون `go run` روی این ماشین ویندوزی بدون gcc/cgo ممکن نبود، منطق تابع با یک صفحهٔ HTML موقت (Vue 2.7.16 از CDN + `RandomUtil` واقعی از `web/assets/js/util/index.js`) در مرورگر داخلی تست شد — متد عیناً و برنامهنویسیشده از خود `core.html` استخراج شد تا هیچ تایپ دستی رخ ندهد. نتیجه: سه بار کلیک پشت سر هم → ۳ endpoint با tagهای `openvpn-server-1/2/3` و شبکههای غیرهمپوشان `10.9.0.0/24`, `10.10.0.0/24`, `10.11.0.0/24`؛ هر پسورد ۱۶ کاراکتر، غیرخالی و کاملاً یکتا بود؛ JSON خروجی هر بار معتبر پارس شد و هیچ خطای کنسول ثبت نشد. فایلهای تست موقت (`_tmp_*`) بعد از تست حذف شدند — چیزی در ریپازیتوری باقی نماند.
- `docs/plans/tunnel-repositories-review-20261128.md`: گزارش تحقیق هفت مخزن تانل (untracked).
- `docs/README.md`, `HANDOFF.md`, `PROGRESS.md`: به‌روزرسانی اسناد وضعیت/مرجع.

### Hysteria Realm — اضافه شد (این نشست، کامل، تستشده)

- سرویس رندهووی NAT traversal برای Hysteria2 (sing-box 1.14+، `services[]` -> `hysteria-realm`) با فرم گرافیکی کامل (نه فقط قالب+JSON، چون فیلدها کم و پایدارند).
- `core.html`: `baseForm` فیلدهای `hysteriaRealm*` گرفت؛ `jsonToForm` اولین سرویس `hysteria-realm` را از `cfg.services` میخواند؛ `formToJson` سرویس را از فرم بازمیسازد و بقیهٔ `services[]` دستنخورده حفظ میشود (کلید `services` کنار `outbounds/endpoints/ntp` به لیست کلیدهای حفظشده اضافه شد). UI جدید بعد از بخش Endpoints: سوییچ فعال/غیرفعال، tag، listen port، TLS اختیاری، لیست users (name/token/maxRealms) با v-for مثل routeRules.
- `web/html/form/protocol/singbox.html`: روی inbound Hysteria2 (طرف سرور پشت NAT)، بعد از Masquerade، سوییچ «Behind NAT (use Realm)» و فیلدهای `realm.server_url/token/realm_id/stun_servers` اضافه شد.
- تست: سینتکس هر دو تابع با استخراج برنامهنویسیشده و `require` (node) تأیید شد. تگهای HTML بخش realm در singbox.html با شمارش باز/بسته متوازن تأیید شدند. رفتار runtime با shim مرورگری (Vue 2.7.16 + توابع واقعی استخراجشده) تست شد: بارگذاری فیکسچر با سرویس realm موجود پر شدن درست فرم را نشان داد؛ افزودن کاربر دوم و ذخیره، JSON خروجی معتبر با هر دو کاربر و توکنهای غیرخالی تولید کرد؛ بدون خطای کنسول. فایلهای تست موقت حذف شدند.
- فیلد realm روی inbound Hysteria2 فقط از نظر ساختار HTML بررسی شد؛ رفتار runtime آن در این نشست تست نشد.

### تحقیق مخازن تانل (2026-11-28، فقط تحقیق — کامل شده)

- گزارش پایدار: `docs/plans/tunnel-repositories-review-20261128.md`؛ لینک در `docs/README.md`.
- مخازن: BackPack, WatTunnel, Kariz, C-TUNNEL, mahdiweb3/tuntunnel, Overal-Tun, Arange-tun. snapshotهای main و SHAs در گزارش ثبت شده‌اند.
- محدودیت‌های مهم مجوز/استفاده: Kariz مجوز فعلی استفاده/ارکستراسیون از پنل دیگر را صراحتاً منع می‌کند؛ TunTunnel بدون مجوز؛ Overal فقط badge MIT بدون متن grant پیدا شد؛ BackPack/C-Tunnel/Arange AGPL؛ WatTunnel MIT.
- هیچ نصب/بنچمارک/تست runtime اجرا نشد؛ نتیجه‌های CI upstream به خود upstream نسبت داده شده‌اند.

## قدم‌های بعدی (Next Steps)

- [ ] **تعیین تکلیف انتشار کامیت‌های محلی sing-box:** ۳ کامیت جلوتر از `origin/main` (`884ad937`, `6607000c`, `951ab31b`) و تغییر uncommitted `web/html/core.html` هنوز push/release نشده‌اند؛ قبل از ادعای انتشار، تصمیم کاربر دربارهٔ commit/push/release لازم است.
- [x] **بررسی/تکمیل OpenVPN Server endpoint — بسته شد (تصمیم کاربر):** رفتار placeholder پسورد رفع و با تست runtime جداگانه (shim HTML در مرورگر) تأیید شد. دربارهٔ ساخت فرم گرافیکی کامل (tls/ciphers/push options) از کاربر پرسیده شد؛ شواهد نشان داد پنل از قبل یک پروتکل OpenVPN **native و کامل** دارد (`form/openvpn`, `form/openvpn/security`, `form/openvpn/network` در `inbound.html` — مستقل از sing-box). تصمیم کاربر: فرم گرافیکی کامل برای OpenVPN endpoint در sing-box **ساخته نشود**؛ همان دکمهٔ «قالب + ویرایش دستی در JSON» فعلی کافی است، چون این endpoint فقط برای سناریوهای پیشرفتهٔ chaining در sing-box کاربرد دارد، نه مسیر اصلی OpenVPN. باقیمانده فقط: تست با سرویس واقعی گو روی build منتشرشده (این محیط ویندوزی gcc/cgo ندارد؛ تست روی لینوکس/WSL یا بعد از نصب toolchain لازم است) — این یک آیتم عمومی است، نه مانع این مورد خاص.
- [ ] **تست کامل runtime sing-box روی build منتشرشده:** پس از انتشار/دیپلوی نسخهٔ مربوط توسط کاربر، تست تولید کانفیگ، ترافیک واقعی، آمار/سهمیه، محدودیت‌ها و chaining را در namespace خصوصی اجرا کن (بدون دست زدن به پورت 12301 میزبان زنده).
- [ ] **گزینه‌های باقی‌ماندهٔ بخش ۳:** Snell/Listen/cloudflared تا حدی اضافه شده‌اند اما نیاز به بررسی کامل UI/مدل/تست دارند؛ Hysteria2 Realm و sing-box OpenVPN server (فراتر از قالب endpoint) همچنان باز هستند.
- [ ] **تصمیم کاربر برای میزبان زنده:** ارتقا به 7.0.2 یا نسخهٔ بعد، و بررسی وضعیت کانتینرهای pre-existing audit/review روی `sweden` فقط با درخواست صریح کاربر.

## تصمیم‌ها و محدودیت‌ها (Decisions & Constraints)

- ذخیرهٔ وضعیت است؛ در این کار تغییر اجرایی جدید روی محصول/سرویس بیرونی انجام نشده است.
- هستهٔ sing-box در پنل «قفل» است (آپدیت از سورس در پنل ندارد) چون آرشیو upstream بدون patch `vpn-aio-limits` است؛ هسته با ریلیز پنل عرضه می‌شود.
- مدل محدودیت‌ها: سیاست در `accounts`/`client_traffics`؛ `users` بومی sing-box فقط credential؛ سرعت/IP از طریق sidecar مشترک `speedlimits.json` که هستهٔ patched مصرف می‌کند؛ شمارش دستگاه فیزیکی رد می‌شود؛ `StreamLimit` سقف همزمانی stream است.
- ShadowTLS chaining همچنان گراف detour (`SingboxDetourInboundID` + raw `detour`) با resolve به process group و یک owner ریشه‌ای برای هر گروه.
- تغییرات میزبان زنده فقط با مجوز صریح کاربر؛ دستیار دیپلوی نمی‌کند.
- تست‌های شبکه فقط در namespace خصوصی؛ listener زنده پورت 12301 دست‌نخورده.
- اطلاعات محرمانه (رمز، توکن، کلید، کوکی) در اسناد وضعیت ثبت نمی‌شود.

## فایل‌های تغییر‌یافته و منابع (Changed Files & References)

- **کامیت‌های محلی جلوتر از origin/main:** `def29152`, `884ad937`, `6607000c`, `951ab31b`.
- **فایل‌های dirty/untracked (مشاهدهٔ تازه):** `PROGRESS.md`, `HANDOFF.md`, `docs/README.md`, `web/html/core.html`, `docs/plans/tunnel-repositories-review-20261128.md` (??).
- **مسیرهای کلیدی مرتبط:** `config/version` (`7.0.2`), `build/backend/build.sh`, `build/backend/singbox-limits.patch`, `.github/workflows/vpn-aio-release.yml`, `web/service/legacy_migrate.go`, `config/legacy.go`, `web/service/singbox_base.go`, `web/service/singbox_accounts.go`, `web/html/form/protocol/singbox.html`, `web/html/modals/inbound_modal.html`, `web/html/core.html`.
- **اسناد مرجع:** `HANDOFF.md`, `docs/README.md`, `docs/plans/tunnel-repositories-review-20261128.md`, `docs/releases/6.1.0-audit-20251125.md` (مسیر تاریخی در repo؛ در صورت نیاز بررسی شود).

## بررسی‌ها و موارد نامشخص (Verification & Unknowns)

- **مشاهدهٔ تازه در این ذخیره:** `git status --short`, `git log --oneline -8`, `git status -sb`, `git log origin/main..HEAD`, `git diff --stat`, `git describe --tags --always --dirty`, `git log v7.0.2..HEAD`, بررسی `config/version` و خواندن فایل‌های وضعیت/قالب.
- **تأیید نشده در این ذخیره:** اجرای تست‌ها، build، CI انتشار v7.0.2، وضعیت میزبان زنده، صحت runtime کامیت‌های محلی sing-box، و اعتبار مهاجرت/deploy واقعی 7.0.2 روی میزبان.
- **شواهد تاریخی (نه بررسی تازه):** موفقیت CI/اعتبارسنجی 7.0.0/7.0.1/7.0.2 طبق یادداشت‌های قبلی PROGRESS/HANDOFF؛ بررسی دارایی‌های منتشرشده روی `sweden` برای 7.0.0/7.0.1؛ تست‌های sing-box audit/UI در کانتینرهای یک‌بارمصرف قبلی.
- **نکتهٔ مشاهده‌شده در diff محلی:** قالب OpenVPN endpoint در `web/html/core.html` مقدار پسورد را به‌صورت placeholder `[REDACTED_PASSWORD]` درج می‌کند؛ این یک رشتهٔ ثابت در کد UI است، نه راز کاربر، اما قبل از استفاده باید به مقدار امن/واقعی جایگزین یا راهنمایی شود.
- **نامشخص:** آیا کامیت‌های محلی باید push شوند، squash شوند، یا با تغییرات uncommitted دیگر در یک release جمع شوند؛ وضعیت نهایی CI v7.0.2؛ وضعیت فعلی میزبان زنده.

## بن‌بست‌ها و روش‌های ناموفق (Failed Attempts - تکرار نشود)

- در 7.0.0 bundle sing-box قدیمی به‌دلیل cache restore بدون اعتبارسنجی نسخه/marker وارد release شد؛ `build.sh` و کلید cache اصلاح شد.
- مهاجرت rename در 7.0.1 هنگام برخورد مسیر جدید، legacy را کنار می‌گذاشت؛ 7.0.2 آن را به `*.legacy-backup` منتقل و لاگ قدیمی را ادغام می‌کند.
- یک پنل دستی leftover می‌تواند پورت API محلی را اشغال کند (مشاهدهٔ Xray bind error روی 127.0.0.1:21111)؛ قبل از تشخیص خرابی سرویس اصلی، پروسه‌های دستی/کانتینری مزاحم بررسی شوند.
- `TestCustomGeoValidateURL` فقط در netns ایزوله بدون DNS برای `example.com` fail می‌شود؛ خارج از netns pass است.
- اسکریپت‌های چندخطی روی ماشین راه‌دور: here-string PowerShell به `ssh ... 'python3 -'` قابل‌اعتمادتر از نقل‌قول ad-hoc است.

## Previous Status — historical (condensed)

- **v7.0.0 (منتشرشده/تأییدشده):** tag `v7.0.0` روی `6f2ef77d`؛ `config/version=7.0.0`؛ CI انتشار 46069263983 و اعتبارسنجی Ubuntu 46069258533 موفق؛ دارایی‌ها روی `sweden` با `sha256sum -c` OK؛ `-v` = 7.0.0. نقص bundle قدیمی sing-box بعداً در 7.0.0 کشف شد و در 7.0.1/7.0.2 رفع/منتشر شد.
- **v6.6.0:** tag `1fb52746`؛ sing-box update از سورس حذف/قفل شد؛ داشبورد tile sing-box؛ CI/اعتبارسنجی موفق.
- **v6.5.1/v6.5.0:** رفع `/dev/tty` در `deploy.sh`؛ unattended custom login؛ ShadowTLS chaining و remediation حساب‌های مدیریت‌شده؛ تست‌های کانتینری/installer/API قبلی.
- جزئیات تاریخی کامل‌تر در `HANDOFF.md` و `docs/releases/6.1.0-audit-20261125.md`.

## Related Documents

- `HANDOFF.md`: سند تحویل؛ باید با این وضعیت هم‌خوان باشد.
- `docs/plans/tunnel-repositories-review-20261128.md`: تحقیق هفت مخزن تانل (فقط تحقیق).
- `docs/README.md`: فهرست مستندات.
- `docs/releases/6.1.0-audit-20261125.md`: ممیزی مستقل پیش از remediation.
