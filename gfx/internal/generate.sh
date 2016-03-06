# OpenGL ES 2 bindings:
glow generate -out=./gles2/2.0/gles2/ -api=gles2 -version=2.0 -restrict=./restrict.json

# OpenGL 2 bindings:
glow generate -out=./gl/2.0/gl/ -api=gl -version=2.0 -restrict=./restrict.json

F="./gles2/2.0/gles2/conversions.go" && echo -e "// +build arm gles2\n\n$(cat $F)" > $F
F="./gles2/2.0/gles2/debug.go" && echo -e "// +build arm gles2\n\n$(cat $F)" > $F
F="./gles2/2.0/gles2/package.go" && echo -e "// +build arm gles2\n\n$(cat $F)" > $F
F="./gles2/2.0/gles2/procaddr.go" && echo -e "// +build arm gles2\n\n$(cat $F)" > $F


F="./gl/2.0/gl/conversions.go" && echo -e "// +build 386,!gles2 amd64,!gles2\n\n$(cat $F)" > $F
F="./gl/2.0/gl/debug.go" && echo -e "// +build 386,!gles2 amd64,!gles2\n\n$(cat $F)" > $F
F="./gl/2.0/gl/package.go" && echo -e "// +build 386,!gles2 amd64,!gles2\n\n$(cat $F)" > $F
F="./gl/2.0/gl/procaddr.go" && echo -e "// +build 386,!gles2 amd64,!gles2\n\n$(cat $F)" > $F
