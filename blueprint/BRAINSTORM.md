# Pulp - Document Summarization Engine

> Ingest documents. Extract the essence. Output human-readable summaries.

## Overview

Pulp is a CLI tool that processes long-form documents (PDF, DOCX, MD, TXT, etc.) and produces natural-sounding summaries using a hybrid local + API LLM approach.

## Goals

- Ingest any document format (PDF, DOCX, PPTX, XLSX, MD, TXT, HTML, images)
- Summarize content that sounds human, not robotic
- Customizable report formats (MD, JSON)
- Minimal API costs through local model preprocessing
- Fast and reliable

---

## Research Findings

### Document Parsing: Docling

After evaluating options, [Docling by IBM](https://github.com/docling-project/docling) is the clear winner:

| Feature | Docling | PyMuPDF + python-docx |
|---------|---------|----------------------|
| Formats | PDF, DOCX, PPTX, XLSX, HTML, images, audio | Need multiple libs |
| Setup | `pip install docling`, 5 lines | Multiple installs |
| Output | Native Markdown, JSON, HTML | Manual conversion |
| Tables | AI-powered extraction | Basic/manual |
| OCR | Built-in (scanned PDFs) | Separate setup |
| RAG-ready | LangChain/LlamaIndex integration | DIY |

Basic usage:

```python
from docling.document_converter import DocumentConverter
result = DocumentConverter().convert("doc.pdf")
markdown = result.document.export_to_markdown()
```

### Local Model Selection

Based on [DataCamp's 2026 SLM analysis](https://www.datacamp.com/blog/top-small-language-models) and [Ollama benchmarks](https://ollama.com/library):

| Model | Params | RAM | Strength |
|-------|--------|-----|----------|
| **Qwen3 4B** | 4B | ~3GB | Best balance of speed/quality |
| Phi-4-mini | 3.8B | ~3GB | Strong reasoning, 128K context |
| Llama 3.2 3B | 3B | ~2GB | Fastest, slightly less capable |

**Recommendation**: Qwen3 4B - trained on 18T tokens, multilingual, consistently tops extraction benchmarks.

### API Model: Groq

From [Groq's pricing page](https://groq.com/pricing):

| Model | Input/M | Output/M | Speed | Use Case |
|-------|---------|----------|-------|----------|
| **Llama 3.1 8B Instant** | $0.05 | $0.08 | 500+ tok/s | MVP - very cheap |
| Llama 3.3 70B | $0.59 | $0.79 | 300+ tok/s | Production quality |
| Qwen3 32B | $0.29 | $0.59 | Fast | Middle ground |

**Cost estimate**: 50-page document = ~25K tokens input, ~2K output = **$0.0015** with 8B Instant.

**Discounts available**:
- 50% off with Batch API
- 50% off cached prompts

### Chunking Strategy

[Clinical research on RAG chunking](https://pmc.ncbi.nlm.nih.gov/articles/PMC12649634/) shows:

| Strategy | Accuracy | Relevance |
|----------|----------|-----------|
| **Adaptive/Semantic** | 87% | 93% |
| Fixed-size | 50% | baseline |

Adaptive = split on semantic boundaries (paragraphs, sections, headers) not arbitrary token counts.

Docling preserves document structure, enabling semantic chunking out of the box.

---

## Architecture

```
[Files: PDF, DOCX, MD, TXT, etc.]
           |
           v
    [Docling Parser]
           |
           v
    [Structured Markdown with sections]
           |
           v
    [Semantic Chunker] -- split on headers/paragraphs
           |
           v
    [Qwen3 4B local] -- extract 3-5 key points per chunk
           |                "Be specific. Names, numbers, dates."
           v
    [Aggregator] -- dedupe, group by theme
           |
           v
    [Groq Llama 3.1 8B] -- synthesize into human prose
           |                 "Write like explaining to a colleague"
           v
    [Report: MD + JSON]
```

### Why This Architecture

1. **Docling** handles all parsing complexity - no format-specific code
2. **Qwen3 4B local** is free, fast (~50 tok/s on CPU), handles bulk extraction
3. **Groq 8B** costs ~$0.001/doc, runs at 500 tok/s, good enough quality
4. **Adaptive chunking** from Docling's structure = 87% accuracy vs 50% baseline
5. **Upgrade path**: Swap Groq 8B for 70B ($0.01/doc) when quality matters more

---

## Cost Projections

| Docs/month | Local cost | API cost | Total |
|------------|------------|----------|-------|
| 100 | $0 | $0.15 | $0.15 |
| 1,000 | $0 | $1.50 | $1.50 |
| 10,000 | $0 | $15.00 | $15.00 |

---

## Project Structure (MVP)

```
pulp/
  src/
    pulp/
      __init__.py
      converter.py    # Docling wrapper
      chunker.py      # Semantic splitting
      extractor.py    # Ollama/Qwen3 local
      synthesizer.py  # Groq API
      reporter.py     # MD + JSON output
      cli.py          # typer CLI
  tests/
  pyproject.toml
```

## CLI Design

```bash
# Summarize a single file
pulp summarize paper.pdf -o summary.md

# Summarize a folder
pulp summarize ./documents/ -o report.md

# JSON output
pulp summarize paper.pdf -o summary.json --format json

# Custom style
pulp summarize paper.pdf --style "executive briefing"
pulp summarize paper.pdf --style "casual explanation"

# Verbose mode (show extraction progress)
pulp summarize paper.pdf -v
```

## Configuration

```yaml
# ~/.config/pulp/config.yaml
local_model: qwen3:4b
api_provider: groq
api_model: llama-3.1-8b-instant
default_format: markdown
default_style: conversational
```

---

## Tech Stack

- **Python 3.10+** - best ecosystem for document parsing
- **Docling** - document ingestion
- **Ollama** - local LLM runtime
- **Groq SDK** - API calls
- **Typer** - CLI framework
- **Pydantic** - config/validation

---

## Future Enhancements (Post-MVP)

- [ ] PDF report generation
- [ ] HTML output with styling
- [ ] Web interface (FastAPI + HTMX)
- [ ] Batch processing with progress bar
- [ ] Custom prompt templates
- [ ] Multiple summary lengths (brief, standard, detailed)
- [ ] Source citations in output
- [ ] Comparison mode (summarize multiple docs, highlight differences)

---

## References

- [Docling GitHub](https://github.com/docling-project/docling)
- [Docling Documentation](https://docling-project.github.io/docling/)
- [Groq Pricing](https://groq.com/pricing)
- [DataCamp SLM Guide 2026](https://www.datacamp.com/blog/top-small-language-models)
- [RAG Chunking Clinical Study](https://pmc.ncbi.nlm.nih.gov/articles/PMC12649634/)
- [Ollama Model Library](https://ollama.com/library)
- [Weaviate Chunking Strategies](https://weaviate.io/blog/chunking-strategies-for-rag)
- [IBM Docling Announcement](https://research.ibm.com/blog/docling-generative-AI)
