#!/usr/bin/env python3
"""Check a Rudy Map mirror the way the balancer's health check sees it.

Usage:
    python3 mirror_doctor.py https://your.mirror/optional-prefix

Only the Python 3 standard library is used: no pip, no virtualenv.

The list of files and what the other mirrors serve come from the balancer
(https://rudymap.tw/v1/files, or the backup entry rudy.oslab.tw). For every
file the balancer redirects to, the mirror is checked for:

  - HEAD answers 200 (redirects are followed, as the health check does);
  - the size matches the newest copy on the healthy mirrors;
  - the file is not older than that copy (x-amz-meta-mtime is preferred to
    Last-Modified, as object storage sets Last-Modified to the upload time);
  - a strong ETag;
  - Range answers 206;
  - web pages may read it: HEAD with "Origin: null" answers
    "Access-Control-Allow-Origin: *", and a preflight answers 2xx with the
    same origin and Range allowed.

Like the health check, each request has 5 seconds in all (redirects
included) and a file that fails is tried once more 3 seconds later. The
newest copy is taken among the healthy mirrors (the upstream left out, as
the balancer does); the balancer also leaves out a mirror over its monthly
quota, which /v1/files does not show. Within one connection, 5 seconds bounds
each socket operation rather than the whole request.

The exit status is 0 when nothing failed. Warnings (no strong ETag, no CORS)
do not fail the run: a mirror without CORS still serves apps.
"""
import argparse
import concurrent.futures
import email.utils
import http.client
import json
import sys
import time
import urllib.parse

ENTRIES = ("https://rudymap.tw", "https://rudy.oslab.tw")
USER_AGENT = "rudy-mirror-doctor/1.0 (+https://github.com/outdoorsafetylab/rudy-balancer)"
GOOD = 1  # model.Source status for a file the health check could fetch


# ---------------------------------------------------------------- pure helpers

def reference(sources):
    """The newest copy among the healthy mirrors: (modified, size), or None."""
    # the upstream is where the mirrors sync from, not a mirror: the balancer
    # leaves it out when it decides which mirror is lagging
    good = [s for s in sources
            if s.get("Status") == GOOD and s.get("LastModified") and not s.get("Upstream")]
    if not good:
        return None
    newest = max(good, key=lambda s: s["LastModified"])
    return newest["LastModified"], newest.get("Size")


def modified(headers):
    """The source file's modification time in Unix seconds, or None."""
    mtime = headers.get("x-amz-meta-mtime")
    if mtime:
        # whole seconds, as the health check reads it: float() would round
        # .999999999 up and make this mirror look a second newer
        try:
            sec = int(mtime.split(".", 1)[0])
            if sec > 0:
                return sec
        except ValueError:
            pass
    lm = headers.get("last-modified")
    if lm:
        try:
            return int(email.utils.parsedate_to_datetime(lm).timestamp())
        except (TypeError, ValueError):
            pass
    return None


def allows_range(values):
    """Whether Access-Control-Allow-Headers values allow the Range header."""
    for v in values:
        for h in v.split(","):
            h = h.strip().lower()
            if h in ("range", "*"):
                return True
    return False


