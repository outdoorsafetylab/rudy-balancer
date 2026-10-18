# 分流器從 outdoorsafetylab 專案搬到 rudymap

Status: r1
Issue: #32

## 背景

魯地圖分流器目前跑在 `outdoorsafetylab` 這個 GCP 專案。`rudy-tiles` 已經在 `rudymap` 專案，
GSA 也已經搬走（globalspatialatlas#120），分流器是 `outdoorsafetylab` 裡剩下最大的一塊公益服務。
Director 2026-10-19 決定搬到 `rudymap`，先寫計畫。

2026-10-19 盤點到的依賴：

| 項目 | 現況 |
|---|---|
| 服務 | `rubymap`（`rudymap.tw`，主力，一天上千次轉址）、`ruby-balancer`（`rudy.oslab.tw`，備援入口）、`ruby-balancer-alpha`（master 合併即部署）；都以預設 compute 服務帳號執行 |
| 網域 | 兩個都是 Cloud Run domain mapping。`rudymap.tw` 的 DNS 在 Cloudflare，沒開代理；`oslab.tw` 的 DNS 在 GoDaddy |
| Firestore | `outdoorsafetylab` 預設資料庫（asia-northeast2）的 `RudyMirrorSites`、`RudyMirrorFiles`、`RudyMirrorSiteStats`（含每日 `DailyStats`，2022 年起）、`RudyClientSalt`（本季的 client id 鹽） |
| 轉址 log | log sink → BigQuery `outdoorsafetylab.rudymap_file_redirect`（asia-east1，約 34 萬列、196 MB）；季報（`outdoorsafetylab/outdoorsafetylab` 的 `tools/rudymap-report/`）讀這份 dataset 和 `DailyStats` |
| 建置 | 三個 Cloud Build trigger、映像在 `outdoorsafetylab` 的 Artifact Registry（asia-northeast1） |
| 健康檢查 | `deploy/terraform/`（#27、#31）：WIF 身分、`rudy-healthcheck` job 與 Scheduler、`statuspage-key` secret |
| 其他 | CloudFront 用量計的 AWS 金鑰是一般環境變數；geoipd 在 `outdoorsafetylab`、公開、另有其他 app 在用；Statuspage 與 GCP 無關 |

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

### D4 資料：四個 collection 全複製，鹽要在接流量前到位

- 用 Firestore 的匯出／匯入，指定這四個 collection，搬到新資料庫。
- `RudyClientSalt` 必須在新服務接到第一個真實請求前就位。缺少的話，新服務會為本季產生新鹽，
  同一季的 client id 會斷成兩段，季報的不重複用戶數會灌水。
- 切換那幾天，兩邊資料庫都在累加 `DailyStats`。切換完成後補一次差額：
  以匯出時的值為基準，把舊庫之後的增量加到新庫，只處理切換當天和前一天。
- `RudyMirrorSites`、`RudyMirrorFiles` 下一輪健康檢查就會重寫，照樣複製，免得新服務一開始沒有站台狀態。

被否決：接受切換窗口的計數遺失。窗口長度要等 D6 的探測才知道，可能長達數小時。

### D5 轉址 log：歷史表複製到 `rudymap`，sink 改寫新 dataset

- 在 `rudymap` 建同名 dataset（asia-east1，同區可以直接複製表）。先把歷史表複製過去，再建新 sink 寫入。
- 切換窗口兩邊都會有 log，最後把舊 dataset 在窗口內的列併進新表。
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

domain mapping 由 Director 手動建立，不放 Terraform：Terraform 身分要能建 mapping，得成為網域的
Search Console 擁有者，等於合併 PR 就能動到網域控制權（devops 10-12 的意見）。

### D7 三個服務都搬，改用專用服務帳號，金鑰改放 Secret Manager

- `ruby-balancer-alpha` 一起搬：它跟 master 走，留在舊專案會讓舊專案永遠清不掉。
- 服務改用專用服務帳號，只給 Firestore 讀寫和需要的 secret。現在用的預設 compute 帳號在
  `outdoorsafetylab` 有 `datastore.owner` 等多餘權限。
- AWS 金鑰和 Statuspage key 改用 Secret Manager 參照，值由人手動放，不進 state。

### D8 geoipd 不搬

它是公開服務，沒有驗證，另有其他 app 在用。分流器跨專案呼叫它的網址就好，不需要額外授權。

### D9 健康檢查：一次只開一邊

新專案的 job 和 Scheduler 先建好但不排程。切換 `rudymap.tw` 時，新的開、舊的停（兩個 root 各改一個開關，
放在同一個 PR）。這樣不會兩邊同時寫 Statuspage：兩邊對同一個鏡像若看到不同結果，狀態會來回跳動。

新服務接流量前，手動跑一次新 job，刷新新資料庫的站台狀態。

### D10 舊資源在切換後觀察 7 天才刪

7 天包含一個週末的流量高峰。要刪的清單：
- `outdoorsafetylab` 的三個服務、三個 trigger、AR 倉庫、sink 與 dataset、四個 collection、
  `rudymap-aws-*` 兩個 secret。
- 舊 root 裡的全部資源：WIF pool、服務帳號、`statuspage-key`、job、Scheduler。
- 舊 state bucket。

## 步驟

每一步都能單獨驗證、單獨回滾。到第 4 步之前，對外完全沒有變化。

| 步驟 | 內容 | 驗證 | 回滾 |
|---|---|---|---|
| 0 | D6 探測 | 兩種做法各自的中斷秒數 | 刪掉拋棄式子網域 |
| 1 | D2 改名舊 root、新 root bootstrap（Director 建 state bucket＋本機 apply）：GitOps 身分、Firestore、AR、服務帳號、secret 容器、BigQuery dataset 與 sink、job（關閉） | 第二次 plan 為空 | destroy 新 root（Firestore 位置除外，見 D3） |
| 2 | `rudymap` 建 trigger（手動），部署三個服務，不掛網域 | run.app 網址回 302、`/v1/apps` 200 | 刪服務 |
| 3 | D4 複製 Firestore、D5 複製歷史表 | run.app 上 `/v1/stats` 有歷史；鹽的文件與舊庫相同 | 重新匯入 |
| 4 | 切 `rudy.oslab.tw`（D6） | 302 指向的鏡像與舊服務相同；記錄實際中斷時間 | 改掛回舊服務 |
| 5 | 離峰切 `rudymap.tw`；D9 切健康檢查；D4、D5 補差額 | 同上，加上新專案 log 與 BigQuery 有新列 | 改掛回舊服務、健康檢查開關改回 |
| 6 | 季報改讀 `rudymap` | 用 2026-Q3 重算，數字與已發布的季報相同 | revert 季報 PR |
| 7 | 觀察 7 天後刪舊（D10） | `outdoorsafetylab` 裡找不到分流器的資源 | 無（刪除不可逆，所以要等 7 天） |

## 測試策略

這次搬的是資源和資料，不是程式碼，所以要抓的是「搬錯了」和「少搬了」：

- **鹽：** 新舊資料庫本季的鹽文件逐位元相同。不同的話，同季的 client id 會斷開，而且單元測試抓不到。
- **轉址：** 同一個檔案從新舊服務各要一次，302 的目的地要相同，否則就是站台狀態或設定沒搬對。
- **統計：** 用已發布的 2026-Q3 季報當基準重算一次，只要有不同，就代表歷史或差額漏了。
- **Terraform：** 每個 PR 的 plan 留言要跟 PR 描述的預期一致；bootstrap 之後的第二次 plan 必須為空。

## 變更歷程

- r1（2026-10-19）：初版。
