# 分流器從 outdoorsafetylab 專案搬到 rudymap

Status: r4
Issue: #32

## 背景

魯地圖分流器目前跑在 `outdoorsafetylab` 這個 GCP 專案。`rudy-tiles` 已經在 `rudymap` 專案，
GSA 也已經搬走，分流器是 `outdoorsafetylab` 裡剩下最大的一塊公益服務。
維護者 2026-10-19 決定搬到 `rudymap`，先寫計畫。

2026-10-19 盤點到的依賴：

| 項目 | 現況 |
|---|---|
| 服務 | `rubymap`（`rudymap.tw`，主力，一天上千次轉址）、`ruby-balancer`（`rudy.oslab.tw`，備援入口）、`ruby-balancer-alpha`（master 合併即部署） |
| 網域 | 兩個都是 Cloud Run domain mapping。`rudymap.tw` 的 DNS 在 Cloudflare，沒開代理；`oslab.tw` 的 DNS 在 GoDaddy |
| Firestore | `outdoorsafetylab` 預設資料庫（asia-northeast2）的 `RudyMirrorSites`、`RudyMirrorFiles`、`RudyMirrorSiteStats`（含每日 `DailyStats`，2022 年起）、`RudyClientSalt`（本季的 client id 鹽） |
| 轉址 log | log sink → BigQuery `outdoorsafetylab.rudymap_file_redirect`（asia-east1，約 34 萬列、196 MB）；季報（`outdoorsafetylab/outdoorsafetylab` 的 `tools/rudymap-report/`）讀這份 dataset 和 `DailyStats` |
| 建置 | 三個 Cloud Build trigger、映像在 `outdoorsafetylab` 的 Artifact Registry（asia-northeast1） |
| 健康檢查 | `deploy/terraform/`（#27、#31）：WIF 身分、`rudy-healthcheck` job 與 Scheduler、`statuspage-key` secret |
| 其他 | CloudFront 用量計需要 AWS 金鑰；geoipd 在 `outdoorsafetylab`，另有其他 app 在用；Statuspage 與 GCP 無關 |

完整盤點見文末〈附錄：盤點〉（2026-10-19）。

## 決定

### D1 目標專案是既有的 `rudymap`

魯地圖的兩個服務放在同一個專案，帳單與權限一起看。
被否決：另開新專案（多一個專案要管，沒有換到任何隔離上的好處）。

### D2 Terraform：`deploy/terraform/` 改管 `rudymap`，舊的改名留到最後

一個專案一個 root。現有 root 改名為 `deploy/terraform-outdoorsafetylab/`（backend 不變，state 不動），
`deploy/terraform/` 重新指向 `rudymap`，state 放新的 `rudy-balancer-tfstate-rudymap`，
GitOps 身分在 `rudymap` 重建。plan／apply workflow 兩個 root 都跑，直到舊 root 在最後一步清空、
apply 刪掉剩下的資源後移除。

`rudymap` 專案的 root 放在 rudy-balancer。`rudy-tiles` 目前手動建的資源（AR、trigger、bucket、
告警）不在本票收編，之後另開票 import 進同一個 root。

被否決：
- 只改 `project` 變數、沿用同一份 state：一次 apply 會在舊專案刪、新專案建，切換前舊的健康檢查就沒了。
- 兩個 repo 各自一個 root 管同一個專案：違反一專案一 root。

### D3 Firestore：在 `rudymap` 新建預設資料庫，位置 asia-east1

服務都在 asia-east1，現在的資料庫在大阪（asia-northeast2），每次讀寫都跨區。
位置建立後不能改，所以這是本計畫最不可逆的一項。

### D4 資料：只按路徑複製分流器的資料，鹽要在第一個下載請求前到位

- 要搬的是 `RudyMirrorSites`、`RudyMirrorFiles`、`RudyMirrorSiteStats`、`RudyClientSalt`，
  以及統計本身所在的子集合 `RudyMirrorSiteStats/{site}/DailyStats/{day}`。
