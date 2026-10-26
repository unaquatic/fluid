package syntax

import "testing"

func TestDockerfile(t *testing.T) {
	testLanguage(t, "dockerfile", []string{"docker", "containerfile", "Dockerfile", "Containerfile", "build/Dockerfile"}, dockerfileTests)
}

var dockerfileTests = []kindCase{
	{name: "instructions", src: "# syntax=docker/dockerfile:1\nFROM golang:1.22 AS build\nWORKDIR /src\nCOPY --from=build /app /app\nEXPOSE 8080\nUSER nobody\nVOLUME /data\nSTOPSIGNAL SIGTERM", want: []Token{
		tok(Comment, "# syntax=docker/dockerfile:1"), tok(Keyword, "FROM"), tok(Text, " golang:1.22 "), tok(Keyword, "AS"),
		tok(Text, " build"), tok(Keyword, "WORKDIR"), tok(Text, "/src"), tok(Keyword, "COPY"), tok(Text, "--from=build"),
		tok(Keyword, "EXPOSE"), tok(Number, "8080"), tok(Keyword, "USER"), tok(Text, "nobody"), tok(Keyword, "VOLUME"),
		tok(Keyword, "STOPSIGNAL"), tok(Text, "SIGTERM"),
	}},
	{name: "assignments", src: "ENV CGO_ENABLED=0 GOOS=linux\nARG VERSION=dev\nLABEL maintainer=\"me\"", want: []Token{
		tok(Keyword, "ENV"), tok(Variable, "CGO_ENABLED"), tok(Operator, "="), tok(Text, "0"), tok(Variable, "GOOS"),
		tok(Operator, "="), tok(Text, "linux"), tok(Keyword, "ARG"), tok(Variable, "VERSION"), tok(Keyword, "LABEL"),
		tok(Variable, "maintainer"), tok(Operator, "="), tok(String, `"me"`),
	}},
	{name: "exec form", src: "ENTRYPOINT [\"/app\"]\nCMD [\"--port\", \"80\"]\nSHELL [\"/bin/bash\", \"-c\"]", want: []Token{
		tok(Keyword, "ENTRYPOINT"), tok(Punctuation, "["), tok(String, `"/app"`), tok(Punctuation, "]"), tok(Keyword, "CMD"),
		tok(String, `"--port"`), tok(Punctuation, ","), tok(String, `"80"`), tok(Keyword, "SHELL"),
	}},
	{name: "shell commands across lines", src: "RUN apt-get update && \\\n    apt-get install -y git curl && \\\n    rm -rf /var/lib/apt/lists/*\nRUN echo \"$HOME\" > /tmp/x", want: []Token{
		tok(Keyword, "RUN"), tok(Text, "apt-get update"), tok(Operator, "&&"), tok(StringEscape, `\`),
		tok(Text, "apt-get install -y git curl"), tok(Operator, "&&"), tok(StringEscape, `\`),
		tok(Text, "rm -rf /var/lib/apt/lists/*"), tok(Keyword, "RUN"), tok(BuiltinFunction, "echo"), tok(Variable, "$HOME"),
		tok(Operator, ">"),
	}},
	{name: "comments inside a continued command", src: "RUN make \\\n    # build it\n    all", want: []Token{
		tok(Keyword, "RUN"), tok(StringEscape, `\`), tok(Comment, "# build it"),
	}},
	{name: "instructions ignore case", src: "from alpine\nrun ls\nONBUILD RUN make", want: []Token{
		tok(Keyword, "from"), tok(Keyword, "run"), tok(Keyword, "ONBUILD"), tok(Keyword, "RUN"), tok(Text, "make"),
	}},
	{name: "heredocs", src: "RUN <<EOF\napt-get update\nEOF\nCOPY . .", want: []Token{
		tok(Keyword, "RUN"), tok(Operator, "<<"), tok(String, "EOF"), tok(String, "apt-get update"), tok(String, "EOF"),
		tok(Keyword, "COPY"), tok(Text, ". ."),
	}},
	{name: "healthchecks", src: "HEALTHCHECK --interval=30s CMD curl -f http://x/ || exit 1", want: []Token{
		tok(Keyword, "HEALTHCHECK"), tok(Operator, "||"), tok(Keyword, "exit"), tok(Number, "1"),
	}},
	{name: "unclosed continuation", src: "RUN make \\", open: true, want: []Token{
		tok(Keyword, "RUN"), tok(StringEscape, `\`),
	}},
}
