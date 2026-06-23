#!/usr/bin/env python3

### SPDX-License-Identifier: GPL-2.0-only

"""Generic test runner framework for VSE Sync Tests.

This module eliminates code duplication across 66 test implementations by providing
a single generic test runner that dynamically loads parser/analyzer classes based
on test metadata.

Usage:
    In a testimpl.py file:

    from generic_test_runner import create_test_implementation
    refimpl, main = create_test_implementation(__file__)

    if __name__ == '__main__':
        main()
"""

import sys
import importlib
from argparse import ArgumentParser
from os.path import join as joinpath, dirname

import yaml


# Parser modules that use canonical() method
CANONICAL_PARSERS = {'dpll', 'pmc', 'gnss'}

# Analyzers that require config parameter in refimpl signature
CONFIG_REQUIRED_ANALYZERS = {
    'MaxTimeIntervalErrorAnalyzer',
}

# Parser modules that typically need interface parameter
INTERFACE_AWARE_PARSERS = {'ts2phc', 'phc2sys', 'ptp4l'}

# Pattern C parsers that accept interface parameter (ts2phc, phc2sys)
# ptp4l does NOT accept interface parameter
PATTERN_C_INTERFACE_PARSERS = {'ts2phc', 'phc2sys'}


def import_class(module_path, class_name):
    """Dynamically import a class from a module path.

    Args:
        module_path: Full module path (e.g., 'vse_sync_pp.parsers.dpll')
        class_name: Class name to import (e.g., 'TimeErrorParser')

    Returns:
        The imported class
    """
    module = importlib.import_module(module_path)
    return getattr(module, class_name)


def load_test_metadata(testimpl_file_path):
    """Load test metadata from test_metadata.yaml.

    Args:
        testimpl_file_path: Path to the testimpl.py file (__file__)

    Returns:
        dict: Test metadata configuration
    """
    from os.path import realpath
    # Use realpath to resolve symlinks and find the actual testimpl.py location
    # where test_metadata.yaml is stored
    real_path = realpath(testimpl_file_path)
    metadata_path = joinpath(dirname(real_path), 'test_metadata.yaml')
    with open(metadata_path, encoding='utf-8') as fid:
        return yaml.safe_load(fid)


def _get_display_name(config_path):
    """Read display_name from YAML config file."""
    with open(config_path, encoding='utf-8') as fid:
        return yaml.safe_load(fid).get('display_name', '')


def detect_parse_method(metadata):
    """Determine whether to use canonical() or parse() method.

    Args:
        metadata: Test metadata dict

    Returns:
        str: 'canonical' or 'parse'
    """
    # Check if explicitly specified in metadata
    if 'parser' in metadata and 'method' in metadata['parser']:
        return metadata['parser']['method']

    # Auto-detect based on parser module
    parser_module = metadata['parser']['module'].split('.')[-1]  # Get last part (e.g., 'dpll')

    if parser_module in CANONICAL_PARSERS:
        return 'canonical'
    else:
        return 'parse'


def detect_signature_pattern(metadata):
    """Determine the refimpl function signature pattern.

    Patterns:
        A: refimpl(filename, encoding='utf-8')
        B: refimpl(filename, interface=None, encoding='utf-8')
        C: refimpl(filename, config, interface=None, encoding='utf-8')

    Args:
        metadata: Test metadata dict

    Returns:
        str: 'A', 'B', or 'C'
    """
    # Check if explicitly specified in metadata
    if 'signature' in metadata and 'pattern' in metadata['signature']:
        return metadata['signature']['pattern']

    # Auto-detect based on analyzer and parser
    analyzer_class = metadata['analyzer']['class']
    parser_module = metadata['parser']['module'].split('.')[-1]

    # Pattern C: Config required analyzers
    if analyzer_class in CONFIG_REQUIRED_ANALYZERS:
        return 'C'

    # Pattern B: Interface-aware parsers
    if parser_module in INTERFACE_AWARE_PARSERS:
        return 'B'

    # Pattern A: Default
    return 'A'


def _execute_test(Parser, Analyzer, filename, encoding, interface, config, parse_method, config_path):
    """Execute the test with the given parser and analyzer.

    Args:
        Parser: Parser class
        Analyzer: Analyzer class
        filename: Input log file path
        encoding: File encoding
        interface: Optional interface parameter
        config: Optional config parameter
        parse_method: 'canonical' or 'parse'
        config_path: Path to config.yaml

    Returns:
        dict: Test result with keys: result, reason, timestamp, duration, analysis, pdf_display_name
    """
    # Lazy import to avoid unnecessary dependencies at module load time
    from vse_sync_pp.common import open_input, print_loj
    from vse_sync_pp.analyzers.analyzer import Config

    # Initialize parser (with interface if provided)
    if interface is not None:
        parser = Parser(interface)
    else:
        parser = Parser()

    # Initialize analyzer (with config if provided)
    if config is not None:
        # Config parameter is the config file path
        analyzer = Analyzer(Config.from_yaml(config))
    else:
        # Use default config from test directory
        analyzer = Analyzer(Config.from_yaml(config_path))

    # Parse and collect data
    with open_input(filename, encoding=encoding) as fid:
        if parse_method == 'canonical':
            # Loop iteration pattern
            for parsed in parser.canonical(fid):
                analyzer.collect(parsed)
        else:
            # Unpacking pattern
            analyzer.collect(*parser.parse(fid))

    # Return test results
    return {
        'result': analyzer.result,
        'reason': analyzer.reason,
        'timestamp': analyzer.timestamp,
        'duration': analyzer.duration,
        'analysis': analyzer.analysis,
        'pdf_display_name': _get_display_name(config_path),
    }