- **用一次性的複製程式，只讀這些路徑，不用 Firestore 的匯出／匯入。** 匯出只能指定 collection
  group，而舊資料庫裡另一個 app 也有叫 `DailyStats` 的子集合（`Accounts/{id}/DailyStats`）。
  用匯出會把那個 app 的資料一起帶進 `rudymap`。資料量約一萬筆文件，程式複製幾分鐘就完成，
  補差額也用同一支程式。
- `RudyMirrors` 不搬：那是 2022 年的舊資料，現在的程式已經不讀。
- **匯入完成前，不對新服務發任何下載請求。** 下載會讀 `RudyMirrorFiles`（空庫回 500），也會在
  找不到本季鹽時產生新鹽並快取在 instance 裡；之後再匯入正確的鹽，已在跑的 instance 也不會換。
  缺鹽的後果是同一季的 client id 斷成兩段，季報的不重複用戶數灌水。
- 搬遷期間兩邊資料庫都在累加 `DailyStats`（以 UTC 分日）。補差額涵蓋**從匯出那天起，到舊服務不再
  收到轉址為止的每一天**：新庫的值加上「舊庫最終值減匯入時的值」。
- **補差額只做一次，而且等到不再回滾才做**：第 5 步切換後穩定 24 小時，才算定案。定案之前若要回滾，
  只需把新庫自匯入以來的增量加回舊庫；舊庫沒被動過，不會重複計算。
- `RudyMirrorSites`、`RudyMirrorFiles` 下一輪健康檢查就會重寫，照樣複製，免得新服務一開始沒有站台狀態。

被否決：
- 接受搬遷窗口的計數遺失：窗口長度要等 D6 的探測才知道，而且第 3 到 5 步可能分在不同天。
- 用 Firestore 匯出／匯入：會把另一個 app 的 `DailyStats` 帶過來（見上）。

### D5 轉址 log：歷史表複製到 `rudymap`，sink 改寫新 dataset

- 在 `rudymap` 建同名 dataset（asia-east1），第 3 步建 sink 開始收新專案的 log。第 2 步部署後的驗證
  請求沒有 sink 收，只是測試流量，不影響季報。
- 舊 dataset 的全部歷史在**最後一次**併進新表：等第 5 步定案（同 D4 的 24 小時），舊服務不再收到轉址，
  且已過一天，讓延遲送達的 log 都已落表。合併以每筆 log 的 `insertId` 去重，只補新表沒有的列，
  因此不靠任何時間點切分。
- 季報改讀 `rudymap`，包括 BigQuery 和 Firestore 兩處，這是 `outdoorsafetylab/outdoorsafetylab` 的一個 PR。

被否決：新專案的 sink 跨專案寫回舊 dataset。這樣歷史不用搬，但分流器又依賴回 `outdoorsafetylab`，等於沒搬完。

### D6 網域切換方式由探測決定

同一個網域的 Cloud Run domain mapping 不能同時掛在兩個專案。重新掛載時要重新簽憑證，
HTTPS 可能中斷幾分鐘到幾小時。各家 app 直接打 `rudymap.tw`，中斷就是下載失敗。

所以第一步是探測：用一個拋棄式子網域（`rudymap.tw` 底下）先掛在 `outdoorsafetylab` 的服務，
再改掛到 `rudymap`，量兩種做法的中斷時間：

- (a) 直接重掛。
- (b) 重掛期間先在 Cloudflare 開代理（SSL Full，跟 `tiles.rudymap.tw` 一樣）。

`rudymap.tw` 用中斷較短的做法。`rudy.oslab.tw` 的 DNS 不在 Cloudflare、流量也小，直接重掛，
並且先切它，當作正式切換的預演。

**探測結果（2026-10-19）與定案：直接重掛，`rudymap.tw` 在台灣時間凌晨 3 點切。**

