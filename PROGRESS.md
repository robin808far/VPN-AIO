# Project Progress & State

آخرین به‌روزرسانی: 2026-11-28 (ذخیرهٔ وضعیت؛ رفع باگ cloudflared + commit + کشف باگ واقعی تازه در OpenVPN endpoint users، وسط رفعش)

Project root: `C:\Users\mef\Documents\OpenCode Projects\VPN-AIO`

## وضعیت فعلی (Current Status)

- **شاخه/نسخهٔ محلی:** `main` روی `cfa97d22` (کامیت رفع باگ cloudflared)، جلوتر از `origin/main` با چندین کامیت محلی هنوز push‌نشده (`884ad937`, `6607000c`, `951ab31b`, `b010e8be`, `7099d00c`, `f757691d`, `cfa97d22`؛ `7099d00c` قبلاً push شده بود طبق مشاهدهٔ قبلی، اما در این نشست دوباره کامیت جدید روی آن اضافه شد — وضعیت دقیق ahead/behind در این ذخیره دوباره بررسی نشده، با `git status -sb` تازه چک شود). `config/version` = `7.1.0` (در این نشست از `7.0.2` به `7.1.0` ارتقا و ریلیز شد).
- **ریلیز v7.1.0:** push شد، تگ `v7.1.0` ساخته و push شد، هر دو CI workflow (`VPN-AIO release`, `VPN-AIO Ubuntu validation`) موفق شدند؛ asset‌های amd64/arm64 + SHA256SUMS روی GitHub منتشر و چکسام تأیید شد.
- **کاربر میزبان زنده `sweden` را به v7.1.0 آپدیت کرد** (تأیید شده با `VPN-AIO-amd64 -v` = `7.1.0` و SHA مطابق asset منتشرشده). سرویس اصلی زنده (`xray` روی پورت 12301) دست‌نخورده ماند.
- **تست عملیاتی کامل روی `sweden` در حال انجام است** (با اجازهٔ صریح کاربر؛ بک‌آپ کامل قبل از شروع گرفته شد در `/home/ubuntu/assistant-test-backup-20261128/` — دیتابیس، کانفیگ sing-box، باینری در حال اجرا؛ بک‌آپ هنوز بازگردانده نشده، چون سرویس زنده‌ٔ xray دست‌نخورده مانده و فقط باینری panel/sing-box چند بار جایگزین شده‌اند بدون آسیب به داده‌ها).
- **در این نشست یک باگ واقعی تازه در endpoint OpenVPN کشف شد** (نه مربوط به تصمیم قبلی «فرم گرافیکی کامل نسازیم» — آن تصمیم همچنان پابرجاست): قالب `addOpenvpnServerEndpoint()` در `core.html` یک فیلد نادرست `name` در شیء کاربر می‌گذاشت که sing-box واقعی آن را با خطای `unknown field "name"` رد می‌کند؛ ساختار صحیح طبق sing-box فقط `username`/`password` است. این باگ **در حال رفع** است (ادیت روی `core.html` اعمال شده، هنوز build/test/commit نشده).
- **درخواست‌های باز کاربر:** ادامهٔ تست عملیاتی کامل (کانفیگ واقعی روی سرور) برای همهٔ ویژگی‌های sing-box اضافه‌شده در این چرخه: Snell ✅، Listen fields ✅، Cloudflared ✅ (بعد از ۴ رفع باگ + build tag)، OpenVPN endpoint (در حال رفع باگ users، هنوز تکمیل نشده)، Hysteria Realm (هنوز تست عملیاتی نشده در این نشست).

## کارهای انجام‌شده (Completed)

### چرخهٔ قبلی — تا v7.0.2 (خلاصهٔ تاریخی، جزئیات کامل در نسخه‌های قبلی این فایل و HANDOFF.md)

