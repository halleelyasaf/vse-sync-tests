#!/usr/bin/env python3

### SPDX-License-Identifier: GPL-2.0-only

"""A reference implementation for tests under:



Use a symbolic link to specify this file as the reference implementation for a test.
"""

import sys
from os.path import join as joinpath, dirname, realpath
import os

# Find tests/common directory by walking up the directory tree
current_dir = dirname(realpath(__file__))
while True:
    test_common = joinpath(current_dir, 'common')
    if os.path.exists(test_common) and os.path.basename(dirname(test_common)) == 'tests':
        sys.path.insert(0, test_common)
        break
    parent_dir = dirname(current_dir)
    if parent_dir == current_dir:
        raise ImportError(f"Unable to locate tests/common from {__file__}")
    current_dir = parent_dir

from vse_sync_pp.parsers.ptp4l import TimeErrorParser
from vse_sync_pp.analyzers.ptp4l import TimeErrorAnalyzer
from generic_test_runner import create_test_implementation

refimpl, main = create_test_implementation(
    __file__,
    parser_class=TimeErrorParser,
    analyzer_class=TimeErrorAnalyzer
)

if __name__ == '__main__':
    main()
