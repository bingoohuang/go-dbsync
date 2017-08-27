package main

import "bytes"

func mergeCss() string {
	var scripts bytes.Buffer
	scripts.Write(MustAsset("res/stylesheet.css"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/index.css"))
	return scripts.String()
}

func mergeScripts() string {
	var scripts bytes.Buffer
	scripts.Write(MustAsset("res/jquery-3.2.1.min.js"))
	scripts.Write([]byte("\n"))
	scripts.Write(MustAsset("res/index.js"))
	return scripts.String()
}
