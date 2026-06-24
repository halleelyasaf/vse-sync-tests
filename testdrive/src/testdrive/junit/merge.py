### SPDX-License-Identifier: GPL-2.0-or-later

"""Merge JUnit output"""

from argparse import ArgumentParser
from datetime import timedelta

from xml.etree import ElementTree as ET

from ..run import timevalue


def _indent_xml(elem, level=0):
    """Manual XML indentation for Python 3.6-3.8 compatibility.

    Replicates ET.indent() behavior for older Python versions.
    """
    indent = "\n" + "  " * level
    if len(elem):
        if not elem.text or not elem.text.strip():
            elem.text = indent + "  "
        if not elem.tail or not elem.tail.strip():
            elem.tail = indent
        for child in elem:
            _indent_xml(child, level + 1)
        if not child.tail or not child.tail.strip():
            child.tail = indent
    else:
        if level and (not elem.tail or not elem.tail.strip()):
            elem.tail = indent


def combine(attrs, e_suite):
    """Combine attribute values from `e_suite` into `attrs`."""
    for name in ("tests", "errors", "failures", "skipped"):
        count = int(e_suite.get(name, 0))
        try:
            attrs[name] += count
        except KeyError:
            attrs[name] = count
    timestamp = e_suite.get("timestamp")
    duration = float(e_suite.get("time", 0))
    if timestamp:
        try:
            # time value when [c]ombined testing [b]egan
            tv_cb = timevalue(attrs["timestamp"])
            # time value when [c]ombined testing [f]inished
            tv_cf = tv_cb + timedelta(seconds=attrs["time"])
        except KeyError:
            attrs["timestamp"] = timestamp
            attrs["time"] = duration
        else:
            # time value when this [s]uite [b]egan
            tv_sb = timevalue(timestamp)
            # time value when this [s]uite [f]inished
            tv_sf = tv_sb + timedelta(seconds=duration)
            if tv_sb < tv_cb:
                attrs["timestamp"] = timestamp
                if tv_sf < tv_cf:
                    attrs["time"] += (tv_cb - tv_sb).total_seconds()
                else:
                    attrs["time"] = duration
            elif tv_cf < tv_sf:
                attrs["time"] += (tv_sf - tv_cf).total_seconds()


def main():
    """Merge JUnit files and print the output to stdout."""
    aparser = ArgumentParser(description=main.__doc__)
    aparser.add_argument(
        "--prettify",
        action="store_true",
        help="pretty print XML output",
    )
    aparser.add_argument(
        "inputs",
        nargs="+",
        help="input files",
    )
    args = aparser.parse_args()
    attrs = {}
    e_suites = []
    for filename in args.inputs:
        for e_suite in ET.parse(filename).getroot().iter("testsuite"):
            e_suites.append(e_suite)
            combine(attrs, e_suite)
    e_root = ET.Element("testsuites", {k: str(v) for (k, v) in attrs.items()})
    for e_suite in e_suites:
        e_root.append(e_suite)
    if args.prettify:
        # Python 3.6 compatibility - ET.indent was added in Python 3.9
        if hasattr(ET, 'indent'):
            ET.indent(e_root)
        else:
            _indent_xml(e_root)

    # Python 3.6 compatibility - xml_declaration parameter added in Python 3.8
    try:
        print(ET.tostring(e_root, encoding="unicode", xml_declaration=True))
    except TypeError:
        # Python 3.6/3.7 fallback - manually add XML declaration
        xml_str = ET.tostring(e_root, encoding="unicode")
        print('<?xml version="1.0"?>\n' + xml_str)


if __name__ == "__main__":
    main()
