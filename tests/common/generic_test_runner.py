#!/usr/bin/env python3

### SPDX-License-Identifier: GPL-2.0-only

"""Generic test runner framework for VSE Sync Tests.

This module eliminates code duplication across test implementations by providing
a single generic test runner that dynamically loads parser/analyzer classes.

Usage:
    In a testimpl.py file:

    from vse_sync_pp.parsers.dpll import TimeErrorParser
    from vse_sync_pp.analyzers.ppsdpll import TimeErrorAnalyzer
    from generic_test_runner import create_test_implementation

    refimpl, main = create_test_implementation(
        __file__,
        parser_class=TimeErrorParser,
        analyzer_class=TimeErrorAnalyzer
    )

    if __name__ == '__main__':
        main()
"""

import sys
import inspect
from argparse import ArgumentParser
from os.path import join as joinpath, dirname

import yaml


def _get_display_name(config_path):
    """Read display_name from YAML config file."""
    with open(config_path, encoding='utf-8') as fid:
        return yaml.safe_load(fid).get('display_name', '')


def parser_accepts_interface(Parser):
    """Check if Parser.__init__ accepts an interface parameter."""
    try:
        sig = inspect.signature(Parser.__init__)
        params = [p for p in sig.parameters.keys() if p != 'self']
        return len(params) > 0
    except Exception:
        return False


def uses_canonical_method(Parser):
    """Check if Parser uses canonical() method instead of parse().

    Based on parser module - dpll, pmc, gnss use canonical().
    """
    module = Parser.__module__
    parser_name = module.split('.')[-1] if '.' in module else module
    return parser_name in {'dpll', 'pmc', 'gnss'}


def parser_needs_interface_cli(Parser):
    """Check if this parser type needs interface in CLI arguments.

    This is based on how e2e.sh calls the test, not Parser.__init__ signature.
    Only ts2phc tests get interface from CLI arguments.
    ptp4l tests pass empty string "" to refimpl (no CLI argument).
    phc2sys tests don't use interface at all.
    DPLL tests are NOT called with interface even though Parser accepts it.
    """
    module = Parser.__module__
    parser_name = module.split('.')[-1] if '.' in module else module
    return parser_name == 'ts2phc'


def create_refimpl(Parser, Analyzer, config_path):
    """Create refimpl function - same signature for all tests.
    
    Runtime inspection determines how to initialize Parser and Analyzer.
    
    Args:
        Parser: Parser class
        Analyzer: Analyzer class
        config_path: Path to config.yaml
        
    Returns:
        function: refimpl(filename, interface=None, encoding='utf-8')
    """
    use_canonical = uses_canonical_method(Parser)
    parser_takes_interface = parser_accepts_interface(Parser)
    
    def refimpl(filename, interface=None, encoding='utf-8'):
        """Execute test and return results."""
        from vse_sync_pp.common import open_input
        from vse_sync_pp.analyzers.analyzer import Config
        
        # Initialize parser - use interface if parser accepts it and it's provided
        if parser_takes_interface and interface is not None:
            parser = Parser(interface)
        else:
            parser = Parser()
        
        # All analyzers get config
        analyzer = Analyzer(Config.from_yaml(config_path))
        
        # Parse and collect data
        with open_input(filename, encoding=encoding) as fid:
            if use_canonical:
                for parsed in parser.canonical(fid):
                    analyzer.collect(parsed)
            else:
                analyzer.collect(*parser.parse(fid))
        
        return {
            'result': analyzer.result,
            'reason': analyzer.reason,
            'timestamp': analyzer.timestamp,
            'duration': analyzer.duration,
            'analysis': analyzer.analysis,
            'pdf_display_name': _get_display_name(config_path),
        }
    
    return refimpl


def create_main(refimpl, Parser):
    """Create main function with CLI argument parsing.

    Args:
        refimpl: The refimpl function to call
        Parser: Parser class (to determine if CLI needs interface)

    Returns:
        function: main function
    """
    needs_interface_cli = parser_needs_interface_cli(Parser)
    parser_takes_interface = parser_accepts_interface(Parser)

    # Determine module for special cases
    module = Parser.__module__
    parser_name = module.split('.')[-1] if '.' in module else module

    def main():
        """Run this test and print test output as JSON to stdout"""
        from vse_sync_pp.common import print_loj

        aparser = ArgumentParser(description=main.__doc__)
        aparser.add_argument('input', help="log file to analyze")

        if needs_interface_cli:
            # ts2phc: get interface from CLI args
            aparser.add_argument('interface', nargs='+',
                               help="interface identifier(s) to capture")
            args = aparser.parse_args()
            output = refimpl(args.input, interface=args.interface)
        elif parser_takes_interface and parser_name == 'ptp4l':
            # ptp4l: pass empty string to refimpl (no CLI arg)
            args = aparser.parse_args()
            output = refimpl(args.input, interface="")
        else:
            # phc2sys, dpll, pmc, gnss: no interface at all
            args = aparser.parse_args()
            output = refimpl(args.input)

        if not print_loj(output):
            sys.exit(1)

    return main


def create_test_implementation(testimpl_file_path, parser_class, analyzer_class):
    """Create refimpl and main functions from parser and analyzer classes.

    This is the main entry point for test implementations.

    Args:
        testimpl_file_path: __file__ from the testimpl.py
        parser_class: Parser class (e.g., TimeErrorParser)
        analyzer_class: Analyzer class (e.g., TimeErrorAnalyzer)

    Returns:
        tuple: (refimpl_function, main_function)

    Example:
        from vse_sync_pp.parsers.dpll import TimeErrorParser
        from vse_sync_pp.analyzers.ppsdpll import TimeErrorAnalyzer
        from generic_test_runner import create_test_implementation

        refimpl, main = create_test_implementation(
            __file__,
            parser_class=TimeErrorParser,
            analyzer_class=TimeErrorAnalyzer
        )

        if __name__ == '__main__':
            main()
    """
    # Use original __file__ for config.yaml (not realpath - for symlink support)
    config_path = joinpath(dirname(testimpl_file_path), 'config.yaml')
    
    refimpl = create_refimpl(parser_class, analyzer_class, config_path)
    main = create_main(refimpl, parser_class)
    
    return refimpl, main
