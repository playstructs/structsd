"""Validate local fixture timing before any chain commands execute."""
import os
from pathlib import Path
import re
import subprocess
import unittest


SOURCE = Path(__file__).with_name('test_chain.sh').read_text()
CONFIGURE = re.search(r'(?ms)^configure_fixture_timing\(\) \{.*?^\}', SOURCE).group()
ENVIRONMENT_KEYS = ('STRUCTS_TEST_SETTLE_SECONDS', 'STRUCTS_TEST_LONG_SETTLE_SECONDS', 'STRUCTS_TEST_DURATION_MULTIPLIER')


def invoke(**values):
    environment = {key: value for key, value in os.environ.items() if key not in ENVIRONMENT_KEYS}
    environment.update(values)
    assignments = '\n'.join(re.search(r'(?m)^\s*' + name + r'=.*$', SOURCE).group().strip()
        for name in ('AGREE_DURATION', 'SETTLE_DUR_A', 'SETTLE_DUR_B', 'SETTLE_DUR_C'))
    script = 'set -euo pipefail\n' + CONFIGURE + '\nconfigure_fixture_timing\n' + assignments
    script += '\nprintf "%s\\n" "$SLEEP" "$BIGGER_SLEEP" "$AGREE_DURATION" "$SETTLE_DUR_A" "$SETTLE_DUR_B" "$SETTLE_DUR_C"\n'
    return subprocess.run(['bash', '-c', script], env=environment, capture_output=True, text=True)


class FixtureTimingTests(unittest.TestCase):
    def test_defaults_and_fast_block_quotes_preserve_expiry_control(self):
        result = invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), ['2', '15', '40', '120', '120', '5'])
        result = invoke(STRUCTS_TEST_SETTLE_SECONDS='0.2', STRUCTS_TEST_LONG_SETTLE_SECONDS='1.5', STRUCTS_TEST_DURATION_MULTIPLIER='10')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), ['0.2', '1.5', '400', '1200', '1200', '5'])

    def test_invalid_configuration_refuses_before_callers_continue(self):
        for key in ENVIRONMENT_KEYS[:2]:
            for value in ('0', '-1', '01', '0.001', '61', 'NaN', 'Infinity', '1e2', '$(false)', '1;exit 0'):
                result = invoke(**{key: value})
                self.assertNotEqual(result.returncode, 0, (key, value))
                self.assertEqual(result.stdout, '')
        for value in ('0', '-1', '01', '101', '1.5', '9223372036854775807'):
            result = invoke(STRUCTS_TEST_DURATION_MULTIPLIER=value)
            self.assertNotEqual(result.returncode, 0, value)
            self.assertEqual(result.stdout, '')
        self.assertLess(SOURCE.index('\nconfigure_fixture_timing\n'), SOURCE.index('\nPARAMS_TX='))


if __name__ == '__main__':
    unittest.main()