def create_refimpl_function(Parser, Analyzer, pattern, parse_method, base_path):
    """Create refimpl function with the correct signature.

    Args:
        Parser: Parser class
        Analyzer: Analyzer class
        pattern: Signature pattern ('A', 'B', or 'C')
        parse_method: 'canonical' or 'parse'
        base_path: Path to the testimpl.py file

    Returns:
        function: refimpl function with appropriate signature
    """
    config_path = joinpath(dirname(base_path), 'config.yaml')

    if pattern == 'A':
        def refimpl(filename, encoding='utf-8'):
            return _execute_test(
                Parser, Analyzer, filename, encoding,
                interface=None, config=None, parse_method=parse_method,
                config_path=config_path
            )

    elif pattern == 'B':
        def refimpl(filename, interface=None, encoding='utf-8'):
            return _execute_test(
                Parser, Analyzer, filename, encoding,
                interface=interface, config=None, parse_method=parse_method,
                config_path=config_path
            )

    elif pattern == 'C':
        # Pattern C: Like Pattern B but uses config in analyzer
        # Original behavior: config was hardcoded (CONFIG constant), not a CLI parameter
        def refimpl(filename, interface=None, encoding='utf-8'):
            return _execute_test(
                Parser, Analyzer, filename, encoding,
                interface=interface, config=config_path, parse_method=parse_method,
                config_path=config_path
            )

    else:
        raise ValueError(f"Unknown signature pattern: {pattern}")

    return refimpl


def create_main_function(refimpl, pattern, parser_module=None):
    """Create main function with correct argument parsing.

    Args:
        refimpl: The refimpl function to call
        pattern: Signature pattern ('A', 'B', or 'C')
        parser_module: Parser module name (for Pattern C interface detection)

    Returns:
        function: main function
    """
    def main():
        """Run this test and print test output as JSON to stdout"""
        # Lazy import to avoid unnecessary dependencies
        from vse_sync_pp.common import print_loj

        aparser = ArgumentParser(description=main.__doc__)
        aparser.add_argument('input', help="log file to analyze")

        if pattern == 'B':
            # Pattern B: interface parameter (required, multi-value)
            # Use nargs='+' to require at least one interface (matches original behavior)
            aparser.add_argument('interface', nargs='+', help="interface identifier(s) to capture")
            args = aparser.parse_args()
            output = refimpl(args.input, interface=args.interface)

        elif pattern == 'C':
            # Pattern C: Config required, interface may or may not be needed
            # ts2phc/phc2sys parsers need interface, ptp4l does not
            # Config is always hardcoded, never passed via CLI
            parser_name = parser_module.split('.')[-1] if parser_module else ''
            needs_interface = parser_name in PATTERN_C_INTERFACE_PARSERS

            if needs_interface:
                # Pattern C with interface (ts2phc, phc2sys parsers)
                # Interface is REQUIRED - use nargs='+' (matches original behavior)
                aparser.add_argument('interface', nargs='+', help="interface identifier(s) to capture")
                args = aparser.parse_args()
                output = refimpl(args.input, interface=args.interface)
            else:
                # Pattern C without interface (ptp4l parser)
                args = aparser.parse_args()
                output = refimpl(args.input)

        else:
            # Pattern A: filename only
            args = aparser.parse_args()
            output = refimpl(args.input)

        # Print output and exit appropriately
        if not print_loj(output):
            sys.exit(1)

    return main


def create_test_implementation(testimpl_file_path):
    """Create refimpl and main functions from test metadata.

    This is the main entry point for test implementations using the generic framework.

    Args:
        testimpl_file_path: __file__ from the testimpl.py calling this

    Returns:
        tuple: (refimpl_function, main_function)

    Example:
        In a testimpl.py file:

        from generic_test_runner import create_test_implementation
        refimpl, main = create_test_implementation(__file__)

        if __name__ == '__main__':
            main()
    """
    # Load test metadata
    metadata = load_test_metadata(testimpl_file_path)

    # Import parser and analyzer classes
    Parser = import_class(metadata['parser']['module'], metadata['parser']['class'])
    Analyzer = import_class(metadata['analyzer']['module'], metadata['analyzer']['class'])

    # Detect patterns
    parse_method = detect_parse_method(metadata)
    pattern = detect_signature_pattern(metadata)

    # Create refimpl and main functions
    refimpl = create_refimpl_function(Parser, Analyzer, pattern, parse_method, testimpl_file_path)
    main = create_main_function(refimpl, pattern, parser_module=metadata['parser']['module'])

    return refimpl, main
