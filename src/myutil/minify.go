package myutil

import (
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
)

func MinifyHtml(htmlStr string, devMode bool) string {
	if devMode {
		return htmlStr
	}

	mini := minify.New()
	mini.AddFunc("text/html", html.Minify)
	minified, _ := mini.String("text/html", htmlStr)
	return minified
}

func MinifyCssJs(mergedCss, mergedJs string, devMode bool) (string, string) {
	if devMode {
		return mergedCss, mergedJs
	}

	mini := minify.New()
	mini.AddFunc("text/css", css.Minify)
	mini.AddFunc("text/javascript", js.Minify)

	minifiedCss, _ := mini.String("text/css", mergedCss)
	minifiedJs, _ := mini.String("text/javascript", mergedJs)

	return minifiedCss, minifiedJs
}
