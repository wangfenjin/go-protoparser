package parser

import (
	"github.com/yoheimuta/go-protoparser/internal/lexer/scanner"
	"github.com/yoheimuta/go-protoparser/parser/meta"
)

// Option can be used in proto files, messages, enums and services.
type Option struct {
	OptionName string
	Constant   string

	// Comments are the optional ones placed at the beginning.
	Comments []*Comment
	// InlineComment is the optional one placed at the ending.
	InlineComment *Comment
	// Meta is the meta information.
	Meta meta.Meta
}

// SetInlineComment implements the HasInlineCommentSetter interface.
func (o *Option) SetInlineComment(comment *Comment) {
	o.InlineComment = comment
}

// Accept dispatches the call to the visitor.
func (o *Option) Accept(v Visitor) {
	if !v.VisitOption(o) {
		return
	}

	for _, comment := range o.Comments {
		comment.Accept(v)
	}
	if o.InlineComment != nil {
		o.InlineComment.Accept(v)
	}
}

// ParseOption parses the option.
//  option = "option" optionName  "=" constant ";"
//
// See https://developers.google.com/protocol-buffers/docs/reference/proto3-spec#option
func (p *Parser) ParseOption() (*Option, error) {
	p.lex.NextKeyword()
	if p.lex.Token != scanner.TOPTION {
		return nil, p.unexpected("option")
	}
	startPos := p.lex.Pos

	optionName, err := p.parseOptionName()
	if err != nil {
		return nil, err
	}

	p.lex.Next()
	if p.lex.Token != scanner.TEQUALS {
		return nil, p.unexpected("=")
	}

	var constant string
	switch p.lex.Peek() {
	// Cloud Endpoints requires this exception.
	case scanner.TLEFTCURLY:
		if !p.permissive {
			return nil, p.unexpected("constant or permissive mode")
		}

		constant, err = p.parseCloudEndpointsOptionConstant()
		if err != nil {
			return nil, err
		}
	default:
		constant, _, err = p.lex.ReadConstant()
		if err != nil {
			return nil, err
		}
	}

	p.lex.Next()
	if p.lex.Token != scanner.TSEMICOLON {
		return nil, p.unexpected(";")
	}

	return &Option{
		OptionName: optionName,
		Constant:   constant,
		Meta:       meta.NewMeta(startPos),
	}, nil
}

// cloudEndpointsOptionConstant = "{" ident ":" constant { [","] ident ":" constant } "}"
//
// See https://cloud.google.com/endpoints/docs/grpc-service-config/reference/rpc/google.api
func (p *Parser) parseCloudEndpointsOptionConstant() (string, error) {
	var ret string

	p.lex.Next()
	if p.lex.Token != scanner.TLEFTCURLY {
		return "", p.unexpected("{")
	}
	ret += p.lex.Text

	for {
		p.lex.Next()
		if p.lex.Token != scanner.TIDENT {
			return "", p.unexpected("ident")
		}
		ret += p.lex.Text

		p.lex.Next()
		if p.lex.Token != scanner.TCOLON {
			return "", p.unexpected(":")
		}
		ret += p.lex.Text

		constant, _, err := p.lex.ReadConstant()
		if err != nil {
			return "", err
		}
		ret += constant

		p.lex.Next()
		switch {
		case p.lex.Token == scanner.TCOMMA:
			ret += p.lex.Text
		case p.lex.Token == scanner.TRIGHTCURLY:
			ret += p.lex.Text
			return ret, nil
		default:
			ret += "\n"
			p.lex.UnNext()
		}
	}
}

// optionName = ( ident | "(" fullIdent ")" ) { "." ident }
func (p *Parser) parseOptionName() (string, error) {
	var optionName string

	p.lex.Next()
	switch p.lex.Token {
	case scanner.TIDENT:
		optionName = p.lex.Text
	case scanner.TLEFTPAREN:
		optionName = p.lex.Text
		fullIdent, _, err := p.lex.ReadFullIdent()
		if err != nil {
			return "", err
		}
		optionName += fullIdent

		p.lex.Next()
		if p.lex.Token != scanner.TRIGHTPAREN {
			return "", p.unexpected(")")
		}
		optionName += p.lex.Text
	default:
		return "", p.unexpected("ident or left paren")
	}

	for {
		p.lex.Next()
		if p.lex.Token != scanner.TDOT {
			p.lex.UnNext()
			break
		}
		optionName += p.lex.Text

		p.lex.Next()
		if p.lex.Token != scanner.TIDENT {
			return "", p.unexpected("ident")
		}
		optionName += p.lex.Text
	}
	return optionName, nil
}
