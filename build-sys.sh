mkdir output
dhall-to-json --file build-sys.dhall | go run main.go -config - -number -parent-dir output hello