- v7.0.0/v7.0.1/v7.0.2: مهاجرت نام‌گذاری `vpn-ui`->`vpn-aio`، رفع نقص bundle sing-box قدیمی، رفع برخورد مهاجرت legacy.
- کامیت‌های sing-box: Snell inbound (`884ad937`)، Listen fields مشترک (`6607000c`)، فرم Cloudflared (`951ab31b`)، OpenVPN Server endpoint template با رفع باگ پسورد placeholder (`b010e8be`)، Hysteria Realm سرویس+فیلد inbound (`f757691d`).
- تصمیم کاربر: فرم گرافیکی کامل برای OpenVPN endpoint در sing-box ساخته نشود؛ همان دکمهٔ قالب+JSON کافی است چون پنل یک پروتکل OpenVPN native جدا دارد.

### این نشست — ارتقا به v7.1.0 و تست عملیاتی کامل روی سرور زنده

- **نسخه و ریلیز:** `config/version` به `7.1.0` تغییر، commit (`7099d00c`)، push شاخه + تگ، هر دو CI موفق، assets تأیید شد.
- **کاربر سرور `sweden` را آپدیت کرد.**
- **بک‌آپ کامل قبل از تست عملیاتی:** `/home/ubuntu/assistant-test-backup-20261128/` شامل دیتابیس pre-test، کانفیگ `/etc/vpn-aio-singbox` tar، باینری panel در حال اجرا، وضعیت سرویس.
- **تست Snell (موفق):** از طریق API پنل یک inbound واقعی ساخته شد. کشف مهم (ثبت‌شده در حافظهٔ دانش هم): برای ساخت دستی JSON (نه از طریق فرم UI که خودش `userkey` می‌سازد)، شیء کاربر Snell باید **هم `password` هم `userkey`** داشته باشد — پنل از `password` برای اعتبارسنجی سطح بالا می‌خواند اما sing-box واقعی فقط `userkey`/`psk` سطح بالا را می‌پذیرد. نسخهٔ معتبر Snell فقط `5` یا `6` است (نه `4`). `sing-box check` واقعی رد شد، instance (`server-6`) بالا آمد و پایدار ماند.
- **تست Listen fields (موفق):** یک inbound VLESS با `tcp_fast_open`, `tcp_multi_path`, `udp_fragment`, `tcp_keep_alive*` ساخته شد؛ پذیرفته شد، instance (`server-7`) بالا آمد.
- **تست Cloudflared — ۵ باگ واقعی پیدا و رفع شد (کامیت `cfa97d22`):**
  1. `build/backend/build.sh`: تگ build `with_cloudflared` در لیست tags مربوط به sing-box نبود؛ بدون آن باینری sing-box اصلاً پروتکل Cloudflared را نمی‌شناخت (خطای واقعی: "Cloudflared is not included in this build"). اضافه شد.
  2. `validateSingboxLimits` (`web/service/singbox_inbound.go`): بدون قید و شرط `users[]` الزامی می‌کرد؛ Cloudflared کاربر مدیریت‌شده ندارد. استثنا اضافه شد.
  3. `saveSingboxInbound` (همان فایل): `listen`/`listen_port` را از JSON می‌خواند و رنج `1-65535` را الزامی می‌کرد؛ Cloudflared این فیلدها را اصلاً ندارد. استثنا اضافه شد (مقدار موجود روی `inbound` دست‌نخورده می‌ماند).
  4. `runtimeInbound` (`web/service/singbox.go`): هنگام تولید کانفیگ runtime، بدون قید و شرط آرایهٔ `users` می‌خواست. استثنا اضافه شد.
  5. `applyGroup` (همان فایل): Cloudflared را به‌عنوان «عضو با لیست کاربر خالی» می‌شمرد و کل گروه را silently متوقف می‌کرد (بدون خطای قابل‌دیدن) — این مشکل‌سازترین باگ بود چون هیچ خطایی برنمی‌گشت، فقط فرآیند sing-box هیچ‌وقت start نمی‌شد. استثنا اضافه شد (اگر همهٔ اعضای گروه cloudflared باشند، چک entries/emptyMember رد نمی‌شود).
  - **تست با توکن واقعی Cloudflare Tunnel** (کاربر خودش یک Tunnel تستی در Zero Trust ساخت و توکن را داد): بعد از رفع هر ۵ باگ، inbound ساخته شد، `server-9` instance بالا آمد و **۴ اتصال UDP فعال واقعی به Cloudflare edge (`198.41.192.0/24`:7844)** دیده شد — یعنی تانل واقعاً زنده و متصل بود، نه فقط پذیرش کانفیگ. بعد از تأیید، inbound تستی حذف شد (instance متوقف شد). کاربر باید Tunnel تستی را از داشبورد Cloudflare خودش هم حذف کند (دستیار دسترسی به آن داشبورد ندارد).
  - **کامیت شد:** `cfa97d22 fix(singbox): make cloudflared inbounds actually runnable` — شامل هر ۴ رفع کد + build tag + پیام commit کامل با جزئیات تست.
