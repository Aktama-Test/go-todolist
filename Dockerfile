# syntax=docker/dockerfile:1
FROM golang:1.26 AS build


WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download && go mod verify

RUN curl -L https://depot.dev/install-cli.sh | sh

COPY . .

# RUN --mount=type=cache,target=/go/pkg/mod \
#   --mount=type=cache,target=/root/.cache/go-build \
#   go build -o /bin/app .
RUN --mount=type=secret,id=DEPOT_TOKEN,env=DEPOT_TOKEN \
  PATH="/root/.depot/bin:$PATH" \
  GOCACHEPROG="depot gocache" \
  go build -v -o /bin/app .

FROM ubuntu:24.04 AS runtime

# RUN groupadd -g 1001 nonroot && \
#   useradd -u 1001 -g nonroot -m -d /app -s /bin/false nonroot

COPY --from=build /bin/app /usr/local/bin/app/todolist

# RUN chown nonroot:nonroot /usr/local/bin/app

# USER nonroot

ENV TZ=UTC \
  GOMAXPROCS=0

ENTRYPOINT ["/usr/local/bin/app/todolist"]
