#!/usr/bin/env python3

### SPDX-License-Identifier: GPL-2.0-only

"""Test implementation using generic test runner framework.

sync/G.8272/wander-TDEV-in-locked-mode/DPLL-to-PHC/testimpl.py
"""

import sys
from os.path import join as joinpath, dirname, abspath
import os

# Import generic runner framework
# Walk up to find tests/common directory
current_dir = dirname(abspath(__file__))
while current_dir and current_dir != '/':
    test_common = joinpath(current_dir, 'common')
    if os.path.exists(test_common) and os.path.basename(dirname(test_common)) == 'tests':
        sys.path.insert(0, test_common)
        break
    current_dir = dirname(current_dir)

from generic_test_runner import create_test_implementation

# Pass __file__ as-is (works correctly with symlinks for config.yaml)
refimpl, main = create_test_implementation(__file__)

if __name__ == '__main__':
    main()
