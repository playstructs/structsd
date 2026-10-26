import copy
import json
import unittest

from verify_upgrade_grid import verify_grid


def event(kind, key, value, mode=None):
    attributes = [{"key": key, "value": json.dumps(value)}]
    if mode:
        attributes.append({"key": "mode", "value": mode})
    return {"type": "structs.structs.Event" + kind, "attributes": attributes}


class GridReplayTests(unittest.TestCase):
    def fixture(self):
        old = {"destinationId": "3-1", "address": "source", "fuel": "1"}
        new = {**old, "address": "vault"}
        before = {"initial_height": "7", "app_state": {"structs": {
            "infusionList": [old], "gridList": [{"attributeId": "2-1-1", "value": "5"}]}}}
        after = copy.deepcopy(before)
        after["initial_height"] = "8"
        after["app_state"]["structs"] = {"infusionList": [new], "gridList": []}
        events = [event("Infusion", "infusion", new),
            event("Delete", "objectId", "3-1-source", "EndBlock"),
            event("Grid", "gridRecord", {"attributeId": "2-1-1", "value": "0"}, "EndBlock")]
        return before, after, [{"height": "7", "finalize_block_events": events}]

    def test_replays_complete_rows_and_zero_grid_removal(self):
        result = verify_grid(*self.fixture())
        self.assertEqual(result["differences"], [])
        self.assertEqual(result["event_counts"], {"infusion": 1, "grid": 1, "infusion_deletion": 1})

    def test_new_protobuf_defaults_do_not_require_unchanged_generator_events(self):
        before, after, blocks = self.fixture()
        old = {"destinationId": "5-7", "address": "unchanged", "fuel": "42"}
        before["app_state"]["structs"]["infusionList"].append(old)
        new = dict(old, ownerId="", generatorDisabled=False)
        after["app_state"]["structs"]["infusionList"].append(new)
        self.assertEqual([], verify_grid(before, after, blocks)["differences"])
        for field, value in (("ownerId", "0-1"), ("generatorDisabled", True)):
            with self.subTest(field=field):
                changed = copy.deepcopy(after)
                changed["app_state"]["structs"]["infusionList"][-1][field] = value
                self.assertTrue(verify_grid(before, changed, blocks)["differences"])

    def test_each_missing_event_is_detected(self):
        for index in range(3):
            before, after, blocks = self.fixture()
            blocks[0]["finalize_block_events"].pop(index)
            self.assertTrue(verify_grid(before, after, blocks)["differences"])

    def test_changed_fuel_and_large_grid_integer(self):
        before, after, blocks = self.fixture()
        after["app_state"]["structs"]["infusionList"][0]["fuel"] = "2"
        self.assertTrue(verify_grid(before, after, blocks)["differences"])
        before, after, blocks = self.fixture()
        row = {"attributeId": "2-1-2", "value": "9007199254740993"}
        after["app_state"]["structs"]["gridList"].append(row)
        blocks[0]["finalize_block_events"].append(event("Grid", "gridRecord", row))
        self.assertEqual(verify_grid(before, after, blocks)["differences"], [])

    def test_incomplete_boundary_and_duplicate_attributes_rejected(self):
        before, after, blocks = self.fixture()
        with self.assertRaisesRegex(ValueError, "boundary"):
            verify_grid(before, after, [])
        attributes = blocks[0]["finalize_block_events"][0]["attributes"]
        attributes.append(attributes[0])
        with self.assertRaisesRegex(ValueError, "duplicate"):
            verify_grid(before, after, blocks)



if __name__ == "__main__":
    unittest.main()
