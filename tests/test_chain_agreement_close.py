"""Closure must prove absence and release state, rather than swallow RPC errors."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_object_missing"))
BLOCK = SOURCE.split("# Verify teardown, including both temporary grants in this small fixture.", 1)[1].split(
    '    else\n        assert_not_empty "Required agreement lifecycle', 1)[0]
BASE = "set -euo pipefail\nGREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_QUERY=''\n"


class AgreementCloseTests(unittest.TestCase):
    def test_absence_requires_failed_query_with_object_not_found(self):
        for output, code, missing in (("Error: rpc error: code = Unknown desc = object not found", 1, True),
                                      ("object not found", 0, False), ("{}", 0, False),
                                      ("", 1, False), ("connection refused", 1, False),
                                      ("context deadline exceeded", 1, False), ("command not found", 1, False),
                                      ('{"Agreement":{"id":"11-1"}}', 0, False)):
            with self.subTest(output=output, code=code):
                script = BASE + 'payload="$1" code="$2"\nstructsd() { printf "%s" "$payload" >&2; return "$code"; }\n'
                script += HELPERS + '\nassert_object_missing removed agreement 11-1\n'
                script += f'[ "$FAIL_COUNT" -eq {0 if missing else 1} ]\n'
                result = subprocess.run(["bash", "-c", script, "test", output, str(code)], capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_settlement_removal_call_sites_require_real_absence(self):
        sites = (("Settlement agreement A removed", "agreement", "11-1"),
                 ("Settlement agreement B removed", "agreement", "11-2"),
                 ("Expired agreement removed by EndBlocker", "agreement", "11-3"),
                 ("Expired guild agreement is removed", "agreement", "11-4"),
                 ("Settlement agreement A allocation removed", "allocation", "6-1"),
                 ("Settlement agreement B allocation removed", "allocation", "6-2"),
                 ("Expired agreement C allocation is removed", "allocation", "6-3"),
                 ("Expired agreement D allocation is removed", "allocation", "6-4"),
                 ("Settlement provider deleted", "provider", "10-1"))
        for label, kind, object_id in sites:
            call = re.search(r'(?m)^\s*assert_object_missing "' + re.escape(label) + r'"[^\n]*$', SOURCE).group()
            for response, status, failures in (("Error: object not found", 1, 0),
                                                ("{}", 0, 1), ("connection refused", 1, 1)):
                with self.subTest(label=label, response=response):
                    script = BASE + r'''
SETTLE_ALLOC_A=6-1 SETTLE_ALLOC_B=6-2 SETTLE_ALLOC_C=6-3 SETTLE_ALLOC_D=6-4
SETTLE_AGREE_A=11-1 SETTLE_AGREE_B=11-2 SETTLE_AGREE_C=11-3 SETTLE_AGREE_D=11-4 SETTLE_PROV_ID=10-1
payload="$1" status="$2" expected="$3"
structsd() {
    [ "$*" = "$expected" ] || return 9
    printf '%s' "$payload" >&2
    return "$status"
}
'''
                    script += HELPERS + "\n" + call + f'\n[ "$FAIL_COUNT" -eq {failures} ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", response, str(status),
                                             f"query structs {kind} {object_id}"], capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_teardown_state_checks(self):
        for fault in ("none", "omitted_zero", "agreement", "allocation", "load", "agreement_grant", "allocation_grant", "rpc", "missing_grid"):
            with self.subTest(fault=fault):
                script = BASE + r'''
fault="$1" AGREE_ID=11-1 AGREE_ALLOC_ID=6-1 PROVIDER_ID=10-1 PLAYER_2_ID=1-2
structsd() {
    if [ "$fault" = rpc ]; then echo 'connection refused' >&2; return 1; fi
    case "$*" in
        'query structs agreement 11-1')
            if [ "$fault" = agreement ]; then echo '{"Agreement":{"id":"11-1"}}'; return; fi ;;
        'query structs allocation 6-1')
            if [ "$fault" = allocation ]; then echo '{"Allocation":{"id":"6-1"}}'; return; fi ;;
        *) return 9 ;;
    esac
    echo 'Error: object not found' >&2; return 1
}
query() {
    [ "$*" = 'query structs provider 10-1' ]
    case "$fault" in
        missing_grid) echo '{}' ;;
        load) echo '{"gridAttributes":{"load":"65000"}}' ;;
        omitted_zero) echo '{"gridAttributes":{}}' ;;
        *) echo '{"gridAttributes":{"load":"0"}}' ;;
    esac
}
get_permission_value_for_player() {
    [ "$2" = 1-2 ]
    if { [ "$fault" = agreement_grant ] && [ "$1" = 11-1 ]; } ||
       { [ "$fault" = allocation_grant ] && [ "$1" = 6-1 ]; }; then echo 4; else echo 0; fi
}
'''
                script += HELPERS + '\n' + BLOCK + '\n[ "$PASS_COUNT" -eq 5 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                result = subprocess.run(["bash", "-c", script, "test", fault], capture_output=True, text=True)
                if fault in ("none", "omitted_zero"):
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                else:
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
