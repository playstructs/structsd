"""The bad-rate control must prove the client refusal without global state."""
from pathlib import Path
import subprocess
import unittest


SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
CONTROL = SOURCE.split('# ─── Negative test: bad rate', 1)[1].split('# ─── Good provider creation', 1)[0]
CONTROL = '\n'.join(CONTROL.splitlines()[1:])
REFUSAL = 'invalid argument "1" for "--1" flag: invalid input format'


class ProviderBadRateTests(unittest.TestCase):
    def invoke(self, output):
        script = '''set -euo pipefail
SUBSTATION_ID=4-1 PROVIDER_ACCESS=guild-market
PROVIDER_PROVIDER_PENALTY=0 PROVIDER_CONSUMER_PENALTY=0
PROVIDER_CAP_MIN=100000 PROVIDER_CAP_MAX=5000000 PROVIDER_DUR_MIN=10 PROVIDER_DUR_MAX=10000
response="$1"
run_tx_expect_fail() {
    shift
    [ "$*" = 'tx structs provider-create 4-1 1 guild-market 0 0 100000 5000000 10 10000 --from alice' ]
    LAST_TX_OUTPUT="$response"
}
query() { echo 'UNEXPECTED GLOBAL QUERY'; return 9; }
assert_eq() { [ "$2" = "$3" ]; }
'''
        return subprocess.run(['bash', '-c', script + CONTROL, 'test', output], capture_output=True, text=True)

    def test_exact_cli_refusal_needs_no_global_provider_query(self):
        result = self.invoke(REFUSAL)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, '')

    def test_unrelated_failures_or_success_do_not_count_as_bad_rate_validation(self):
        for output in ('', '{}', '{"code":0,"txhash":"' + 'A' * 64 + '"}',
                'rpc error: connection refused', 'missing permission', 'unknown flag: --1',
                'invalid input format', REFUSAL + '\n{"code":0}'):
            with self.subTest(output=output):
                self.assertNotEqual(self.invoke(output).returncode, 0)


if __name__ == '__main__':
    unittest.main()
