"""Tests for mirror_doctor's judgements: cd tools && python3 -m unittest mirror_doctor_test"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import mirror_doctor as md  # noqa: E402

REF = (1795694628, 1000)  # newest copy among healthy mirrors: 2026-11-26 12:03:48Z, 1000 bytes
LM = "Thu, 26 Nov 2026 12:03:48 GMT"


def head(**h):
    base = {"content-length": "1000", "last-modified": LM, "etag": '"abc"'}
    base.update({k.replace("_", "-"): v for k, v in h.items()})
    return 200, {k: v for k, v in base.items() if v is not None}


CORS = {"access-control-allow-origin": "*"}
PREFLIGHT = (204, {"access-control-allow-origin": "*"}, ["Range, If-Range"])


def levels(problems):
    return sorted(lv for lv, _ in problems)


class Reference(unittest.TestCase):
    def test_newest_healthy_copy(self):
        sources = [
            {"Status": 1, "LastModified": 10, "Size": 1},
            {"Status": 1, "LastModified": 30, "Size": 3},
            {"Status": 2, "LastModified": 99, "Size": 9},  # failed check: ignored
        ]
        self.assertEqual(md.reference(sources), (30, 3))

    def test_upstream_left_out(self):
        sources = [
            {"Status": 1, "LastModified": 10, "Size": 1},
            {"Status": 1, "LastModified": 30, "Size": 3, "Upstream": True},
        ]
        self.assertEqual(md.reference(sources), (10, 1))

    def test_none_healthy(self):
        self.assertIsNone(md.reference([{"Status": 2, "LastModified": 5}]))


class Modified(unittest.TestCase):
    def test_object_storage_mtime_wins(self):
        # S3: Last-Modified is the upload time, x-amz-meta-mtime the source's
        h = {"last-modified": "Thu, 26 Nov 2026 20:56:55 GMT", "x-amz-meta-mtime": "1795694628"}
        self.assertEqual(md.modified(h), 1795694628)

    def test_last_modified(self):
        self.assertEqual(md.modified({"last-modified": LM}), 1795694628)

    def test_missing(self):
        self.assertIsNone(md.modified({}))

    def test_fraction_truncated_not_rounded(self):
        # as model/source.go: .999999999 must not round up to the next second
        self.assertEqual(md.modified({"x-amz-meta-mtime": "1795694628.999999999"}), 1795694628)

    def test_zero_mtime_falls_back(self):
        self.assertEqual(md.modified({"x-amz-meta-mtime": "0", "last-modified": LM}), 1795694628)


class Evaluate(unittest.TestCase):
    def test_healthy(self):
        self.assertEqual(md.evaluate(REF, head(), 206, CORS, PREFLIGHT), [])

    def test_missing_file_fails(self):
        p = md.evaluate(REF, (404, {}), None, None, None)
        self.assertEqual(levels(p), ["FAIL"])
        self.assertIn("404", p[0][1])

    def test_unreachable_fails(self):
        self.assertEqual(levels(md.evaluate(REF, None, None, None, None)), ["FAIL"])

    def test_stale_fails(self):
        p = md.evaluate(REF, head(last_modified="Wed, 19 Nov 2026 12:03:48 GMT"), 206, CORS, PREFLIGHT)
        self.assertEqual(levels(p), ["FAIL"])
        self.assertIn("168", p[0][1])  # a week behind

    def test_s3_upload_time_is_not_stale(self):
        h = head(last_modified="Thu, 26 Nov 2026 20:56:55 GMT", x_amz_meta_mtime="1795694628")
        self.assertEqual(md.evaluate(REF, h, 206, CORS, PREFLIGHT), [])

    def test_size_mismatch_fails(self):
        self.assertEqual(levels(md.evaluate(REF, head(content_length="999"), 206, CORS, PREFLIGHT)), ["FAIL"])

    def test_no_range_fails(self):
        self.assertEqual(levels(md.evaluate(REF, head(), 200, CORS, PREFLIGHT)), ["FAIL"])

    def test_weak_or_missing_etag_warns(self):
        self.assertEqual(levels(md.evaluate(REF, head(etag='W/"abc"'), 206, CORS, PREFLIGHT)), ["WARN"])
        self.assertEqual(levels(md.evaluate(REF, head(etag=None), 206, CORS, PREFLIGHT)), ["WARN"])

    def test_no_cors_warns_not_fails(self):
        self.assertEqual(levels(md.evaluate(REF, head(), 206, {}, PREFLIGHT)), ["WARN"])

    def test_preflight_without_range_warns(self):
        bad = (204, {"access-control-allow-origin": "*"}, ["If-None-Match"])
        self.assertEqual(levels(md.evaluate(REF, head(), 206, CORS, bad)), ["WARN"])
        self.assertEqual(levels(md.evaluate(REF, head(), 206, CORS, (405, {}, []))), ["WARN"])

    def test_wildcard_allow_headers(self):
        ok = (204, {"access-control-allow-origin": "*"}, ["*"])
        self.assertEqual(md.evaluate(REF, head(), 206, CORS, ok), [])


class Report(unittest.TestCase):
    CORS = ("WARN", "沒開 CORS")

    def test_shared_warning_is_one_line(self):
        results = [("a.zip", [self.CORS]), ("b.zip", [self.CORS]), ("c.zip", [self.CORS])]
        lines = md.report(results)
        warn = [l for l in lines if l.startswith("WARN")]
        self.assertEqual(warn, ["WARN 全部 3 個檔：沒開 CORS"])
        self.assertEqual(lines[-1], "3 個檔：0 正常、3 有警告、0 失敗")

    def test_failures_listed_by_file_and_warning_counted(self):
        results = [("ok.zip", []), ("gone.pmtiles", [("FAIL", "HEAD 回 404")])] + \
                  [("f%d.zip" % i, [self.CORS]) for i in range(6)]
        lines = md.report(results)
        self.assertEqual(lines[:2], ["FAIL gone.pmtiles", "       FAIL HEAD 回 404"])
        self.assertIn("WARN 6 個檔：沒開 CORS", lines)
        self.assertEqual(lines[-1], "8 個檔：1 正常、6 有警告、1 失敗")

    def test_few_files_are_named(self):
        results = [("a.zip", [self.CORS]), ("b.zip", [])]
        self.assertIn("WARN a.zip：沒開 CORS", md.report(results))

    def test_verbose_lists_every_file(self):
        results = [("a.zip", [self.CORS]), ("b.zip", [])]
        lines = md.report(results, verbose=True)
        self.assertEqual(lines[:3], ["WARN a.zip", "       WARN 沒開 CORS", "OK   b.zip"])


if __name__ == "__main__":
    unittest.main()