- **روش build/test محلی روی سرور (نکتهٔ تکنیکال مهم برای ادامه):** این ماشین ویندوزی قبلاً تصور می‌شد gcc/cgo ندارد، اما **خود سرور `sweden` یک toolchain کامل Go+gcc پشتیبانی می‌کند** — راه‌حل واقعی: Go (نسخهٔ 1.24.0) موقتاً در `/tmp/go-toolchain` روی `sweden` نصب شد (بدون نیاز به دسترسی root سیستمی دائمی)، سورس کامل پروژه (شامل تغییرات uncommitted) با `tar`+`scp` منتقل شد به `/tmp/vpnaio_build/`، و با `CGO_ENABLED=1` کامپایل کامل (نه stub) ساخته شد. باینری sing-box جداگانه هم با build tag اصلاح‌شده، از طریق `docker run` با Alpine (که از قبل روی سرور نصب بود) دوباره ساخته شد. این روش برای ادامهٔ تست OpenVPN/Hysteria Realm هم باید تکرار شود.
- **چهار بار جایگزینی باینری panel زنده روی سرور** (هر بار با `systemctl stop` -> `mv` (نه `cp`, چون فایل در حال اجرا busy است) -> `chmod +x` -> `systemctl start`) — هر بار بعد از تأیید سالم بودن سرویس و بازگشت sing-box instances قبلی.
- **یک رویداد بحرانی موقت (رفع‌شده):** اولین تلاش کراس-کامپایل با `CGO_ENABLED=0` روی ویندوز باینری ساخت که `go-sqlite3` را stub (غیرفعال) می‌کرد؛ جایگزینی آن باینری باعث کرش فوری سرویس شد (`Error initializing database: ... requires cgo`). فوراً با بازگردانی باینری بک‌آپ‌شده رفع شد؛ هیچ دیتایی آسیب نخورد (چکسام دیتابیس قبل/بعد یکسان بود).

### این نشست — کار نیمه‌تمام روی OpenVPN endpoint (هنوز کامل نشده)

- **باگ واقعی تازه کشف شد:** تست مستقیم endpoint `openvpn-server` از طریق base-config (`endpoints[]`) با یک inbound واقعی sing-box نشان داد ساختار JSON شیء کاربر که `core.html` می‌سازد (`{ name, username, password }`) دارای فیلد اضافی `name` است که sing-box واقعی قبول نمی‌کند (خطا: `endpoints[0].users[0].name: json: unknown field "name"`). ساختار صحیح طبق مستند رسمی sing-box فقط `username`/`password` است (بدون `name`).
- **ادیت اعمال‌شده (uncommitted):** در `web/html/core.html`، تابع `addOpenvpnServerEndpoint()` اصلاح شد تا `users: [{ username: 'user1', password: RandomUtil.randomSeq(...) }]` بسازد (فیلد `name` حذف شد).
- **هنوز رفع‌نشده/تست‌نشده در این توقف:** بعد از حذف `name`، خطای endpoint OpenVPN به یک باگ دیگر (شبیه Cloudflared) تغییر کرد: **پروتکل OpenVPN هم در build tags فعلی sing-box نیست** (خطای واقعی: `OpenVPN is not included in this build, rebuild with -tags with_openvpn,with_gvisor for system:false`). باید `with_openvpn` به لیست tags در `build/backend/build.sh` اضافه شود (مثل `with_cloudflared`)، سپس دوباره باینری sing-box روی سرور بازسازی و تست کامل شود.
- **فایل محلی `_tmp_check_openvpn_users.js`** (موقت، برای تأیید عینی حذف فیلد `name` بدون نگاه به مقدار پسورد) هنوز روی دیسک است؛ باید قبل از commit پاک شود.

