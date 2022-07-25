# Rudy Balancer

魯地圖分流器

1. [魯地圖主頁](https://rudy.outdoors.tw/)
1. [App 更新主頁](https://rudy.outdoors.tw/app)
   1. [OruxMaps 傳送門](https://rudy.outdoors.tw/app/oruxmaps)
   1. [綠野遊蹤傳送門](https://rudy.outdoors.tw/app/gts)
   1. [Cartograph Pro 2 傳送門](https://rudy.outdoors.tw/app/carto)
1. [監控主頁](https://outdoorsafetylab1.statuspage.io/)

## 運作方式

### 魯地圖主頁

固定對 [Happyman](https://map.happyman.idv.tw/rudy/) 進行 Reverse Proxy。因不明原因 Golang 內建的 [Reverse Proxy](https://pkg.go.dev/net/http/httputil#ReverseProxy) 無法導向 [KC Wu](https://moi.kcwu.csie.org/taiwan_topo.html)，root cause 待查。

### 定時測試

1. 每60分鐘對各 mirror 站台進行 HTTP HEAD 測試, 超過5秒沒有回應則為逾時。
   1. 所有連結都有回應時則判定為 Operational
   1. 部份連結沒有回應時則判定為 Partial outage
   1. 所有連結都無回應時則判定為 Major outage
   1. 每次成功的測試都會紀錄 Latency，並與已儲存的 Latency 值平均後再存回資料庫。
   1. 使用 [Statuspage API](https://developer.statuspage.io/) 來更新狀態。
1. 由 Operational 轉為其它狀態時會自動建立 Incident，例如：[Rex is not operational](https://outdoorsafetylab1.statuspage.io/incidents/lghlzv7h9ztq)
1. 由其它狀態回覆為 Operational 後即會自動 Resolve Incident。

> **TODO**
>
> 1. 檢查 Last-Modifed，若日期太過老舊則視為 Degraded performance 或 Partial outage。
> 1. 老舊可依各別檔案定義，例如 Daily Build 可設定為 1 天， Weekly Build 可設定為 7 天，DEM 檔可設定為 1 年。

### 分流方式

1. 分流前會排除 HTTP HEAD 測試失敗的連結。
1. 分流前會使用設定檔內定義的 mirror 站台權重。
1. 最後會以亂數選擇要使用的連結，權重較高的連結獲選率也較高。
1. 選定連結後會以 HTTP 302 重新導向。

測試方式：

```shell
$ curl -I https://rudy.outdoors.tw/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip
HTTP/2 302 
content-type: text/html; charset=utf-8
location: https://moi.kcwu.csie.org/MOI_OSM_Taiwan_TOPO_Rudy.zip
x-cloud-trace-context: 70b67fd60f7f9fe2c9ed2b726465de28
date: Sun, 24 Jul 2022 09:38:23 GMT
server: Google Frontend

$ curl -I https://rudy.outdoors.tw/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip

HTTP/2 302 
content-type: text/html; charset=utf-8
location: https://map.happyman.idv.tw/rudy/MOI_OSM_Taiwan_TOPO_Rudy.zip
x-cloud-trace-context: 450ca45fdabf0ab2b5872b8a6fedb49d
date: Sun, 24 Jul 2022 09:38:25 GMT
server: Google Frontend
```

### 自動部署 (Continuous Deplayment)

當 git 有以下變動時會觸發自動部署：

* `master` 分支: 部署至 https://alpha-rudy.outdoors.tw/
  * 背後服務: [Google Cloud Run](https://cloud.google.com/run)
  * 部署地區: `asia-east1` (彰化)
* 建立 tag 時: 部署至 https://rudy.outdoors.tw/
  * 背後服務: [Google Cloud Run](https://cloud.google.com/run)
  * 部署地區: `asia-east1` (彰化)
