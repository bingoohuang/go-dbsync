package main

import "bytes"

func mergeCss() string {
	var scripts bytes.Buffer
	scripts.Write(MustAsset("res/stylesheet.css"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/codemirror-5.29.0.min.css"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/index.css"))
	return scripts.String()
}

func mergeScripts() string {
	var scripts bytes.Buffer
	scripts.Write(MustAsset("res/jquery-3.2.1.min.js"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/codemirror-5.29.0.min.js"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/matchbrackets-5.29.0.min.js"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/javascript-5.29.0.min.js"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/autosize-4.0.0.min.js"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/index.js"))
	return scripts.String()
}