- (a) 實測：刪除舊 mapping 後，舊站還照常服務約 5 分鐘；之後連續 TLS 失敗約 **11.5 分鐘**，
  直到新憑證簽發並擴散完成。
- (b) 沒有實測，理由有二。第一，中斷不會變短：Cloudflare 連到 Google 時，一樣會在新憑證簽發前
  握手失敗。第二，會讓統計失真：服務把 `X-Forwarded-For` 的最後一跳當成用戶位址，經過代理後會變成
  Cloudflare 機房的位址，國家統計和 client id 都會錯。
- 時段：過去 30 天，`rubymap` 在台灣時間 3 點那一小時平均約 7 次轉址，是全天最低；
  11.5 分鐘的中斷約影響 1 到 2 次下載。

domain mapping 由維護者手動建立，不放 Terraform：Terraform 身分要能建 mapping，得成為網域的
Search Console 擁有者，等於合併 PR 就能動到網域控制權。

### D7 三個服務都搬，改用專用服務帳號，金鑰改放 Secret Manager

- `ruby-balancer-alpha` 一起搬：它跟 master 走，留在舊專案會讓舊專案永遠清不掉。
- 服務改用專用服務帳號，只給 Firestore 讀寫和需要的 secret。
- 服務需要的金鑰都改用 Secret Manager 參照，值由人手動放，不進 state。
- 服務設定照舊專案的現況重建（2026-10-19 讀回）：
  - 三個服務：128Mi、1 vCPU、並行 80、關閉 startup CPU boost、TCP 啟動探測、ingress all、公開呼叫。
  - 各自不同的部分照舊：`rubymap` 最多 10 個 instance、逾時 10 秒；`ruby-balancer` 最多 100 個、逾時 10 秒；
    alpha 最多 5 個、逾時 300 秒。
  - 環境設定也照各自現況：只有 `rubymap` 和 alpha 有 CloudFront 用量計，`ruby-balancer` 沒有，照舊不加。
    alpha 上一個程式已經不讀的舊設定不帶過去。

### D8 geoipd 不搬

另有其他 app 在用，不屬於分流器。分流器照舊呼叫它的網址。

### D9 健康檢查：一次只開一邊

新專案的 job 和 Scheduler 先建好但不排程。切換 `rudymap.tw` 時，新的開、舊的停（兩個 root 各改一個開關，
放在同一個 PR）。這樣不會兩邊同時寫 Statuspage：兩邊對同一個鏡像若看到不同結果，狀態會來回跳動。

**前置條件：GitHub 的排程 workflow（`.github/workflows/healthcheck.yml`）在第 5 步之前已經移除**，
連同它用的 `GCP_SA_KEY`。它也寫舊 Firestore 和同一個 Statuspage；留著的話，「一次只開一邊」不成立，
清理後還會把舊 collection 重新寫出來。這原本就排在 #27 之後的收尾裡。

匯入完成後、新服務接流量前，手動跑一次新 job，刷新新資料庫的站台狀態。

### D10 舊資源在切換後觀察 7 天才刪

7 天包含一個週末的流量高峰。要刪的清單：
- `outdoorsafetylab` 的三個服務、三個 trigger、AR 倉庫、sink 與 dataset、`rudymap-aws-*` 兩個 secret。
- Firestore 只刪分流器的路徑：`RudyMirrorSites`、`RudyMirrorFiles`、`RudyMirrorSiteStats`（含其下的
  `DailyStats`）、`RudyClientSalt`，以及停用的 `RudyMirrors`。**不可以用 collection group 刪 `DailyStats`**，
  那會連另一個 app 的資料一起刪掉。
- 舊 root 裡的全部資源：WIF pool、服務帳號、`statuspage-key`、job、Scheduler。
- 舊 state bucket。

## 步驟

每一步都能單獨驗證、單獨回滾。到第 4 步之前，對外完全沒有變化。

