# Rudy Balancer

魯地圖分流器

1. [魯地圖說明網頁](https://rudymap.tw/)
1. [App 更新主頁](https://rudymap.tw/app)
   1. [OruxMaps 傳送門](https://rudymap.tw/app/oruxmaps)
   1. [綠野遊蹤傳送門](https://rudymap.tw/app/gts)
   1. [Cartograph Pro 2 傳送門](https://rudymap.tw/app/carto)
1. [監控主頁](https://rudymap.statuspage.io/)

## 技術規格

- **程式語言**: Go 1.24
- **主要框架**: Gorilla Mux, Cobra CLI, Viper 配置管理
- **資料庫**: Google Cloud Firestore
- **部署平台**: Google Cloud Run
- **監控服務**: StatusPage API

## 運作方式

- 站台設定檔: [mirrors.yaml](https://github.com/outdoorsafetylab/rudy-balancer/blob/master/config/mirrors.yaml)

### 魯地圖說明網頁

1. 使用反向代理 ([Reverse Proxy](https://pkg.go.dev/net/http/httputil#ReverseProxy)) 方式提供主頁 HTML 及素材等，亦即流量會經過分流器端點。
1. 當收到 HTTP 請求時會即時對所有 mirror 站台同步進行 HTTP HEAD 測試，並以最快回應的伺服器作為來源進行反向代理。
1. 若 HTTP 請求為[正面表列的圖資檔](https://github.com/outdoorsafetylab/rudy-balancer/blob/master/config/mirrors.yaml)，則會改以 HTTP 302 重新導向，流量不會經過分流器端點。
1. 若所有 mirror 都無法在3秒內回應，則會回覆 HTTP 504 Gateway Timeout 錯誤。

### 圖資定時測試

1. 每30分鐘對各 mirror 站台進行 HTTP HEAD 測試, 超過5秒沒有回應則為逾時。
   1. 會對所有正面表列的圖資檔進行測試。
   1. 所有連結都有回應時則判定為 Operational
   1. 部份連結沒有回應時則判定為 Partial outage
   1. 所有連結都無回應時則判定為 Major outage
   1. 每次成功的測試都會紀錄 Latency，並與已儲存的 Latency 值平均後再存回資料庫。
   1. 使用 [Statuspage API](https://developer.statuspage.io/) 來更新狀態。
1. 由 Operational 轉為其它狀態時會自動建立 Incident，例如：[Rex is not operational](https://rudymap.statuspage.io/incidents/blp2ytvrjg05)
1. 由其它狀態回覆為 Operational 後即會自動 Resolve Incident。
1. 定期測試的端點部署於 Google Cloud Platform `asia-east1` (彰化)

### 入口網站健康檢查

1. 新增對入口網站 (Portal Sites) 的健康檢查功能
1. 檢查各入口網站的關鍵資源 (如 CSS、JavaScript、圖片等) 是否正常運作
1. 根據資源可用性百分比自動更新 StatusPage 組件狀態
1. 支援動態建立 StatusPage 組件，無需手動配置

> **TODO**
>
> 1. 檢查 Last-Modifed，若日期太過老舊則視為 Degraded performance 或 Partial outage。
> 1. 老舊可依各別檔案定義，例如 Daily Build 可設定為 1 天， Weekly Build 可設定為 7 天，DEM 檔可設定為 1 年。

### 圖資分流方式

1. 分流前會排除定期測試後失敗的連結。
1. 分流前會使用設定檔內定義的站台權重。
1. 最後會以亂數選擇要使用的連結，權重較高的連結獲選率也較高。
1. 選定連結後會以 HTTP 302 重新導向，流量不會經過分流器端點。

測試方式：

```shell
$ curl -I https://rudymap.tw/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip
HTTP/2 302 
content-type: text/html; charset=utf-8
location: https://moi.kcwu.csie.org/MOI_OSM_Taiwan_TOPO_Rudy.zip
x-cloud-trace-context: 70b67fd60f7f9fe2c9ed2b726465de28
date: Sun, 24 Jul 2022 09:38:23 GMT
server: Google Frontend

$ curl -I https://rudymap.tw/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip

HTTP/2 302 
content-type: text/html; charset=utf-8
location: https://map.happyman.idv.tw/rudy/MOI_OSM_Taiwan_TOPO_Rudy.zip
x-cloud-trace-context: 450ca45fdabf0ab2b5872b8a6fedb49d
date: Sun, 24 Jul 2022 09:38:25 GMT
server: Google Frontend
```

### 自動部署 (Continuous Deployment)

當 git 有以下變動時會觸發自動部署：

- `master` 分支: 部署至 https://rudy-balancer-alpha-mgl7xqygta-de.a.run.app/
  - 背後服務: [Google Cloud Run](https://cloud.google.com/run)
  - 部署地區: `asia-east1` (彰化)
- 建立 tag 時: 部署至 https://rudymap.tw/
  - 背後服務: [Google Cloud Run](https://cloud.google.com/run)
  - 部署地區: `asia-east1` (彰化)

### 本地開發

```shell
# 安裝依賴
go mod tidy

# 本地執行
make serve

# 監控模式 (需要安裝 nodemon)
make watch

# 健康檢查測試
make healthcheck

# 建置
make

# 清理
make clean
```

### Docker 建置

```shell
# 建置映像檔
docker build -t rudy-balancer .

# 執行容器
docker run -p 8080:8080 rudy-balancer
```
