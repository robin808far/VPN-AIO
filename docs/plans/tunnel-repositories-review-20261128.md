# بررسی فنی هفت پروژهٔ تانل برای VPN-AIO

تاریخ بررسی: **2026-11-28**. درخواست کاربر: بررسی عمیق امکانات و روش پیاده‌سازی و نگهداری یافته‌ها؛ انتخاب طرح، پیاده‌سازی، نصب و انتشار به تصمیم بعدی کاربر موکول است.

این سند نتیجهٔ مطالعهٔ سورس، مستندات، قراردادهای کانفیگ، مسیرهای اصلی انتقال و مدیریت، نصب، مجوزها و اطلاعات انتشار است. «وجود در کد»، «ادعای مستندات» و «تأیید عملی» معادل نیستند. در این بررسی هیچ نصاب، هسته، پنل، تست شبکه یا benchmark این پروژه‌ها اجرا نشد. ارقام سرعت READMEها تأیید مستقل نشده‌اند. این سند گزارش بررسی معماری و قابلیت‌هاست، نه ادعای ممیزی خط‌به‌خط تمام وابستگی‌ها یا تضمین نبود باگ.

## نسخه و منشأ شواهد

هر هفت مخزن به‌صورت shallow clone در مسیر موقت زیر دریافت شدند؛ هیچ سورس خارجی وارد محصول نشده است:

`C:\Users\mef\AppData\Local\Temp\opencode\tunnel-review-20261128\<owner>--<repo>`

این مسیر cache است؛ حافظهٔ ماندگار همین سند و فایل‌های وضعیت پروژه است. برای بازیابی آینده از commitهای جدول استفاده شود، نه صرفاً `main` یا نام نسخه.