| 步驟 | 內容 | 驗證 | 回滾 |
|---|---|---|---|
| 0 | D6 探測（2026-10-19 完成，結果見 D6） | 直接重掛的中斷約 11.5 分鐘 | 拋棄式子網域已刪 |
| 1 | D2 改名舊 root、新 root bootstrap（維護者建 state bucket＋本機 apply）：GitOps 身分、Firestore、AR、服務帳號、secret 容器、BigQuery dataset、job（關閉） | 第二次 plan 為空 | destroy 新 root（Firestore 位置除外，見 D3） |
| 2 | `rudymap` 建 trigger（手動），部署三個服務，不掛網域；**不發下載請求**（D4） | 服務 ready、`/v1/apps` 200 | 刪服務 |
| 3 | D4 匯入 Firestore；D5 建 sink；手動跑一次新 job | 鹽的文件與舊庫相同；`/v1/stats` 有歷史；run.app 上的下載回 302（判準見測試策略） | 重新匯入 |
| 4 | 切 `rudy.oslab.tw`（D6） | 轉址判準同第 3 步；記錄實際中斷時間 | 改掛回舊服務；新庫增量加回舊庫（D4） |
| 5 | 離峰切 `rudymap.tw`；D9 切健康檢查（前置：GitHub workflow 已移除）；穩定 24 小時定案後，D4 補差額、D5 合併歷史 | 轉址判準同上；新專案 log 與 BigQuery 有新列；窗口統計判準見測試策略 | 僅限定案前：改掛回舊服務、健康檢查開關改回，新庫增量加回舊庫（D4） |
| 6 | 季報改讀 `rudymap` | 用 2026-Q3 重算，數字與已發布的季報相同 | revert 季報 PR |
| 7 | 觀察 7 天後刪舊（D10） | `outdoorsafetylab` 裡找不到分流器的資源 | 無（刪除不可逆，所以要等 7 天） |

## 測試策略

這次搬的是資源和資料，不是程式碼，所以要抓的是「搬錯了」和「少搬了」：

- **鹽：** 新舊資料庫本季的鹽文件逐位元相同。不同的話，同季的 client id 會斷開，而且單元測試抓不到。
- **站台與檔案清單：** 切換前，比對新舊服務 `/v1/sites` 與 `/v1/files` 中決定分流的內容：有哪些站台與檔案、
  每個檔案在哪些站台可用、大小與修改時間、站台狀態。檢查時間、延遲這類每輪重新量的值不比，
  因為新舊 job 各自量測，本來就會不同。這兩份清單也是 app 下載前讀的資料。
- **轉址：** 分流是依權重隨機挑鏡像，兩次請求本來就可能去不同地方，所以不比對單次目的地。比的是：
  新舊服務對同一個檔案列出的可用鏡像集合相同，而且對新服務多次下載的每個目的地都在這個集合裡。
- **歷史統計：** D4、D5 都做完後，用已發布的 2026-Q3 季報當基準在新專案重算一次，只要有不同，就代表歷史漏搬（例如漏了 `DailyStats`）。
- **搬遷窗口的統計：** Q3 重算看不到 10 月的窗口。窗口內每個 UTC 日，新庫 `DailyStats` 的總次數要等於
  BigQuery 同一天的轉址列數。搬遷前先用前一週確認這兩者本來就相等，當作基準；本來就不相等的話，
  改用兩邊各自的差額互相核對。
- **Terraform：** 每個 PR 的 plan 留言要跟 PR 描述的預期一致；bootstrap 之後的第二次 plan 必須為空。

## 附錄：盤點

2026-10-19 逐類列出 `outdoorsafetylab` 的資源，並搜尋本機 59 個 repo，看還有誰直接讀分流器的資料。

