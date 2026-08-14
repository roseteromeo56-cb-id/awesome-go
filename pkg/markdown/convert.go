package markdown

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/avelino/awesome-go/pkg/slug"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	xhtml "golang.org/x/net/html"
)

// ToHTML converts markdown byte slice to a HTML byte slice
func ToHTML(markdown []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // generate heading IDs for content navigation
		),
		goldmark.WithRendererOptions(
			html.WithXHTML(),
			html.WithUnsafe(), // allow README inline HTML before sanitizing below
		),
	)

	ctx := parser.NewContext(
		parser.WithIDs(&IDGenerator{}), // register custom ID generator
	)

	var buf bytes.Buffer
	if err := md.Convert(markdown, &buf, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	return sanitizeHTML(buf.Bytes())
}

var allowedElements = map[string]bool{
	"a": true, "b": true, "blockquote": true, "br": true, "code": true,
	"del": true, "div": true, "em": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "hr": true, "i": true, "img": true,
	"li": true, "ol": true, "p": true, "pre": true, "strong": true, "sub": true,
	"sup": true, "table": true, "tbody": true, "td": true, "th": true,
	"thead": true, "tr": true, "ul": true,
}

var removedElements = map[string]bool{
	"applet": true, "base": true, "embed": true, "form": true, "frame": true,
	"frameset": true, "iframe": true, "link": true, "math": true, "meta": true,
	"object": true, "script": true, "style": true, "svg": true,
}

var allowedAttributes = map[string]map[string]bool{
	"*": {
		"align": true,
		"id":    true,
		"title": true,
	},
	"a": {
		"href":   true,
		"rel":    true,
		"target": true,
		"title":  true,
	},
	"img": {
		"align":  true,
		"alt":    true,
		"height": true,
		"src":    true,
		"title":  true,
		"width":  true,
	},
	"table": {
		"cellpadding": true,
		"cellspacing": true,
	},
	"td": {
		"colspan": true,
		"rowspan": true,
	},
	"th": {
		"colspan": true,
		"rowspan": true,
	},
}

func sanitizeHTML(input []byte) ([]byte, error) {
	context := &xhtml.Node{Type: xhtml.ElementNode, Data: "body"}
	nodes, err := xhtml.ParseFragment(bytes.NewReader(input), context)
	if err != nil {
		return nil, fmt.Errorf("parse html fragment: %w", err)
	}

	root := &xhtml.Node{Type: xhtml.DocumentNode}
	for _, node := range nodes {
		root.AppendChild(node)
	}
	sanitizeChildren(root)

	var buf bytes.Buffer
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		if err := xhtml.Render(&buf, node); err != nil {
			return nil, fmt.Errorf("render sanitized html: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func sanitizeChildren(parent *xhtml.Node) {
	for node := parent.FirstChild; node != nil; {
		next := node.NextSibling
		sanitizeNode(parent, node)
		node = next
	}
}

func sanitizeNode(parent, node *xhtml.Node) {
	if node.Type != xhtml.ElementNode {
		return
	}

	node.Data = strings.ToLower(node.Data)
	if removedElements[node.Data] {
		parent.RemoveChild(node)
		return
	}

	sanitizeChildren(node)

	if !allowedElements[node.Data] {
		unwrapNode(parent, node)
		return
	}

	node.Attr = sanitizeAttributes(node.Data, node.Attr)
}

func unwrapNode(parent, node *xhtml.Node) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		node.RemoveChild(child)
		parent.InsertBefore(child, node)
		child = next
	}
	parent.RemoveChild(node)
}

func sanitizeAttributes(tag string, attrs []xhtml.Attribute) []xhtml.Attribute {
	clean := attrs[:0]
	for _, attr := range attrs {
		attr.Namespace = ""
		attr.Key = strings.ToLower(attr.Key)

		if strings.HasPrefix(attr.Key, "on") || !isAllowedAttribute(tag, attr.Key) {
			continue
		}
		if isURLAttribute(attr.Key) && !isSafeURL(attr.Val) {
			continue
		}

		clean = append(clean, attr)
	}

	if tag == "a" && hasAttribute(clean, "target", "_blank") && !hasAttributeKey(clean, "rel") {
		clean = append(clean, xhtml.Attribute{Key: "rel", Val: "noopener noreferrer"})
	}

	return clean
}

func isAllowedAttribute(tag, attr string) bool {
	return allowedAttributes["*"][attr] || allowedAttributes[tag][attr]
}

func isURLAttribute(attr string) bool {
	return attr == "href" || attr == "src"
}

func isSafeURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}

	switch strings.ToLower(u.Scheme) {
	case "", "http", "https", "mailto":
		return true
	default:
		return false
	}
}

func hasAttribute(attrs []xhtml.Attribute, key, val string) bool {
	for _, attr := range attrs {
		if attr.Key == key && attr.Val == val {
			return true
		}
	}
	return false
}

func hasAttributeKey(attrs []xhtml.Attribute, key string) bool {
	for _, attr := range attrs {
		if attr.Key == key {
			return true
		}
	}
	return false
}

// IDGenerator for goldmark to provide IDs more similar to GitHub's IDs on markdown parsing
type IDGenerator struct {
	used map[string]bool
}

// Generate an ID
func (g *IDGenerator) Generate(value []byte, _ ast.NodeKind) []byte {
	return []byte(slug.Generate(string(value)))
}

// Put an ID to the list of already used IDs
func (g *IDGenerator) Put(value []byte) {
	g.used[util.BytesToReadOnlyString(value)] = true
}