## قدم‌های بعدی (Next Steps)

- [ ] **تکمیل رفع باگ OpenVPN endpoint:** `with_openvpn` را هم به build tags در `build/backend/build.sh` اضافه کن (کنار `with_cloudflared` موجود)؛ باینری sing-box را دوباره روی `sweden` با docker بازسازی کن؛ باینری panel را هم دوباره بسازد اگر کد Go دیگری لازم شد (فعلاً فقط `core.html` که کد فرانت‌اند embedded است، نیاز به rebuild کامل panel هم دارد چون embedded است)؛ دوباره endpoint OpenVPN را با یک inbound/endpoint واقعی تست کن تا `sing-box check` و بالا آمدن instance تأیید شود.
- [ ] **تست عملیاتی Hysteria Realm:** هنوز در این نشست تست عملیاتی روی سرور انجام نشده. باید یک سرویس `hysteria-realm` در base-config + یک inbound Hysteria2 با فیلد `realm` ساخته و روی سرور با باینری اصلاح‌شده تست شود.
- [ ] **پاکسازی فایل‌های موقت محلی:** `_tmp_check_openvpn_users.js` و هر فایل `_tmp_*` دیگر باقی‌مانده را قبل از commit حذف کن.
- [ ] **پاکسازی فایل‌های موقت روی سرور:** `/tmp/vpnaio_build`, `/tmp/go-toolchain`, `/tmp/singbox_out`, `/tmp/singbox_patch_test`, `/tmp/*.json`, `/tmp/*.py`, `/tmp/*.sh`, باینری‌های تستی قدیمی (`vpnaio_test_native`, `_native2`, `_native3`) — بعد از اتمام کامل تست عملیاتی، همهٔ این مسیرهای موقت باید پاک شوند تا فضای دیسک آزاد شود (آخرین مشاهده: ۱۷ گیگ آزاد از ۵۸ گیگ کل).
- [ ] **commit رفع باگ OpenVPN endpoint** بعد از تکمیل و تست (مثل الگوی commit Cloudflared: شرح کامل باگ‌ها + نحوهٔ تست واقعی).
- [ ] **تعیین تکلیف push کامیت‌های جدید این نشست** (`cfa97d22` و هر کامیت بعدی OpenVPN/Hysteria) به `origin/main` — کاربر باید تصمیم بگیرد آیا یک ریلیز patch جدید (`v7.1.1`؟) لازم است یا این رفع‌باگ‌ها در کامیت بعدی روی `main` بمانند تا ریلیز بعدی.
- [ ] **بازگرداندن بک‌آپ تست روی `sweden`:** طبق قول داده‌شده به کاربر، بعد از اتمام کامل همهٔ تست‌های عملیاتی، باینری panel/sing-box نهایی (نسخهٔ commit‌شده و بازسازی‌شده، نه نسخه‌های آزمایشی میانی) باید جایگزین شود و/یا وضعیت نهایی با کاربر تأیید شود که آیا بازگرداندن کامل بک‌آپ لازم است یا باینری فعلی (که از کد commit‌شده ساخته شده) همان چیزی است که باید بماند.
- [ ] **رمز موقت پنل:** در ابتدای این نشست رمز کاربر `mef` موقتاً به یک مقدار تستی تغییر داده شد (`--pass`) تا بتوان لاگین کرد؛ طبق قول داده‌شده، این باید با بازگرداندن بک‌آپ دیتابیس یا تنظیم مجدد توسط کاربر به حالت اصلی برگردد — **هنوز انجام نشده در این ذخیره**.

## تصمیم‌ها و محدودیت‌ها (Decisions & Constraints)

