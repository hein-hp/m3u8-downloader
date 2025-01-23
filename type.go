package main

type M3U8 struct {
	baseUrl       string
	baseUrlPrefix string
	encrypt       Encrypt
	frames        []Frame
}

type Encrypt struct {
	method string
	uri    string
	iv     []byte
	key    string
}

type Frame struct {
	Name string
	Url  string
}