| مخزن | commit بررسی‌شده از main | انتشار پایدار مشاهده‌شده در GitHub API | زبان/ساختار |
|---|---|---|---|
| [BackPack](https://github.com/AminMGMT/BackPack) | `ca59b8bdd8d568e330f16e076e560b18c583062d` | `v2.0.0`، 2026-11-20 | Go، هسته و مدیریت در یک باینری |
| [WatTunnel](https://github.com/JohnWatsoon/WatTunnel) | `d67047970dba95a6742063e489e821aa87b8a276` | `v4.0.4`، 2026-11-27 | Go، کتابخانهٔ `wat` و برنامهٔ `wattun` |
| [Kariz](https://github.com/Erfan-XRay/Kariz) | `c6d322508faaea91da846e96ac2487c86c05eed3` | `v2.8.5`، 2026-11-16 | Rust، core و crate جداگانهٔ panel/agent |
| [C-TUNNEL](https://github.com/CIAUB/C-TUNNEL) | `5702a87a2650d9a3a870be178e6e277d32e48e37` | `v2.1.1`، 2026-11-26 | Go، دو باینری و helper شبکهٔ Bash |
| [tuntunnel](https://github.com/mahdiweb3/tuntunnel) | `887e0da588d9324415d44e607723cc8501adb4d5` | `v2.7.1`، 2026-11-26 | Bash با مدیر Python و چند هستهٔ بیرونی |
| [Overal-Tun](https://github.com/ibooplay/Overal-Tun) | `cdee06b891f18bbd5bddcfbec6de0b7ad2d0c64f` | endpoint آخرین release برابر 404؛ سورس خود را `5.0` معرفی می‌کند | Bash، FRP رسمی و پنل Python توکار |
| [Arange-tun](https://github.com/MAHDI-byte64/Arange-tun) | `f10849338e2b9182f041c5ea14d4d07557925e66` | endpoint آخرین release برابر 404؛ `internal/app/app.go` برابر `v1.31.1` | Go، مشتق از BackPack با هسته‌های اضافی |

404 در `releases/latest` یعنی انتشار پایدار Latest از آن endpoint پیدا نشد؛ به‌تنهایی نبود هر tag یا prerelease را ثابت نمی‌کند. BackPack main جلوتر از release است؛ قابلیت‌های جدید main را نباید به باینری `v2.0.0` نسبت داد.

## تفاوت‌هایی که برای طراحی پنل تعیین‌کننده‌اند

| پروژه | شکل اصلی سرویس | جهت | مدیریت ماشین مقابل | آمار قابل اتصال به پنل |
|---|---|---|---|---|
| BackPack | reverse port forwarding؛ direct L3 جدا؛ PCK مستقیم جدا | reverse؛ direct با محدودیت نوع موتور | SSH و عملیات JSON محدود؛ وب API محلی | فایل JSON و Unix control socket جدید |
| WatTunnel | port forwarding با session رمز‌شده روی carrierهای متعدد | direct و reverse؛ محدودیت بعضی carrierها | Scepter داخل تانل؛ قابلیت‌های سطح‌بندی‌شده | HTTP JSON روی loopback، آمار پورت و مسیر |
| Kariz | port forwarding رمز‌شده با entry/exit | direct و reverse | agent با اتصال پایدار و عملیات محدود | Unix socket و داده‌های agent/panel |
| C-Tunnel | kernel L2/L3، TUN رمز‌شده و L4 بدون interface | direct و reverse طبق family | setup link؛ agent سلامت/failover، نه SSH fleet مشابه BackPack | sysfs، فایل آمار L4 و SSE پنل |
| TunTunnel | تولید کانفیگ و orchestration چند هسته | بومی یا با adapter | setup link و benchmark pair | CLI، systemd و probe؛ API وب عمومی ندارد |
| Overal-Tun | FRP رسمی با Nginx اختیاری | reverse | connection code؛ چند frpc روی یک frps | API محلی FRP و ledger پنل |
| Arange-tun | reverseهای اختصاصی و خروجی SOCKS برای برخی موتورهای مستقیم | بسته به نوع | تنظیم دو طرف؛ read token پنل | فایل JSON، Prometheus و API پنل |

- **جهت اتصال** تعیین می‌کند چه ماشینی dial می‌کند؛ با محل ورود کاربر یا server/client بودن در نام محصول یکی نیست.
- **port forwarding** با **شبکهٔ L3/TUN** و **خروجی SOCKS** سه قرارداد متفاوت است. داشتن هرکدام، دیگری را ایجاد نمی‌کند.
- **carrier** شبکهٔ بین دو سرور با TCP/UDP بودن ترافیک کاربران متفاوت است. مثلاً TCP کاربران ممکن است روی KCP/UDP حمل شود.
- نامی مثل `quic-v1`، `game-dtls` یا `frp` در همهٔ پروژه‌ها معنای یکسان یا سازگاری با upstream ندارد.
- آمار payload کاربر، آمار interface و مصرف واقعی روی لینک یکسان نیستند؛ شمارش هر دو سر تانل نیز می‌تواند مصرف را دوبار جمع کند.
- حفظ IP با PROXY protocol صرفاً آدرس مبدأ را منتقل می‌کند؛ شمارش دستگاه فیزیکی، احراز هویت یا quota اکانت از آن به‌تنهایی به دست نمی‌آید.

## ۱. BackPack

### معماری و مسیر انتقال

- `main.go` بین engine، منوی CLI، وب، monitor، proxy و عملیات node dispatch می‌کند.
- حالت engine با `-c` یک کانفیگ TOML و یک تانل را اجرا می‌کند. نصب استاندارد برای هر تانل یک process/systemd unit دارد.
- `cmd/cmd.go` اول `[l3]` را بررسی می‌کند، سپس `[direct]` قدیمی را رد می‌کند، سپس `[server]` یا `[client]` را به موتور reverse می‌دهد.
- reverse: ایران listener و پورت‌های ورودی را دارد؛ خارج control connection و pool اتصال‌های داده را به ایران برقرار می‌کند. کاربر به اتصال آماده یا stream یک mux متصل می‌شود.
- مدیریت generation، control heartbeat، اتصال کاربر به data connection و pool maintenance بین transportها اشتراک دارند. تغییر transport فقط تغییر یک فیلد مستقل از lifecycle نیست.
- main فعلی مسیرهای `internal/plain/{server,client,core}` را نیز دارد؛ سند قدیمی معماری به‌تنهایی نقشهٔ کامل اجرای نسخهٔ فعلی نیست. تشخیص peer قدیمی بعضی اتصال‌ها را به موتور سازگار `v2.0.0` می‌برد.

### امکانات و کانفیگ

- reverse transportها: `tcp`, `tcpmux`, `stealth`, `ws`, `wss`, `wsmux`, `wssmux`, `udp`, `kcp`, `quic`, `xdi`, `pck`.
- `kcp` با تنظیم پنجره، MTU، زمان‌بندی و FEC؛ QUIC واقعی با کتابخانهٔ quic-go؛ xDi روی ICMP؛ PCK روی مسیر raw packet. رفتار UDP خام را با «TCP روی UDP قابل‌اطمینان» یکی نگیریم: transport خام UDP برای forwarding دیتاگرام است.
- port map شامل پورت ساده، bind address، target متفاوت، range و چند backend است. `port_protocol` و fallback قدیمی `accept_udp` تعیین می‌کنند چه ترافیکی پذیرفته شود.
- pool، smux، heartbeat، timeout، source/interface/mark، proxy بیرونی، MSS، buffer، JSON log، zero-copy اختیاری، محدودیت bandwidth و اتصال.
- PROXY protocol v2 در مسیرهای پشتیبانی‌شده برای رساندن آدرس کاربر به backend؛ نه تضمین عمومی برای همهٔ transportها و UDP.
- address fallback، load balancing آدرس‌های همان server و health scoring وجود دارد. این مورد با load balancing بین چند مقصد مستقل یکسان نیست.
- transport fallback فهرست و dwell هماهنگ دو سر دارد؛ تغییر carrier ممکن است اتصال‌های موجود را قطع کند.
- quota/monitor، تاریخچه، watchdog، backup/restore، rollback، Telegram، هشدار و پنل وب از امکانات مدیریت‌اند.

### direct و تغییرات ناسازگار مهم

- `[direct]` stream engine در v2.0.0 حذف شده؛ تشخیص آن برای خطای روشن نگه داشته شده، نه اجرای خودکار.
- `[l3]` برای interface و IP routing است. carrierهای فعلی شامل `xdi`, `sni`, `spoof` و GRE کرنلی `gre`/`ip6gre` هستند.
- مقدار `[l3].carrier="pck"` به موتور Paqet جدا می‌رود: port forwarding است و TUN نمی‌سازد. ظاهر کلید `l3` نباید نوع خروجی را تعیین کند.
- carrierهای L3 قدیمی `udp`/`quic` حذف شده‌اند. GRE کرنلی رمزنگاری ندارد؛ GRE داخل session اختصاصی با آن متفاوت است.
- L3 رمز‌شده از Noise، replay window، rekey، FEC اختیاری، MTU probe و MSS clamp استفاده می‌کند. محدودیت اتصال forwardها، محدودیت تمام ترافیک routed نیست.
- main افزونهٔ encryption با مقادیر `none`, `stealth`, `stealth+`, `shadow`, `scrap` دارد؛ **تعریف enum به معنای قابل انتخاب بودن نیست**: `EncryptionModes` در commit بررسی‌شده `shadow` را در فهرست مجاز transportها قرار نمی‌دهد.
- main همچنین helperهای اختیاری **Naive/HTTP2** با sing-box سمت ایران و NaiveProxy رسمی سمت خارج، و **XHTTP/TLS** یا **RAW/REALITY/Vision** با Xray رسمی دارد. این‌ها wrapper موتور reverse TCP هستند، نه transport مستقل قابل ترکیب با هر گزینه.
- helperها قرارداد نسخه‌دار، credentials مستقل و lifecycle زیر context تانل دارند. در نسخهٔ بررسی‌شده forwarding فقط TCP و یک endpoint است؛ UDP، address/transport fallback و چند ترکیب دیگر رد می‌شوند. wizard ترمینال helperها را تنظیم می‌کند؛ web wizard آن credentials را جمع نمی‌کند. تغییر معتبر certificate/key می‌تواند generation را reload کند.
- این قابلیت‌های main به نسخهٔ helper و build سازگار روی هر دو endpoint وابسته‌اند؛ موجود بودن Xray یا sing-box در VPN-AIO به‌تنهایی سازگاری باینری یا adapter را ثابت نمی‌کند. [قرارداد helperها](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/docs/transports.md).

### مرز مدیریت و ادغام احتمالی

- اجرای مستقل engine، ساخت TOML و خواندن metrics مسیر طبیعی اتصال به پنل ماست؛ import مستقیم بسیاری از بسته‌های `internal` بدون fork ممکن نیست.
- `backpack check -c` برای validation واقعی وجود دارد؛ watcher هر دو ثانیه فایل را بررسی می‌کند و کانفیگ نامعتبر یا بی‌تغییر را اعمال نمی‌کند. reload معتبر معمولاً بازسازی generation است، نه حفظ تمام sessionها.
- snapshot آمار کنار کانفیگ با پسوند `.metrics.json`، معمولاً هر ۳۰ ثانیه؛ شامل connected، peer، traffic، runtime، pool و KCP است.
- Unix socket جدید `/run/backpack/<name>.sock` عملیات `ping`, `status`, `restart-transport` دارد. این ویژگی main را برای release بدون بررسی نباید فرض کرد؛ حتی جملهٔ قدیمی «هیچ‌چیز به engine دسترسی زنده ندارد» در docs با این سورس به‌روز نیست.
- پنل وب routes مشخص `/api/tunnel/*`, `/api/nodes`, `/metrics` و scopeهای دسترسی دارد. این‌ها API مدیریت محصول‌اند، نه پروتکل tunnel data plane.
- fleet فعلی با SSH password و host-key pin از نوع TOFU کار می‌کند؛ `backpack node exec -` درخواست JSON را از stdin می‌گیرد. root login روی پنل نگهداری می‌شود. حذف محلی تانل لزوماً حذف peer نیست.
- source build جدید برای PCK مستقیم به CGO/libpcap نیاز دارد؛ build بدون CGO آن قابلیت را ندارد. release workflow ساخت static با libpcap را در نظر گرفته است. اجرای صرف `go build` تضمین تساوی capabilityها نیست.
- مجوز AGPL-3.0 همراه NOTICE/attribution/trademark؛ تغییر نام نباید noticeهای لازم را حذف کند.

منابع ثابت: [dispatch](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/cmd/cmd.go)، [server factory](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/internal/server/server.go)، [encryption](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/config/encryption.go)، [config reference](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/docs/config-reference.md)، [control socket](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/internal/enginectl/enginectl.go)، [managed servers](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/docs/managed-servers.md)، [static build](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/scripts/build-linux.sh)، [NOTICE](https://github.com/AminMGMT/BackPack/blob/ca59b8bdd8d568e330f16e076e560b18c583062d/NOTICE).

## ۲. WatTunnel

### معماری و امکانات

- هستهٔ `wat` حول `PacketIO` و `Session` ساخته شده؛ carrier بستهٔ از قبل رمز‌شده را حمل می‌کند. stream multiplexing، ACK/retransmit، replay protection، congestion/pacing و datagram forwarding بالاتر از carrier قرار دارند.
- reliability carrierهای TCP/HTTP به session اعلام می‌شود تا مانند UDP/ICMP با آن‌ها رفتار نشود. BondIO چند carrier را زیر یک session ترکیب می‌کند.
- carrierهای اصلی: `icmp`, `udp`, `tcp`, `tls`, `ws`, `wss`, `h2`, `h2c`, `grpc`, `gre`, `ipip`؛ همچنین `ptcp`, `spray` و شکل‌های raw/UDP اضافی.
- شکل‌های `vxlan`, `geneve`, `xfrm-ipsec`, `openvpn-dco` و مشابه، الزاماً implementation پروتکل استاندارد نیستند؛ دو سر باید WatTunnel سازگار باشند. وجود GRE/raw socket در این محصول به معنای ساخت interface GRE کرنلی نیست.
- direct: client ایران پورت‌های کاربر را باز می‌کند و به server خارج متصل می‌شود؛ server مقصد را به پورت مجاز روی `127.0.0.1` خودش محدود می‌کند.
- reverse جهت dial را برای carrierهای پشتیبانی‌شده برعکس می‌کند؛ client/server کاربرد خود را حفظ می‌کنند. همهٔ carrierها، sessions چندگانه و bondها ترکیب آزاد با reverse ندارند.
- bonding با balanced، active-backup و replicate؛ leg جهت‌دار، mute/unmute زنده، سلامت هر مسیر و retry مسیر قطع‌شده. نام‌های carrier تکمیلی محدودیت bond دارند.
- چند session مستقل در بعضی حالت‌ها، connection pool، UDP flow timeout/fragmentation، adaptive FEC، stream window و memory budget.
- TLS/uTLS/ECH/CDN و ابزارهای shaping در کد وجود دارند؛ نتیجهٔ دسترسی روی یک ISP یا مسیر باید جدا سنجیده شود. Auto test مصرف ترافیک چشمگیر دارد.
- رمزنگاری داخل session از ChaCha20-Poly1305 و تبادل ephemeral X25519 استفاده می‌کند. TLS بیرونی به‌صورت پیش‌فرض لزوماً chain verification ندارد؛ `tls-pin` یک کنترل مستقل است. به‌خاطر رمزنگاری session نباید خودسرانه فرض کرد هر لایهٔ TLS هم CA-verified است.

### قرارداد کانفیگ و کنترل

- `wattun -config <file>`؛ YAML **محدود و دست‌ساز**، نه parser کامل YAML. keyها عمدتاً همان flagها هستند؛ `ports` و `limits` دو list ویژه‌اند. flag صریح بر فایل اولویت دارد.
- `ports` نگاشت listen به target و تفکیک TCP/UDP دارد؛ روی server فهرست targetهای مجاز است.
- `ports` و `limits` تقریباً هر سه ثانیه reload می‌شوند؛ سایر کلیدها restart می‌خواهند. فایل خراب state قبلی را حفظ می‌کند.
- rate و quota در سطح **پورت**؛ rate برحسب bytes/s و جدا برای هر جهت است. quota مجموع payload بالا و پایین است، می‌تواند day/week/month reset شود و مصرف آن پایدار ذخیره می‌شود.
- فایل‌های `.quota.json`, `.totals.json`, `.totals.json.host`, `.pn` و اطلاعات pairing بخشی از state هستند؛ نگهداری صرف YAML برای backup کامل کافی نیست. انتقال totals به ماشین دیگر semantics خاص دارد؛ tmpfs روی reboot از بین می‌رود.
- metrics HTTP فقط روی loopback؛ `/stats`, `/ports`, `/stats/history`, `/quota/reset`, `/carrier/toggle`, `/speedtest` و pprof. connected با آخرین بستهٔ authenticated سنجیده می‌شود، نه صرفاً process up.
- برای ادغام محلی می‌توان باینری مستقل را supervise و metrics را از backend پنل خواند. API وب خودش session/2FA و احراز هویت مجزا دارد؛ اجبار به نمایش iframe یا نصب وب آن لازم نیست.
- در مسیرهای اصلی بررسی‌شده گزینه/تولیدکنندهٔ PROXY protocol پیدا نشد؛ نباید حفظ IP واقعی end user برای پنل مقصد را پیش‌فرض گرفت. peer IP در آمار، IP کاربر VPN نیست.

### Scepter و استقرار

- management داخل stream رزروشدهٔ تانل، با JSONL، request ID، deadline/TTL، capability و journal برای جلوگیری از اعمال دوبارهٔ درخواست.
- scopeها: `tunnel`, `create`, `manage`, `full`؛ شامل status، create/edit/delete، lifecycle، carrier test/change، backup و update با rollback در سطح مجاز.
- عملیات remote سطح بالا همچنان به فایل‌ها و systemd خود WatTunnel وابسته‌اند؛ فراخوانی Scepter معادل adapter آماده برای VPN-AIO نیست.
- نسخه‌های جدید برای بعضی تانل‌های قدیمی auto-pair دارند؛ برای پنل ما نقش tunnel token و مجوز مدیریت باید روشن و مستقل مدل شود.
- Linux amd64/arm64 و Windows asset دارد؛ raw carrierها محدودیت OS/privilege دارند. build عمدتاً Go و بدون libpcap خارجی است؛ قابلیت همهٔ OSها یکسان نیست.
- مجوز MIT برای reuse از این فهرست ساده‌تر است؛ اخطارها و مجوز وابستگی‌ها همچنان باید حفظ شوند. این مزیت حقوقی، حکم برتری سرعت یا انتخاب نهایی نیست.

منابع ثابت: [entry point و flagها](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/cmd/wattun/main.go)، [config/reload](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/cmd/wattun/config.go)، [PacketIO](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/wat/io.go)، [session](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/wat/session.go)، [bond](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/wat/bond.go)، [Scepter](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/wat/scepter.go)، [limits/state/metrics](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/docs/TECHNICAL.md#per-port-traffic-and-limits)، [TLS](https://github.com/JohnWatsoon/WatTunnel/blob/d67047970dba95a6742063e489e821aa87b8a276/wat/carrier_tls.go).

## ۳. Kariz

### معماری و قابلیت‌ها

- Rust/Tokio؛ هستهٔ `kariz` هم library است و هم executable. `panel` crate جدا، با HTTP API مبتنی بر Axum، agent و frontend مستقل است.
- `role=entry|exit` مستقل از `mode=direct|reverse`؛ entry پورت کاربر را باز می‌کند و exit به target می‌رسد. acceptor طبق جهت عوض می‌شود.
- هفت transport: TCP، TCP mux، WS، WSS، QUIC، KCP، Paqet؛ featureهای build می‌توانند QUIC/KCP را حذف کنند.
- mux اختصاصی، pool، stream/window/queue limit، timeout، rotation، profileهای balanced/ultraspeed/gaming و tuning جزئی.
- TCP/UDP یا هر دو برای forward rule؛ UDP datagram مستقل در مسیرهای پشتیبانی‌شده، FEC و duplication اختیاری برای کاربرد کم‌تأخیر.
- PROXY v1/v2 برای TCP forward؛ نیازمند هماهنگی backend. UDP آن header را ندارد.
- crypto با mutual token authentication، X25519 و AES-GCM/ChaCha؛ WSS دارای CA verification یا pin؛ QUIC واقعی با quinn و seal اختیاری بسته‌ها.
- Paqet raw با KCP و مدیریت قواعد لازم شبکه؛ با L3/TUN یکی نیست. شبکه‌های GRE/VXLAN یک امکان جدا در مدیریت‌اند.
- `kariz check`, `run`, `status`, `speedtest` و Unix control socket کنار TOML؛ unknown key خطاست.

### مدیریت چند سرور

- agent معمولاً به panel dial می‌کند؛ reverse management هم وجود دارد. مسیر مدیریت از core تانل جداست و آفلاین بودن agent الزاماً به معنای قطع data tunnel نیست.
- join code یک‌بارمصرف و زمان‌دار؛ پس از enrollment کلید اختصاصی agent و عملیات محدود، نه remote shell عمومی.
- setup/edit دو سر با preflight، آغاز acceptor پیش از dialer، انتظار connected و rollback جبرانی. این distributed transaction کامل پایگاه‌داده نیست؛ قطع یک سر همچنان وضعیت rollback قابل گزارش می‌خواهد.
- panel اطلاعات pair، history، events، زمان‌بندی restart، مصرف و quota هشدار، backup رمز‌شده، Telegram و توزیع update امضاشده را مدیریت می‌کند.
- token تانل روی دو endpoint ذخیره می‌شود؛ panel طبق طراحی آن را نگه نمی‌دارد. backup پنل جای backup کانفیگ تانل‌های تمام agentها نیست.
- systemd حالت عادی است؛ `services="process"` برای child process وجود دارد، با تفاوت در پایداری هنگام توقف agent و نگهداری log.

### محدودیت مجوز برای این درخواست

**مجوز فعلی Kariz اجازهٔ ادغام با VPN-AIO را نمی‌دهد.** صرف public بودن سورس، مجوز open source نیست.

- LICENSE بند 3.1 صریحاً wrap/bundle/call و مدیریت، drive یا orchestration هسته از نرم‌افزار دیگر را منع می‌کند.
- بند 3.2 کپی، port و پیاده‌سازی سازگار با protocol/config را نیز منع می‌کند؛ بند 2.1 مطالعه و بررسی را اجازه می‌دهد.
- اجرای مستقل نسخهٔ رسمی برای سرویس‌های خود طبق بند 2.2 با مدیریت کردن آن از پنل خودمان یکی نیست.
- بنابراین ثبت معماری برای شناخت محصول مجاز است، اما adapter یا کپی کد را بدون مجوز مستقل صاحب پروژه نباید گزینهٔ آمادهٔ اجرا دانست.
- این نتیجه دربارهٔ commit/مجوز همین نسخه است؛ وضعیت قانونی نسخه‌های قدیمی باید جدا از روی مجوز همان نسخه بررسی شود. پشتیبانی TunTunnel از Kariz این محدودیت را برای ما برنمی‌دارد.

منابع ثابت: [workspace/dependencies](https://github.com/Erfan-XRay/Kariz/blob/c6d322508faaea91da846e96ac2487c86c05eed3/Cargo.toml)، [schema](https://github.com/Erfan-XRay/Kariz/blob/c6d322508faaea91da846e96ac2487c86c05eed3/src/config.rs)، [pair orchestration](https://github.com/Erfan-XRay/Kariz/blob/c6d322508faaea91da846e96ac2487c86c05eed3/panel/src/pair.rs)، [agent](https://github.com/Erfan-XRay/Kariz/blob/c6d322508faaea91da846e96ac2487c86c05eed3/panel/src/agent.rs)، [API](https://github.com/Erfan-XRay/Kariz/blob/c6d322508faaea91da846e96ac2487c86c05eed3/panel/src/api.rs)، [config documentation](https://github.com/Erfan-XRay/Kariz/blob/c6d322508faaea91da846e96ac2487c86c05eed3/docs/configuration.md)، [LICENSE](https://github.com/Erfan-XRay/Kariz/blob/c6d322508faaea91da846e96ac2487c86c05eed3/LICENSE).

## ۴. C-Tunnel

### معماری و خانواده‌ها

- manager/CLI/panel به نام `ctunnel`، data plane جدا به نام `ctunnel-engine` و helper به نام `ctunnel-kernel` دارد.
- state اصلی JSON؛ هر تانل پس از Normalize فایل env می‌گیرد که helper Bash و engine می‌خوانند. systemd قبل و بعد از engine ساخت/حذف interface و firewall را انجام می‌دهد.
- شمارش مستقیم `pkg/catalog/catalog.go`: **۵۱ روش، ۲۸ فعال برای ساخت جدید، ۲۳ legacy**. فهرست بلند README بدون فیلتر Legacy فهرست قابل انتخاب امروز نیست.
- active kernel L3: `6to4`, `ipip`, `gre`, `ip6gre`, `xgre`, `xgre-ip`؛ active L2: `eoip`, `vxlan`.
- active UDP-Crypt: `wg-stealth`, `rudp-fec`؛ Stealth: `faketcp`, `icmp`, `stealth-tcp`, `stealth-ws`.
- active L4: `tcp`, `tcpmux`, `pck`, `ws`, `wsmux`, `wss`, `wssmux`, `udp`, `udpmux`.
- gaming: `game-udp`, `game-fec`, `game-dup`, `game-kcp`, `game-dtls`؛ `game-kcp` در کد L4 است و بقیه می‌توانند مسیر packet داشته باشند.
- `NoInterface` برای بعضی خانواده‌های جدید forwarding بدون TUN می‌دهد. تصمیم براساس flag و قابلیت روش است، نه صرفاً نام family.
- روش‌های WireGuard، xfrm-ipsec، QUIC-shaped، BareUDP و چند روش دیگر در catalog هنوز هستند ولی در این نسخه برای ساخت جدید legacy شده‌اند.

### معنا و محدودیت روش‌ها

- UDP-Crypt از AEAD اختصاصی AES-256-GCM، کلیدهای جهت‌دار، padding و replay protection استفاده می‌کند؛ header روش روی بسته افزوده می‌شود.
- `quic-v1` implementation QUIC کامل نیست؛ `game-dtls` نیز در `pkg/dpi` به framing قدیمی DTLS-shaped می‌رسد. نام‌های OpenVPN/WireGuard-shaped با پیاده‌سازی واقعی آن سرویس‌ها متفاوت‌اند.
- reverse L4 پورت‌ها را مستقیماً حمل می‌کند؛ kernel/TUN interface و IP لازم ندارد. raw L3/L2 و TUN مسیر دیگری هستند.
- `tcp` و `ws` پس از احراز هویت لزوماً payload را رمز نمی‌کنند؛ HMAC authentication با encryption یکی نیست. WSS/KCP/packet یا AEAD مسیرهای متفاوت دارند.
- PROXY protocol در مدل `TunnelConfig` و مسیر اصلی L4 بررسی‌شده یافت نشد؛ حفظ IP end user نباید بدون روش مستقل فرض شود. kernel DNAT/SNAT نیز semantics متفاوتی دارد.
- شناسهٔ method، نقش، دو IP واقعی، IP خصوصی، subnet index، port، MTU، secret، lanes، forward map، profile، backup method و quota/rate در config هستند.
- نام تانل حداکثر ۱۰ کاراکتر و index معمولاً ۱ تا ۲۵۰ است؛ دلیل نام، محدودیت طول interface لینوکس است.

### مدیریت، نصب و اتصال احتمالی

- Normalize subnet/port/key را تخصیص می‌دهد و conflictهای state را رد می‌کند. `ctunnel://`، JSON base64url از کانفیگ peer است و رمزنگاری‌شده نیست.
- حفظ lifecycle مهم است: **قبل از جایگزینی env قدیمی باید teardown قدیمی انجام شود**؛ وگرنه ExecStop با اطلاعات جدید قواعد قبلی را باقی می‌گذارد. همین ترتیب در `Save` رعایت شده است.
- `AUTO/ECO/TURBO/HYPER` profileها pool/buffer/tuning را تغییر می‌دهند. در برخی L3های direct، انتخاب TURBO/HYPER حتی TCP carrier را فعال می‌کند؛ این فقط برچسب سرعت نیست.
- agent برای failover هماهنگ و benchmark responder؛ backup method با زمان مشترک و دورهٔ rotation هماهنگ می‌شود، بنابراین clock دو سر اهمیت دارد.
- آمار kernel از sysfs و آمار حالت بدون interface از فایل‌های `/run/ctunnel/l4` خوانده می‌شود؛ base پایدار در `/etc/ctunnel/traffic` با boot ID/ifindex از دوباره‌شماری جلوگیری می‌کند.
- quota در سطح تانل با توقف سرویس و manual override/reset/top-up است؛ معادل quota اکانت VPN نیست.
- وب REST/SSE، log، backup/restore، peer IP edit، Telegram، health و connection test دارد. test شکل بسته جای kernel integration test را نمی‌گیرد.
- backend به `ip`, `iptables`, `tc`, kernel modules و برای بعضی روش‌ها TUN/TAP نیاز دارد. port forwarding فعلی قواعد iptables با tag اختصاصی می‌سازد؛ نصب کنار firewall/nftables پنل ما نیازمند طراحی ownership است.
- ادغام مستقل باید manager، engine و network helper را به‌عنوان یک مجموعه ببیند. اجرای تنها `ctunnel-engine` برای اکثر روش‌ها کافی نیست؛ supervisor پنل ما باید setup/teardown را هم پوشش دهد.
- مجوز AGPL-3.0، attribution خود C-Tunnel و قسمت‌های گرفته‌شده از BackPack؛ checksum انتشار موجود است، امضای مستقل release در assets مشاهده‌شده نبود.

منابع ثابت: [catalog/schema](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/pkg/catalog/catalog.go)، [normalization](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/pkg/tunnel/config.go)، [lifecycle](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/pkg/tunnel/service.go)، [env renderer](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/pkg/tunnel/env.go)، [framing](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/pkg/dpi/framing.go)، [crypto](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/pkg/crypto/aead.go)، [traffic](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/pkg/tunnel/traffic.go)، [network helper](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/scripts/ctunnel-kernel.sh)، [NOTICE](https://github.com/CIAUB/C-Tunnel/blob/5702a87a2650d9a3a870be178e6e277d32e48e37/NOTICE).

## ۵. TunTunnel / tuntunnel

### ساختار واقعی

- `tunnel-extra.sh` یک bootstrap Bash و برنامهٔ Python توکار است. فایل ELF عرضه‌شده با `build_binary.py` فقط launcher همان Python است؛ Python runtime را همراه ندارد و core مستقل جدیدی برای همهٔ transportها نیست.
- snapshot مخزن فقط ۱۰ فایل tracked دارد؛ چهار تست Python، دو README، installer، manager، build launcher و تصویر. سورس تمام هسته‌های دانلودی داخل آن نیست.
- `PROFILES` یک registry از recipeهاست، `make_config` کانفیگ هسته و `config_bundle` مجموعهٔ processها را تولید می‌کنند. `bundle_runner.py` از string توکار نوشته می‌شود؛ خروج یک child موجب جمع‌کردن بقیه برای restart کل bundle می‌شود.
- هسته‌ها: sing-box، Xray، BackPack، Backhaul، Rathole، GOST، Kariz و Dagger. version pin هرکدام مستقل است.
- pinهای snapshot: sing-box `1.14.4`، Xray `26.11.9`، BackPack `1.8.5`، Backhaul `0.6.5`، Kariz `2.8.1`، Rathole `0.5.0`، GOST `3.3.3`، Dagger `1.4`. pin جداگانهٔ GOST `3.2.6` برای مسیر قدیمی DTLS در کد باقی مانده، ولی DTLS از profileهای عمومی حذف شده است.

### امکانات و mapping نام‌ها

- ساخت/join/edit/list/status/health/metrics/logs/start/stop/restart/remove، auto-refresh، dry-run، setup link و شبکهٔ خصوصی GRE/IPIP/VXLAN.
- recipeهای TLS، WS/WSS، gRPC، HTTP/2، HTTPUpgrade، XHTTP، REALITY، ShadowTLS، Hysteria2، TUIC و KCP با هسته‌های مختلف اجرا می‌شوند.
- `quic-tls` در این کد به **Hysteria2** روی sing-box می‌رسد؛ `kcp-aes` به BackPack؛ REALITY gRPC/XHTTP و XHTTP به Xray. نباید فقط از جدول README موتور را استنتاج کرد.
- گزینه‌های `pango-*` در registry از sing-box ساخته می‌شوند؛ مدرک استفاده از باینری مستقل Pango یا سازگاری خودکار با هر Pangolin از نام گزینه حاصل نمی‌شود.
- direct برای reverse-onlyهای BackPack/Backhaul/Rathole یک **bundle دو‌لایه** با `tuntunnel-core` است و در validation فقط TCP mapping می‌پذیرد.
- reverse sing-box/Xray با adapter جداگانهٔ Xray reverse انجام می‌شود؛ sing-box به‌خودی‌خود این قرارداد reverse را ندارد.
- GRE/IPIP/VXLAN carrier جدا از transport خارجی است؛ interface، route، MTU و TCPMSS توسط manager اعمال می‌شوند.

### benchmark و مدیریت state

- benchmark یک برنامهٔ Python فشردهٔ توکار است که با AST استخراج شد و ساختار، registry، cleanup و ranking آن بررسی شد؛ کد آن اجرا نشد. ۲۱۴۸ خط؛ SHA-256 متن استخراج‌شده `e27563d2d11af46fd8f2da521f5e34f95ab6db46afc8586d99b95eab22de4353`.
- pairing دو سر، تشخیص شبکه‌ها، prerequisite/core install موقت، تولید کانفیگ هر روش، تست مسیر/transfer و گزارش دارد.
- ranking به هدف وابسته است: gaming روی UDP loss/latency/jitter، ultra-speed روی سرعت دوطرفه، balance روی stability/CPU/RAM، و حالت عادی روی stability/variation/throughput. صرف بزرگ‌ترین download مبنای همهٔ recommendationها نیست.
- cleanup هسته‌های نصب‌شده برای benchmark best effort است؛ prerequisiteهای سیستم ممکن است بمانند. توقف ناگهانی نیازمند بررسی دو سر است.
- state زیر `/etc/tunnel-extra/tunnels`، هسته‌ها زیر `/opt/tunnel-extra/bin`، unitهای systemd و pair JSON و certها. install قبل از تغییر backup محلی از فایل‌ها و enabled/active unitها می‌گیرد و در exception تلاش به restore می‌کند.

### محدودیت‌هایی که باید در طراحی نگه داریم

- pin فعلی BackPack با main/v2 جدید یکسان نیست؛ manager حتی ممکن است `/usr/local/bin/backpack` موجود را بدون version match انتخاب کند. تعویض آن با نسخهٔ جدید تضمین سازگاری recipeها نیست.
- دانلود بعضی هسته‌ها checksum را جست‌وجو می‌کند، اما در `download_engine` اگر تمام URLهای checksum با `URLError` شکست بخورند، پس از پایان loop شرط اجباری «یک checksum معتبر بررسی شد» وجود ندارد. این یک مشاهدهٔ ایستا از مسیر کد است؛ exploit یا reproduction اجرا نشده است.
- `tuntunnel-core` برای direct bridge از release دانلود می‌شود؛ سورس همان bridge در فایل‌های tracked این snapshot پیدا نشد. asset مشاهده‌شدهٔ release فقط amd64 است، هرچند downloader نام arm64 را هم می‌سازد؛ پشتیبانی ARM64 تمام recipeها ثابت نشده است.
- وجود dry-run به معنای تحقق تمام قابلیت‌های runtime یا سازگاری همهٔ engineها نیست؛ بعضی native check ندارند.
- **مجوزی برای خود manager در مخزن نیست.** استفاده از آن به‌عنوان مرجع به معنای مجوز کپی/توزیع نیست؛ هر هسته هم مجوز مستقل دارد. Kariz موجود در registry مجوز ادغام فعلی Kariz را عوض نمی‌کند.
- برای VPN-AIO الگوی registry/render/bundle و تفکیک carrier آموزنده است؛ کپی کل اسکریپت و اجرای آن از وب، مدیریت چرخهٔ عمر و مالکیت فایل‌ها را به پنل نمی‌دهد.

منابع ثابت: [manager و تمام registry/render/adapterها](https://github.com/mahdiweb3/tuntunnel/blob/887e0da588d9324415d44e607723cc8501adb4d5/tunnel-extra.sh)، [launcher](https://github.com/mahdiweb3/tuntunnel/blob/887e0da588d9324415d44e607723cc8501adb4d5/build_binary.py)، [README](https://github.com/mahdiweb3/tuntunnel/blob/887e0da588d9324415d44e607723cc8501adb4d5/README.md)، [release](https://github.com/mahdiweb3/tuntunnel/releases/tag/v2.7.1).

## ۶. Overal-Tun / Over-Tun

### معماری و قابلیت‌ها

- سورس tracked فقط `tunnel.sh`, `README.md`, `logo.png` است. core همان **FRP رسمی fatedier/frp نسخهٔ 0.63.0** است؛ نه هستهٔ شبکهٔ اختصاصی این مخزن.
- ایران `frps` دارد؛ خارج `frpc` دارد و backendها عموماً `127.0.0.1` همان خارج هستند. TCP forward و UDP اختیاری، حداکثر ۲۰ tunnel تعریف‌شده در manager.
- transportهای raw config: TLS/TCP، WebSocket/WSS، QUIC و KCP؛ حالت Stealth یک topology خاص FRP+Nginx است.
- Stealth: frpc خارج → Nginx محلی خارج → HTTPS/WebSocket → Nginx ایران → frps loopback ایران. Nginxها path/header را تبدیل می‌کنند، cert/decoy/IP allowlist دارند و FRP داخلی بدون TLS مضاعف پیکربندی می‌شود.
- کانفیگ واقعی Nginx هم TLS 1.2 و هم 1.3 را مجاز می‌کند؛ عنوان README «TLS 1.3» را نباید «فقط TLS 1.3» خواند. Nginx/OpenSSL هم به‌صرف User-Agent مرورگر، TLS fingerprint مرورگر پیدا نمی‌کند.
- چند خارج از گروه load-balancer **TCP** در FRP استفاده می‌کنند. در generator، گروه برای UDP نوشته نشده؛ ادعای Multi-Kharej را نباید خودکار برای همان شکل از UDP تکرار کرد.
- modeهای medium/turbo/advanced با pool و tuning سراسری؛ watchdog فرایند/اتصال، کنترل پورت، migration، update، IP lock و self-audit.
- config اصلی env/TOML زیر `/etc/overtun`، code به‌شکل base64 از key/value و secrets، unitهای `overtun*` و Nginx conf جدا برای هر tunnel.

### پنل و آمار

- Python stdlib `ThreadingHTTPServer` توکار به فایل جدا نوشته می‌شود؛ cookie session، PBKDF2-SHA256، محدودیت login و APIهای stats/config/traffic/manage/job دارد.
- manager با CLI غیرتعاملی فراخوانی می‌شود؛ کارهای طولانی job/status/log دارند. backend وب خودش data plane نیست.
- per-port ledger دادهٔ API FRP را دوره‌ای جمع می‌کند؛ نمودار live گاهی از service bytes و در fallback از کل NIC تغذیه می‌شود. «همان عدد x-ui» یک ادعای README است و برابری حسابداری اکانت ثابت نشده است.
- در سورس **Fake Speed Display** واقعی وجود دارد: `_fake_bw`, `_fake_ping`, تنظیمات `FAKE_*` و `/api/fakespeed`. نصب تازه `FAKE_SPEED_ON=0` می‌نویسد؛ بنابراین پیش‌فرض جعلی نیست، اما نمودار/پینگ قابل تغییر نمایشی است و شاهد benchmark قابل اعتماد محسوب نمی‌شود.
- unique source IP از connectionهای میزبان، تعداد device/account واقعی نیست.
- وب پیش‌فرض HTTP عمومی است؛ cookie ساخته‌شده HttpOnly/SameSite دارد ولی Secure نیست. TLS تانل با TLS پنل مدیریت یکی نیست.

### محدودیت برای اتصال به VPN-AIO

- معماری FRP رسمی/گروه‌های proxy مرجع قابل شناختی است، ولی اجرای کل installer بخش‌هایی از Nginx، sysctl، route، certificate و حتی cleanup نصب Rathole قبلی را تغییر می‌دهد. این عملیات مناسب lifecycle یک فرم پنل نیست.
- `install_binary` دانلود FRP را با checksum اجباری بررسی نمی‌کند؛ script update هم با اجرای installer یکسان نیست. برای product bundling لازم است artifact نسخه‌دار و قابل بررسی تعریف شود.
- نداشتن source FRP در این مخزن طبیعی است چون upstream مشخص است؛ مجوز FRP و مجوز manager باید مستقل بررسی شوند.
- README badge MIT دارد، اما فایل LICENSE یا grant صریح برای script در snapshot وجود ندارد. آن badge را مجوز کافی برای کپی فرض نکنیم.
- برای طرح آینده می‌توان FRP رسمی را مستقل از این manager ارزیابی کرد؛ انتخاب یا افزودن dependency در این تسک انجام نشده است.

منبع ثابت: [تمام generatorها، پنل و lifecycle](https://github.com/ibooplay/Overal-Tun/blob/cdee06b891f18bbd5bddcfbec6de0b7ad2d0c64f/tunnel.sh)، [README](https://github.com/ibooplay/Overal-Tun/blob/cdee06b891f18bbd5bddcfbec6de0b7ad2d0c64f/README.md). محل‌های مهم script: `install_binary`، `write_nginx_iran_stealth`، `write_nginx_kharej_stealth`، `kh_proxies`، `gen_iran_config`، `gen_kharej_config`، `set_fake_speed_conf`، `sampler` و `ledger_tick`.

## ۷. Arange-tun

### ساختار و خانواده‌های واقعی

- fork با ساختار BackPack قدیمی؛ یک Go binary و engine بر اساس TOML، CLI/web/monitor/proxy در حالت‌های جدا. `-c` هر بار یک تانل را اجرا می‌کند.
- هفت خانوادهٔ قابل تشخیص در جدول/dispatch: Backhaul-style reverse، frp اختصاصی، Rathole تغییریافته، Packet، WireGuard/AWG، SSH و Hedioum. README از «هشت نوع» می‌گوید؛ این شمارش با جدول و هفت مسیر اصلی یکی نیست و باید گزینه‌های UI را از کد گرفت.
- reverse عمومی transportهای TCP/mux/stealth، UDP/KCP/QUIC، WS/WSS/mux، xDi، spoof و PCK دارد. همان نام‌ها تضمین سازگاری با BackPack امروز نیستند.
- `[wireguard]`, `[packet]`, `[ssh]`, `[hedioum]` از `[server]`/`[client]` جدا هستند و در dispatch زودتر بررسی می‌شوند.

### هر موتور چگونه کار می‌کند

- **frp داخلی:** کد `internal/frp` صریحاً protocol اختصاصی و نوشته‌شده از نو است، فقط نام frp را گرفته؛ auth و smux خودش را دارد و با frpc/frps رسمی Overal-Tun یکی نیست. یک peer فعال را با peer جدید جایگزین می‌کند.
- **Rathole داخلی:** control channel و pool data channel با الگوی Rathole دارد، ولی target و framing UDP را تغییر داده؛ سازگاری wire با Rathole رسمی ثابت نیست و طبق توضیح سورس دو سر باید از همین کد باشند.
- **Packet:** کد vendored از Paqet، libpcap/raw packet، KCP+smux، forwarding و SOCKS اختیاری؛ topology خارج server و ایران client است. TUN/خروجی L3 کرنلی برای این حالت فرض نشود.
- **WireGuard/AWG:** server کرنلی خارج؛ client با netstack/userspace device و SOCKS محلی. این مسیر معادل `VpnOutDriver` پنل ما که netdev واقعی برمی‌گرداند نیست. تنظیمات AWG در schema از نسل Jc/Jmin/Jmax/S1/S2/H1..H4 است؛ پذیرش همهٔ نسل‌های AWG جدید ثابت نیست.
- **SSH:** SOCKS محلی روی SSH، احراز هویت password؛ در سورس `ssh.InsecureIgnoreHostKey()` است. برخلاف تعبیر TOFU در متن، fingerprint ذخیره/مقایسه نمی‌شود؛ هویت سرور واقعاً pin نشده است.
- **Hedioum:** engine vendored و adapter به configuration آن، foreign listener و iran hub/SOCKS، pool و چند mimic. NOTICE می‌گوید اجازهٔ نویسنده برای vendoring در همین پروژه گرفته شده؛ انتقال همان مجوز به محصول دیگر را نباید فرض کرد.

### امکانات و حدود آن‌ها

- lifecycle مشترک create/edit/delete/start/stop، config reload، systemd، metrics JSON، web API/Prometheus، health، watchdog، backup/restore، source update/rollback، Telegram از طریق peer.
- pool، preset، fallback و PROXY protocol در reverse اصلی هستند؛ در تمام موتورهای اختصاصی یکسان نیستند.
- در `internal/frp` و `internal/rathole` مسیر خواندن `ServerConfig.MaxConnections/BandwidthMbps/ProxyProtocol` در بررسی ایستا یافت نشد؛ وجود این گزینه‌ها در schema عمومی اثبات اعمال آن روی همهٔ tunnel typeها نیست. این observation تست runtime نشده است.
- counterهای بعضی موتورهای reverse با `metrics.AddBytes` جمع می‌شوند؛ برای SSH/Packet/Hedioum نباید پوشش یکسان traffic یا quota را از نمایش کارت نتیجه گرفت.
- Go module `1.24.4` و toolchain `1.24.7`؛ build همهٔ امکانات با CGO و libpcap است. Makefile cross build وابستگی toolchain/libpcap هر معماری را ذکر می‌کند؛ «یک باینری» الزاماً «کاملاً static و بی‌نیاز از libpcap» نیست.
- installer/update از source build می‌کند؛ آخرین release پایدار API پیدا نشد. CI موفق قدیمی روی commitهای دیگر دیده شد، نه تأیید همین HEAD.
- مجوز AGPL-3.0 و notices اجزای BackPack/Rathole/Paqet/Hedioum باید لحاظ شوند. repo name، saltها و نام tunnelها تغییر یافته‌اند؛ قابلیت interop خودکار با upstream تأیید نشده است.

منابع ثابت: [dispatch](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/cmd/cmd.go)، [schema](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/config/config.go)، [frp اختصاصی](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/internal/frp/frp.go)، [Rathole](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/internal/rathole/rathole.go)، [SSH](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/internal/ssh/ssh.go)، [WireGuard](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/internal/wireguard/wg.go)، [Hedioum notice](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/internal/hedioum/engine/NOTICE.md)، [build](https://github.com/MAHDI-byte64/Arange-tun/blob/f10849338e2b9182f041c5ea14d4d07557925e66/Makefile).

## مجوزها و نحوهٔ استفاده در تصمیم بعدی

| پروژه | وضعیت مشاهده‌شده | اثر بر طرح آینده |
|---|---|---|
| BackPack | AGPL-3.0 + NOTICE/trademark | reuse با رعایت تعهدات مجوز و attribution؛ ادغام source با GPL فعلی پنل نیازمند بررسی نحوهٔ توزیع است |
| WatTunnel | MIT | گزینهٔ قابل بررسی برای باینری مستقل یا reuse کتابخانه با حفظ notice |
| Kariz | مجوز اختصاصی source-available | wrapper/orchestration از پنل دیگر صریحاً ممنوع؛ نیاز به مجوز مستقل |
| C-Tunnel | AGPL-3.0 + notices خود و BackPack | همان ملاحظات copyleft/attribution و اجزای helper |
| TunTunnel | LICENSE ندارد | مطالعهٔ معماری؛ کپی/توزیع manager مجاز فرض نشود؛ engineها مستقل‌اند |
| Overal-Tun | badge MIT، grant صریح پیدا نشد | مجوز script روشن نیست؛ FRP رسمی را جدا ارزیابی کنیم |
| Arange-tun | AGPL-3.0 + اجزای ثالث | مجوزهای تمام engineهای vendored و دامنهٔ permission Hedioum بررسی شود |

این جدول گزارش متن مخازن است و تصمیم نهایی برای fork، linking، bundle یا توزیع محصول در این تسک گرفته نشده است.

## اتصال به معماری فعلی VPN-AIO: فقط نقشهٔ امکان‌ها

سورس محلی در زمان بررسی: HEAD برابر `7d688912ecc807d13f1023a0e272d5636bc1c486`، `config/version=7.0.1` و ابتدا working tree پاک بود. اسناد قبلی وضعیت هنوز checkpointهای قدیمی‌تر دارند؛ وضعیت release/سرور از این observation نتیجه نمی‌شود.

| بخش فعلی پنل | چیزی که واقعاً فراهم می‌کند | نتیجه برای تانل جدید |
|---|---|---|
| `web/service/procmgr.go` | child process، process group، log ring، restart و stop | مناسب اجرای engine مستقل؛ network setup و صحت اتصال باید جدا کنترل شوند |
| `backend/backend.go` و build bundle | daemon manifest، per-arch extraction، atomic replace | می‌تواند مبنای بسته‌بندی هستهٔ تأییدشده باشد؛ manifest/version/hash/capability مهم‌اند |
| `web/service/vpnoutbound.go` | `VpnOutDriver.Up` باید netdev واقعی بدهد؛ تولید freedom outbound pin‌شده | مناسب L3 واقعی؛ reverse port forward یا SOCKS را نباید با interface ساختگی در آن جا داد |
| `vpnoutcarrier.go`, `vpnoutvia.go` | حمل outer transport از طریق netdev یا outbound، routing و fail-closed | حامل بودن تانل و forward شدن سرویس دو مفهوم متفاوت‌اند؛ raw/GRE و TCP/UDP محدودیت مستقل دارند |
| `sshoutbound.go` | الگوی SOCKS facade برای خروجی | برای موتورهایی که SOCKS می‌دهند مرجع نزدیک‌تری از driver netdev است |
| `web/service/node_service.go` و `web/controller/node.go` | ثبت/probe/read وضعیت و inbounds نود با token | در کد فعلی API کامل ساخت/ویرایش تانل روی peer فراهم نیست؛ داشتن Nodes آن را خودکار ایجاد نمی‌کند |
| account/traffic و core services | هویت کاربر، quota/expiry و چرخهٔ عمر coreهای فعلی | آمار port/tunnel نباید جای هویت و quota اکانت نوشته شود |

فایل‌های مرجع موجود: [طرح carrier](tunnel-carrier-plan.md)، [ProcManager](../../web/service/procmgr.go)، [bundle](../../backend/backend.go)، [VPN outbound](../../web/service/vpnoutbound.go)، [Nodes service](../../web/service/node_service.go).

الگوهای قابل استفاده در تصمیم بعدی، بدون انتخاب نهایی:

1. مدیریت یک یا چند engine مستقل از UI فعلی، با renderer و metrics adapter مختص هر engine.
2. مدیریت port forwarding روی دو سر در مدل خودش؛ افزودن L3 و SOCKS فقط وقتی engine آن قرارداد را واقعاً ارائه می‌کند.
3. جداسازی ثبت سرور، pairing/credentials، تنظیم مشترک pair و runtime هر endpoint؛ connected بودن data plane از online بودن control plane جدا گزارش شود.
4. preflight و ذخیرهٔ staged، health واقعی پس از apply، rollback جبرانی و ثبت partial failure. «process started» به‌تنهایی موفقیت تانل نیست.
5. pin نسخه/commit و capability matrix؛ استفاده از `latest` یا binary موجود میزبان بدون version check، خصوصاً برای BackPack و adapterهای TunTunnel، سازگاری را تضمین نمی‌کند.
6. مدیریت مالکیت port، interface، subnet، firewall، cert و فایل‌ها در سطح پنل؛ اجرای installerهای بزرگ از درخواست وب، قرارداد نصب قابل برگشت نمی‌سازد.

## شواهد بررسی و محدودیت‌های واقعی

- GitHub metadata، latest release/assets و سه workflow run اخیر هر مخزن با درخواست read-only بررسی شد.
- CI موفق برای HEADهای BackPack، WatTunnel، Kariz و C-Tunnel مشاهده شد؛ این **نتیجهٔ CI نویسندگان** است و به معنای اجرای تست توسط ما نیست. TunTunnel و Overal-Tun workflow run نداشتند؛ Arange فقط موفقیت commitهای قدیمی‌تر را نشان داد.
- فایل‌های ورودی و config، dispatch موتور، interfaces/transport factory، persistence/metrics، API/agent، نصب/build و مجوزها با docs تطبیق داده شدند. fixtureها و نام/ساختار آزمون‌های موجود برای تعیین پوشش قابل مشاهده بررسی شدند؛ شمارش فایل test معیار کیفیت یا تعداد تست پاس‌شده نیست.
- benchmark فشردهٔ TunTunnel بدون import/exec برنامه استخراج و AST parse شد؛ manager Python آن و پنل Python توکار Overal نیز parse شدند. اجرای benchmark یا نصب prerequisite اتفاق نیفتاد.
- ارجاع‌های سورس با فایل واقعی clone و commit تطبیق داده شدند؛ لینک‌های محلی گزارش و whitespace بررسی شدند. `git diff --check` برای اسناد tracked پاس شد؛ فقط هشدارهای تبدیل LF/CRLF موجود محیط گزارش شد. این‌ها کنترل سند هستند، نه تست محصول.
- catalog C-Tunnel مستقیماً شمرده شد: ۵۱ کل، ۲۸ فعال، ۲۳ legacy.
- سورس کامل `tuntunnel-core` و وابستگی‌های بیرونی از خود این snapshot قابل بررسی کامل نبود. باینری‌های release دانلود/اجرا یا با سورس تطبیق باینری نشدند.
- رفتار روی مسیر واقعی ایران/خارج، کارایی، reconnect طولانی، limit enforcement هر transport، CDN، NAT، kernel، ARM64 و استفادهٔ هم‌زمان کنار VPN-AIO هنوز runtime validation می‌خواهد.
- هیچ اتصال SSH به سرور زنده، تغییر سرویس/پورت/DB، dependency جدید، commit، push یا release انجام نشده است. فقط اسناد پژوهش و وضعیت محلی تغییر کرده‌اند.

## نقطهٔ ادامه

کاربر هنوز نگفته کدام رفتار، engine، صفحهٔ UI و مدل مدیریت دو سر را می‌خواهد. در پاسخ بعدی ابتدا خواستهٔ او را با این capabilityها تطبیق بده؛ از این گزارش مجوز پیاده‌سازی یا نصب استنتاج نکن. قبل از هر طراحی دقیق، commit/مجوز و نسخهٔ مورد انتخاب دوباره کنترل شود، چون چند مخزن فعال و در حال تغییرند.