- فرم گرافیکی کامل برای OpenVPN endpoint در sing-box ساخته نمی‌شود (تصمیم قبلی، همچنان پابرجا) — اما **باگ واقعی ساختار JSON (فیلد `name` اضافی) جدا از آن تصمیم است و باید رفع شود**، چون این یک نقص کارکردی است نه کمبود UI.
- تغییرات روی میزبان زنده فقط با مجوز صریح کاربر؛ کاربر در این نشست صراحتاً اجازهٔ کامل SSH/دستکاری/تست عملیاتی روی `sweden` را داد، به‌شرط بک‌آپ قبل و بازگرداندن بعد از اتمام.
- اطلاعات محرمانهٔ واقعی (توکن Cloudflare Tunnel واقعی که کاربر داد، کلیدهای WireGuard موجود کاربران واقعی دیده‌شده در فهرست inbound) در این فایل یا هیچ‌کجای دیگر ثبت نشده و نباید بشود.
- سرویس زندهٔ اصلی کاربر (`xray` روی پورت 12301) در طول همهٔ این تست‌ها دست‌نخورده ماند.
- مدل محدودیت‌های sing-box، ShadowTLS chaining، و سایر محدودیت‌های معماری قبلی بدون تغییر پابرجا هستند (جزئیات در نسخه‌های تاریخی این فایل / HANDOFF.md).

## فایل‌های تغییر‌یافته و منابع (Changed Files & References)

- **کامیت‌شده در این نشست:** `config/version` (7099d00c)، `web/service/singbox_inbound.go` + `web/service/singbox.go` + `build/backend/build.sh` + `PROGRESS.md` (cfa97d22, رفع ۵ باگ cloudflared).
- **Uncommitted در این لحظه:** `web/html/core.html` (رفع باگ فیلد `name` اضافی در OpenVPN endpoint users — نیاز به build tag `with_openvpn` و تست قبل از commit).
- **مسیرهای کلیدی:** `build/backend/build.sh` (build tags sing-box)، `web/service/singbox_inbound.go` (اعتبارسنجی save-time)، `web/service/singbox.go` (runtime config + applyGroup)، `web/html/core.html` (فرم base-config + OpenVPN endpoint template)، `web/html/form/protocol/singbox.html` (فرم Hysteria2 realm field).
- **مسیرهای موقت روی سرور `sweden` (باید در پایان پاک شوند):** `/tmp/go-toolchain/`, `/tmp/vpnaio_build/`, `/tmp/singbox_out/`, `/tmp/singbox_patch_test/`, `/tmp/vpnaio_test_native*`, `/tmp/*.json`, `/tmp/*.py`, `/tmp/*.sh`, `/tmp/cookie.txt` (سشن لاگین موقت).
- **بک‌آپ تست روی سرور:** `/home/ubuntu/assistant-test-backup-20261128/`.

## بررسی‌ها و موارد نامشخص (Verification & Unknowns)

- **تأیید واقعی این نشست:** Snell (ساخت inbound + `sing-box check` واقعی + instance پایدار)، Listen fields (همان)، Cloudflared (ساخت + check + instance + **اتصال شبکهٔ واقعی زنده به Cloudflare edge با توکن واقعی**) — بالاترین سطح تأیید ممکن بدون دسترسی کاربر نهایی واقعی.
- **تأیید نشده در این ذخیره:** OpenVPN endpoint (در حال رفع، نیمه‌کاره)، Hysteria Realm (هیچ تست عملیاتی در این نشست انجام نشد، فقط در نشست‌های قبلی با shim مرورگری تست شده بود که با اجرای واقعی sing-box یکسان نیست).
- **نامشخص:** آیا باینری panel/sing-box که الان روی `sweden` در حال اجراست دقیقاً همان کدی است که commit شده (`cfa97d22`) یا نسخه‌ای جلوتر/عقب‌تر با تغییرات uncommitted OpenVPN — باید قبل از ادامه با `md5sum`/نسخه تأیید شود.
- **نامشخص:** وضعیت دقیق git ahead/behind نسبت به `origin/main` بعد از کامیت `cfa97d22` — آخرین مشاهدهٔ صریح `git status -sb` قبل از این کامیت بود.

## بن‌بست‌ها و روش‌های ناموفق (Failed Attempts - تکرار نشود)

