package entity

import (
	"bytes"
	"github.com/mdfriday/hugoverse/internal/domain/markdown"
	"github.com/mdfriday/hugoverse/internal/domain/markdown/valueobject"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type Markdown struct {
	GoldMark goldmark.Markdown
	markdown.Highlighter
}

func (md *Markdown) Render(rctx markdown.RenderContext, dctx markdown.DocumentContext) (markdown.Result, error) {
	parseResult, err := md.parse(rctx)
	if err != nil {
		return nil, err
	}

	renderResult, err := md.render(rctx, dctx, parseResult.Doc())
	if err != nil {
		return nil, err
	}

	return struct {
		markdown.RenderingResult
		markdown.ParsingResult
	}{
		RenderingResult: renderResult,
		ParsingResult:   parseResult,
	}, nil
}

func (md *Markdown) parse(ctx markdown.RenderContext) (*valueobject.ParserResult, error) {
	pctx := valueobject.NewParserContext(ctx)
	reader := text.NewReader(ctx.Src)

	doc := md.GoldMark.Parser().Parse(
		reader,
		parser.WithContext(pctx),
	)

	return valueobject.NewParserResult(doc, pctx.TableOfContents(), ctx.Src), nil
}

func (md *Markdown) render(rctx markdown.RenderContext, dctx markdown.DocumentContext, doc ast.Node) (markdown.RenderingResult, error) {
	n := doc
	buf := &valueobject.BufWriter{Buffer: &bytes.Buffer{}}

	rcx := &valueobject.RenderContextDataHolder{
		Rctx: rctx,
		Dctx: dctx,
	}

	w := &valueobject.Context{
		BufWriter:   buf,
		ContextData: rcx,
	}

	if err := md.GoldMark.Renderer().Render(w, rctx.Src, n); err != nil {
		return nil, err
	}

	return buf, nil
}
