# Rudy Balancer

魯地圖下載分流器:

1. [下載主頁](https://rudy.outdoorsafetylab.org/)
   1. [OruxMaps傳送門](https://rudy.outdoorsafetylab.org/oruxmaps)
   1. [綠野遊蹤傳送門](https://rudy.outdoorsafetylab.org/gts)
   1. [Cartograph Pro 2傳送門](https://rudy.outdoorsafetylab.org/carto)
1. [監控主頁](https://outdoorsafetylab1.statuspage.io/)

## 運作方式

### 定時測試

1. 每五分鐘對各 mirror 站台進行 HTTP HEAD 測試, 超過5秒沒有回應則為逾時。
   1. 所有連結都有回應時則判定為 Operational
   1. 部份連結沒有回應時則判定為 Partial outage
   1. 所有連結都無回應時則判定為 Major outage
   1. 每次成功的測試都會紀錄 Latency，並與已儲存的 Latency 值平均後再存回資料庫。
1. 由 Operational 轉為其它狀態時會自動建立 Incident，例如：[Rex is not operational](https://outdoorsafetylab1.statuspage.io/incidents/lghlzv7h9ztq)
1. 由其它狀態回覆為 Operational 後即會自動 Resolve Incident。

> TODO
> 檢查 Last-Modifed，若日期太過老舊則視為 Degraded performance 或 Partial outage。

### 分流方式

1. 分流前會排除 HTTP HEAD 測試失敗的連結。
1. 分流前會以 Latency 計算權重，較低的 Latency 會分配較高的權重。
1. 最後會以亂數選擇要使用的連結，權重較高的連結獲選率也較高。
1. 選定連結後會以 HTTP 302 重新導向。

測試方式：

```shell
$ curl -I https://rudy.outdoorsafetylab.org/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip
HTTP/2 302 
content-type: text/html; charset=utf-8
location: https://moi.kcwu.csie.org/MOI_OSM_Taiwan_TOPO_Rudy.zip
x-cloud-trace-context: 70b67fd60f7f9fe2c9ed2b726465de28
date: Sun, 24 Jul 2022 09:38:23 GMT
server: Google Frontend

$ curl -I https://rudy.outdoorsafetylab.org/v1/MOI_OSM_Taiwan_TOPO_Rudy.zip

HTTP/2 302 
content-type: text/html; charset=utf-8
location: https://map.happyman.idv.tw/rudy/MOI_OSM_Taiwan_TOPO_Rudy.zip
x-cloud-trace-context: 450ca45fdabf0ab2b5872b8a6fedb49d
date: Sun, 24 Jul 2022 09:38:25 GMT
server: Google Frontend
```