- کراس-کامپایل با `CGO_ENABLED=0 GOOS=linux` از ویندوز برای این پروژه باینری **stub بدون sqlite واقعی** می‌سازد که روی اجرا بلادرنگ کرش می‌کند (`requires cgo to work. This is a stub`). راه‌حل درست: کامپایل مستقیم روی خود `sweden` با `CGO_ENABLED=1` (gcc از قبل روی آن سرور نصب است).
- `cp` روی باینری در حال اجرا (`/opt/VPN-AIO/bin/sing-box` وقتی پروسه‌هایش فعال‌اند) با خطای `Text file busy` شکست می‌خورد؛ باید اول `systemctl stop`/پروسه‌ها متوقف شوند یا از `mv` (نه `cp`) استفاده شود چون `mv` inode را عوض می‌کند نه محتوا.
- دستورهای چندخطی پیچیده با quoting تودرتو (`sh -euc "..."` داخل `ssh "..."` از pwsh، یا `python3 -c "..."` با escape چندلایه) تقریباً همیشه در pwsh شکست می‌خورند؛ راه‌حل قابل‌اعتماد همیشه: نوشتن محتوا در یک فایل جدا با ابزار `write`، سپس `scp`/اجرای همان فایل.
- `journalctl _PID=<pid>` و خواندن `/proc/<pid>/fd/1` برای دیدن stdout زندهٔ یک پروسهٔ sing-box در حال اجرا هیچ خروجی مفیدی نداد (sing-box ظاهراً جایی دیگر یا اصلاً فایل لاگ جدا نمی‌نویسد در این تنظیمات)؛ برای تأیید عملکرد واقعی، بررسی اتصالات شبکهٔ فعال (`lsof -p <pid>` / `ss -tnp`) روش قابل‌اعتمادتری بود.
- خطاهای عمومی «users array لازم است» / «port باید 1-65535 باشد» برای پروتکل‌های sing-box بدون کاربر مدیریت‌شده (cloudflared) در **چهار مکان مجزا** در کد (`validateSingboxLimits`, `saveSingboxInbound`, `runtimeInbound`, `applyGroup`) باید همزمان رفع شوند — رفع فقط یک یا دو مکان هنوز خطای بعدی را نشان می‌دهد؛ این الگو برای OpenVPN endpoint هم ممکن است تکرار شود (فعلاً فقط باگ فیلد `name` + نبود build tag کشف شده، ممکن است مکان‌های مشابه دیگری هم باشند که هنوز تست نشده).
- در ۷.۰.۰ bundle sing-box قدیمی به‌دلیل cache restore بدون اعتبارسنجی نسخه/marker وارد release شد؛ `build.sh` و کلید cache اصلاح شد (تاریخی).

## Previous Status — historical (condensed)

- **v7.0.0 (منتشرشده/تأییدشده):** tag `v7.0.0` روی `6f2ef77d`؛ CI انتشار و اعتبارسنجی Ubuntu موفق؛ دارایی‌ها روی `sweden` تأیید شد.
- **v6.6.0:** tag `1fb52746`؛ sing-box update از سورس حذف/قفل شد.
- **v6.5.1/v6.5.0:** رفع `/dev/tty` در `deploy.sh`؛ unattended custom login؛ ShadowTLS chaining.
- جزئیات تاریخی کامل‌تر در [HANDOFF.md](HANDOFF.md) و `docs/releases/6.1.0-audit-20251125.md`.

## Related Documents

- [HANDOFF.md](HANDOFF.md): سند تحویل؛ هنوز با این ذخیرهٔ وضعیت هم‌تراز نشده — نیاز به به‌روزرسانی مشابه دارد تا رفع باگ‌های cloudflared و وضعیت نیمه‌تمام OpenVPN را هم ثبت کند.
- `docs/plans/tunnel-repositories-review-20261128.md`: تحقیق هفت مخزن تانل (فقط تحقیق، بدون ارتباط مستقیم با این چرخهٔ sing-box).
- `docs/README.md`: فهرست مستندات.
- `docs/releases/6.1.0-audit-20251125.md`: ممیزی مستقل پیش از remediation.