| 類別 | 屬於分流器、要搬 | 不屬於分流器 |
|---|---|---|
| Cloud Run 服務 | `rubymap`、`ruby-balancer`、`ruby-balancer-alpha` | `geoipd`、`geoipd-alpha` 等其他服務 |
| Cloud Run job／Scheduler | `rudy-healthcheck`（舊 root） | 其他 Scheduler |
| 網域 | `rudymap.tw`、`rudy.oslab.tw` | 其他 domain mapping |
| Firestore | 見 D4 的路徑 | `Accounts`、`users`、`ErrorReports` 等；`Accounts/*/DailyStats` 的複合索引 |
| BigQuery／log | `rudymap_file_redirect` 與其 sink | `_Default`、`_Required` log bucket |
| 建置 | 三個 trigger、AR `rudy-balancer` | geoipd、moidem 的 trigger；已停用的 `rudy-tiles-taiwan` trigger |
| Bucket | `rudy-balancer-tfstate-outdoorsafetylab` | `rudymap.tw`（見下）、geoip 資料等 |
| Secret | `rudymap-aws-*`、舊 root 的 `statuspage-key` | 其他 |
| 監控 | 無（舊專案沒有分流器的告警或 uptime check） | |

- **直接讀分流器資料的：** 只有季報（第 6 步）。app 與 Locus 等工具都經由網域呼叫，網域不變就不受影響。
- **`rudymap.tw` bucket：** 7.3 GiB 的公開地圖檔，最後更新於 2022-08，過去 30 天沒有對外流量，
  也不在鏡像設定裡。它不是分流器的執行依賴，不在本計畫搬遷；要刪或保留另外決定。
- **用 Cloud Asset Inventory 核對（2026-10-19，1,440 筆資源）：** 沒有找到新的分流器依賴。
  手動列表沒列到的只有以下幾項，都與分流器無關：
  - Firebase 的 BigQuery 匯出與 API key（其他 app）。
  - 一個 2019 年留下、沒有任何端點也沒接上負載平衡器的 NEG `rudymaps`，可在第 7 步一併刪除。
  - 分流器的網頁沒有用到任何 Google API key。
- **看不到的：** 如果有 BI 報表直接讀那個 BigQuery dataset，從專案裡看不出來。

## 變更歷程

- r1（2026-10-19）：初版。
- r2（2026-10-19）：a2a r1（1 High、5 Medium）。D4：
  - 改為五個 collection group，補上子集合 `DailyStats`。
  - 匯入前不發下載請求，避免新鹽被快取。
  - 差額改為涵蓋匯出起到舊服務停止收轉址的每一天，並補上回滾時的反向做法。

  其他：
  - D5：先複製歷史表才建 sink，sink 移到第 3 步。
  - D9：GitHub 排程 workflow 移除改為第 5 步的前置條件。
  - 測試策略：轉址不比單次目的地，改比可用鏡像集合；新增搬遷窗口的統計核對。
- r3（2026-10-19）：a2a r2（2 Medium、1 Low）。
  - D5：歷史改在最後一次以 `insertId` 去重合併，不再以複製時間切分（延遲送達的 log 會漏）。
  - D4：補差額等切換定案（穩定 24 小時）才做一次。定案前的回滾只把新庫增量加回舊庫，避免重複計算。
  - 第 1 步的步驟表拿掉 sink。
  - （a2a r3 Low）步驟表的回滾欄標明「僅限定案前」。
- r4（2026-10-19）：完整盤點與探測結果。
  - D4：改用路徑限定的複製程式，因為另一個 app 也有 `DailyStats`；`RudyMirrors` 不搬。
  - D6：記下探測結果，定案為直接重掛、凌晨 3 點。
  - D7：列出要重建的服務設定。
  - D10：Firestore 只按路徑刪除。
  - 測試策略：新增 `/v1/sites`、`/v1/files` 比對。
  - 新增〈附錄：盤點〉。
  - a2a r4（1 Medium）：`/v1/sites`、`/v1/files` 只比對決定分流的內容，排除延遲、檢查時間等每輪量測值。
  - 附錄加上 Cloud Asset Inventory 的核對結果。
