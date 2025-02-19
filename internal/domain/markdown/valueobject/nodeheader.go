package valueobject

import (
	"github.com/mdfriday/hugoverse/internal/domain/markdown"
	"github.com/yuin/goldmark/ast"
)

type HeaderNode struct {
	text  string // 标题文本内容
	level int    // 标题层级 (1 = H1, 2 = H2, ...)

	node ast.Node
	src  []byte
}

func (h *HeaderNode) Name() string {
	return h.text
}

func (h *HeaderNode) Level() int {
	return h.level
}

func (h *HeaderNode) Links() []markdown.Link {
	var links []markdown.Link

	for sibling := h.node.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
		// 遇到下一个 Header，结束收集
		if heading, ok := sibling.(*ast.Heading); ok {
			if heading.Level <= h.Level() {
				break
			}
		}

		// 解析链接节点
		if list, ok := sibling.(*ast.List); ok {
			for listItem := list.FirstChild(); listItem != nil; listItem = listItem.NextSibling() {
				if textBlock, ok := listItem.FirstChild().(*ast.TextBlock); ok {
					for linkNode := textBlock.FirstChild(); linkNode != nil; linkNode = linkNode.NextSibling() {
						if link, ok := linkNode.(*ast.Link); ok {
							text := extractTextFromNode(link, h.src)
							url := string(link.Destination)
							links = append(links, &LinkNode{
								text: text,
								url:  url,
							})
						}
					}
				}
			}
		}
	}

	return links
}

func (h *HeaderNode) ListParagraphs() []markdown.Paragraph {
	var paragraphs []markdown.Paragraph

	for sibling := h.node.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
		// 遇到下一个 Header，结束收集
		if heading, ok := sibling.(*ast.Heading); ok {
			if heading.Level <= h.Level() {
				break
			}
		}

		// 解析链接节点
		if list, ok := sibling.(*ast.List); ok {
			for listItem := list.FirstChild(); listItem != nil; listItem = listItem.NextSibling() {
				if textBlock, ok := listItem.FirstChild().(*ast.TextBlock); ok {
					text := extractTextFromNode(textBlock, h.src)
					paragraphs = append(paragraphs, &ParagraphNode{
						text: text,
					})
				}
			}
		}
	}

	return paragraphs
}

func (h *HeaderNode) Paragraphs() []markdown.Paragraph {
	var paragraphs []markdown.Paragraph

	for sibling := h.node.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
		// 如果遇到下一个 Header，停止收集
		if heading, ok := sibling.(*ast.Heading); ok {
			if heading.Level <= h.Level() {
				break
			}
		}

		// 检查段落节点
		if paragraph, ok := sibling.(*ast.Paragraph); ok {
			text := extractAllTextFromNode(paragraph, h.src)
			paragraphs = append(paragraphs, &ParagraphNode{text: text})
		}
	}

	return paragraphs
}
