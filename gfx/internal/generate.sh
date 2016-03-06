# OpenGL ES 2 bindings:
rm -rf ./gles2
glow generate -out=./gles2/2.0/gles2/ -api=gles2 -version=2.0 -restrict=./restrict.json

# OpenGL 2 bindings:
rm -rf ./gl
glow generate -out=./gl/2.0/gl/ -api=gl -version=2.0 -restrict=./restrict.json
