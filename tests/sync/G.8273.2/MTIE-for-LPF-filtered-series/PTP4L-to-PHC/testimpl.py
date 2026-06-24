#!/usr/bin/env python3

### SPDX-License-Identifier: GPL-2.0-only

"""A reference implementation for tests under:



Use a symbolic link to specify this file as the reference implementation for a test.
"""

from vse_sync_pp.parsers.ptp4l import TimeErrorParser
from vse_sync_pp.analyzers.ptp4l import MaxTimeIntervalErrorAnalyzer
from generic_test_runner import create_test_implementation

refimpl, main = create_test_implementation(
    __file__,
    parser_class=TimeErrorParser,
    analyzer_class=MaxTimeIntervalErrorAnalyzer
)

if __name__ == '__main__':
    main()