def evaluate(ref, head, range_status, cors_head, preflight):
    """Problems for one file: a list of (level, message), level FAIL or WARN.

    head: (status, headers) of the HEAD, headers lower-cased; None if unreachable.
    range_status: status of the Range request, or None.
    cors_head: headers of the HEAD with Origin: null, or None.
    preflight: (status, headers, allow_headers_list) of the OPTIONS, or None.
    """
    problems = []
    if head is None:
        return [("FAIL", "連不上")]
    status, headers = head
    if status != 200:
        return [("FAIL", "HEAD 回 %d" % status)]
    if ref:
        ref_mod, ref_size = ref
        size = headers.get("content-length")
        if ref_size and size and int(size) != ref_size:
            problems.append(("FAIL", "大小 %s，其他站最新的是 %d" % (size, ref_size)))
        mod = modified(headers)
        if mod is None:
            problems.append(("WARN", "沒有 Last-Modified，分流器無法判斷是不是舊版"))
        elif mod < ref_mod:
            problems.append(("FAIL", "比其他站舊 %d 小時 (還在同步？)" % ((ref_mod - mod) // 3600)))
    etag = headers.get("etag", "")
    if not etag:
        problems.append(("WARN", "沒有 ETag"))
    elif etag.startswith("W/"):
        problems.append(("WARN", "ETag 是弱 ETag，續傳無法用 If-Range"))
    if range_status != 206:
        problems.append(("FAIL", "Range 回 %s，不是 206" % range_status))
    if cors_head is None or cors_head.get("access-control-allow-origin") != "*":
        problems.append(("WARN", "沒開 CORS (帶 Origin: null 沒回 *)，分不到網頁的下載"))
    elif preflight is None or not (200 <= preflight[0] < 300) \
            or preflight[1].get("access-control-allow-origin") != "*" \
            or not allows_range(preflight[2]):
        problems.append(("WARN", "預檢 (OPTIONS) 沒有回 2xx、* 並允許 Range，分不到網頁的下載"))
    return problems


# ---------------------------------------------------------------- network

def request(method, url, headers=None, timeout=5, hops=5):
    """(status, lower-cased headers, Access-Control-Allow-Headers values), following
    redirects with the same method, all within timeout seconds as the health
    check's client does. Reads no body beyond what a HEAD returns."""
    deadline = time.monotonic() + timeout
    for _ in range(hops + 1):
        left = deadline - time.monotonic()
        if left <= 0:
            raise TimeoutError("timed out")
        u = urllib.parse.urlsplit(url)
        conn_cls = http.client.HTTPSConnection if u.scheme == "https" else http.client.HTTPConnection
        conn = conn_cls(u.netloc, timeout=left)
        path = u.path or "/"
        if u.query:
            path += "?" + u.query
        h = {"User-Agent": USER_AGENT}
        h.update(headers or {})
        try:
            conn.request(method, path, headers=h)
            res = conn.getresponse()
            status = res.status
            lower = {k.lower(): v for k, v in res.getheaders()}
            allow = res.headers.get_all("Access-Control-Allow-Headers") or []
            location = res.getheader("Location")
            if method == "GET":
                res.read(1)
        finally:
            conn.close()
        if status in (301, 302, 303, 307, 308) and location:
            url = urllib.parse.urljoin(url, location)
            continue
        return status, lower, allow
    raise RuntimeError("too many redirects")


RETRY_DELAY = 3  # seconds, as healthcheck.retryDelay


def check(base, name, ref, timeout):
    url = base.rstrip("/") + "/" + urllib.parse.quote(name)
    head = None
    for attempt in range(2):  # a failed file is tried once more, as the health check does
        if attempt:
            time.sleep(RETRY_DELAY)
        try:
            status, headers, _ = request("HEAD", url, timeout=timeout)
            head = (status, headers)
        except Exception:
            head = None
        if head and head[0] == 200:
            break
    if head is None:
        return name, evaluate(ref, None, None, None, None)
    status = head[0]
    if status != 200:
        return name, evaluate(ref, head, None, None, None)
    try:
        range_status = request("GET", url, {"Range": "bytes=0-0"}, timeout=timeout)[0]
    except Exception:
        range_status = None
    try:
        cors_head = request("HEAD", url, {"Origin": "null"}, timeout=timeout)[1]
    except Exception:
        cors_head = None
    try:
        preflight = request("OPTIONS", url, {
            "Origin": "null",
            "Access-Control-Request-Method": "GET",
            "Access-Control-Request-Headers": "range",
        }, timeout=timeout)
    except Exception:
        preflight = None
    return name, evaluate(ref, head, range_status, cors_head, preflight)


def files(timeout):
    last = None
    for entry in ENTRIES:
        try:
            u = urllib.parse.urlsplit(entry + "/v1/files")
            conn = http.client.HTTPSConnection(u.netloc, timeout=timeout)
            conn.request("GET", u.path, headers={"User-Agent": USER_AGENT})
            res = conn.getresponse()
            body = res.read()
            conn.close()
            if res.status == 200:
                return json.loads(body)
            last = "%s: %d" % (entry, res.status)
        except Exception as e:  # try the backup entry
            last = "%s: %s" % (entry, e)
    raise SystemExit("讀不到分流器的檔案清單 (%s)" % last)


GUIDE = "https://rudymap.tw/deployment#mirrors"
LIST_UP_TO = 5  # a warning shared by more files than this shows a count, not the names


def report(results, verbose=False):
    """The lines to print for [(name, problems)]. Failures are listed file by file, as each
    needs fixing; a warning shared by several files is one line with how many (and which,
    when few). verbose lists every file and problem, as before."""
    lines = []
    failed = warned = 0
    warnings = {}  # message -> [file names], in order
    for name, problems in results:
        levels = {lv for lv, _ in problems}
        mark = "FAIL" if "FAIL" in levels else "WARN" if levels else "OK  "
        failed += mark == "FAIL"
        warned += mark == "WARN"
        if verbose:
            lines.append("%s %s" % (mark, name))
            lines.extend("       %s %s" % (lv, msg) for lv, msg in problems)
            continue
        if mark == "FAIL":
            lines.append("FAIL %s" % name)
            lines.extend("       %s %s" % (lv, msg) for lv, msg in problems if lv == "FAIL")
        for lv, msg in problems:
            if lv == "WARN":
                warnings.setdefault(msg, []).append(name)
    if not verbose and warnings:
        if lines:
            lines.append("")
        for msg, names in warnings.items():
            if len(names) == len(results):
                which = "全部 %d 個檔" % len(names)
            elif len(names) <= LIST_UP_TO:
                which = "、".join(names)
            else:
                which = "%d 個檔" % len(names)
            lines.append("WARN %s：%s" % (which, msg))
        lines.append("     設定方式見 %s；加 --verbose 看每個檔" % GUIDE)
    lines.append("")
    lines.append("%d 個檔：%d 正常、%d 有警告、%d 失敗" % (len(results), len(results) - failed - warned, warned, failed))
    return lines


def main(argv=None):
    ap = argparse.ArgumentParser(description="檢查魯地圖鏡像站 (只需要 python3)")
    ap.add_argument("base", help="鏡像站網址，含路徑前綴，例如 https://example.org/rudy")
    ap.add_argument("--timeout", type=float, default=5, help="每個請求的逾時秒數，含轉址 (預設 5，與健康檢查相同)")
    ap.add_argument("--jobs", type=int, default=4, help="同時檢查幾個檔 (預設 4)")
    ap.add_argument("--verbose", "-v", action="store_true", help="逐檔列出每個問題 (預設把相同的警告合併成一行)")
    args = ap.parse_args(argv)

    listing = files(args.timeout)
    names = sorted(listing)
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as pool:
        results = list(pool.map(lambda n: check(args.base, n, reference(listing[n]), args.timeout), names))

    for line in report(results, args.verbose):
        print(line)
    return 1 if any(lv == "FAIL" for _, problems in results for lv, _ in problems) else 0


if __name__ == "__main__":
    sys.exit(main())
