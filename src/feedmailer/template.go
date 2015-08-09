package main

import (
	"bytes"
	"github.com/aymerick/douceur/inliner"
	rss "github.com/jteeuwen/go-pkg-rss"
	//"github.com/mauidude/go-readability"
	html "html/template"
	"strings"
	"text/template"
)

const plainTextTemplate string = `{{ $items := .Items }}{{ range .Channels }}{{ $channel := . }}
### {{ .Title }}
{{ homePage . }}
{{ range $items }}{{ if eq .ChannelKey $channel.Key }}
{{ .Title }}
{{ firstLink .Links }}
{{ end }}{{ end }}{{ end }}
`

const htmlTextTemplate string = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<style type="text/css">
  p {
    font-family: 'Helvetica Neue', Verdana, sans-serif;
    color: #333;
  }

  div.toc {
  	margin-bottom: 20px;
  }

  div.channelHeader {
  	padding: 10px;
  	background-color: #ccc;
  	margin-top: 20px;
  }

  div.channelHeader p {
  	margin-top: 0;
  	padding-top: 0;
  }

  div.itemHeader {
  	background-color: #ddd;
  	border-top: 20px #333 solid;
  	padding: 10px;
  	margin-bottom: 10px;
  }

  div.itemHeader p {
  	margin-top: 0;
  	padding-top: 0;
  }

  div.itemContent {
  	margin-bottom: 20px;
  }
</style>
</head>
<body>
{{ $items := .Items }}

<div class="toc">
<h2>Contents</h2>
<ol>
	{{ range .Channels }}
	<li>{{ .Title }} - <a href="{{ homePage . }}">{{ homePage . }}</a></li>
	{{ end }}
</ol>
</div>

{{ range .Channels }}
{{ $channel := . }}
<div class="channelHeader">
	<h2>{{ .Title }}</h2>
	<p><a href="{{ homePage . }}">{{ homePage . }}</a></p>
</div>
	{{ range $items }}
	{{ if eq .ChannelKey $channel.Key }}
	<div class="itemHeader">
		<h3>{{ .Title }}</h3>
		<p><a href="{{ firstLink .Links }}">{{ firstLink .Links }}</a></p>
	</div>
	<div class="itemContent">
	{{ readability .FullContent }}
	</div>
	{{ end }}
	{{ end }}
{{ end }}
</body>
</html>
`

type EmailModel struct {
	Channels []*Chnl
	Items    []*Itm
}

var funcMap = map[string]interface{}{
	"title": func(a string) string { return strings.Title(a) },
	"firstLink": func(l []rss.Link) string {
		if len(l) == 0 {
			return ""
		} else {
			return l[0].Href
		}
	},
	"homePage": func(c Chnl) string { return c.HomePage() },
	"readability": func(s string) (html.HTML, error) {
		/*
			doc, err := readability.NewDocument(s)
			if err != nil {
				return html.HTML(""), err
			}
			return html.HTML(doc.Content()), nil
		*/
		return html.HTML(s), nil
	},
}

func renderPlainText(model *EmailModel) (string, error) {
	t, err := template.New("").Funcs(funcMap).Parse(plainTextTemplate)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = t.Execute(&b, model)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func renderHtmlText(model *EmailModel) (string, error) {
	t, err := html.New("").Funcs(funcMap).Parse(htmlTextTemplate)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = t.Execute(&b, model)
	if err != nil {
		return "", err
	}

	emailHtml, err := inliner.Inline(b.String())
	if err != nil {
		return "", err
	}

	return emailHtml, nil
}
