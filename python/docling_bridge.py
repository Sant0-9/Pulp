#!/usr/bin/env python3
"""Bridge between Go and Docling for document conversion."""

import sys
import json
from pathlib import Path
from datetime import datetime

try:
    from docling.document_converter import DocumentConverter
except ImportError:
    print(json.dumps({
        "success": False,
        "error": "Docling not installed. Run: pip install docling"
    }))
    sys.exit(1)


def convert(path: str) -> dict:
    """Convert document and return structured data."""
    p = Path(path)

    if not p.exists():
        return {"success": False, "error": f"File not found: {path}"}

    try:
        converter = DocumentConverter()
        result = converter.convert(str(p))
        doc = result.document

        # Export to markdown
        markdown = doc.export_to_markdown()

        # Get metadata
        metadata = {
            "title": getattr(doc, 'title', None) or p.stem,
            "source_path": str(p.absolute()),
            "source_format": p.suffix.lower().lstrip('.'),
            "file_size_bytes": p.stat().st_size,
            "converted_at": datetime.now().isoformat(),
        }

        # Try to get page count
        if hasattr(doc, 'pages'):
            metadata["page_count"] = len(doc.pages)

        # Word count estimate
        metadata["word_count"] = len(markdown.split())

        # Get preview (first 500 chars)
        preview = markdown[:500].strip()
        if len(markdown) > 500:
            preview += "..."

        return {
            "success": True,
            "markdown": markdown,
            "metadata": metadata,
            "preview": preview,
        }

    except Exception as e:
        return {"success": False, "error": str(e)}


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"success": False, "error": "Usage: docling_bridge.py <file_path>"}))
        sys.exit(1)

    result = convert(sys.argv[1])
    print(json.dumps(result))

    if not result["success"]:
        sys.exit(1)


if __name__ == "__main__":
    main()
