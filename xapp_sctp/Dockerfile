FROM golang:1.22 AS builder

# 1. Instala ferramentas de build
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    build-essential \
    cmake \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 2. Baixa e instala os pacotes .deb do RMR (Release E)
RUN wget -q https://packagecloud.io/o-ran-sc/release/packages/debian/stretch/rmr_4.8.0_amd64.deb/download.deb -O rmr.deb && \
    wget -q https://packagecloud.io/o-ran-sc/release/packages/debian/stretch/rmr-dev_4.8.0_amd64.deb/download.deb -O rmr-dev.deb && \
    dpkg -i rmr.deb rmr-dev.deb && \
    rm -f rmr.deb rmr-dev.deb

# 3. CORREÇÃO DO CGO: Injeta a constante que falta no arquivo de cabeçalho C do RMR
RUN find /usr -name "RIC_message_types.h" -exec sh -c 'echo "\n#ifndef RIC_SUB_DEL_REQUIRED\n#define RIC_SUB_DEL_REQUIRED 12023\n#endif" >> {}' \;

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 4. COMPILAÇÃO DO C: Compila os arquivos C do E2AP e gera a biblioteca estática libe2ap.a
RUN gcc -c -I./e2ap/headers -I./e2ap -I./e2ap/lib ./e2ap/wrapper.c ./e2ap/lib/*.c -fPIC && \
    ar rcs /app/control/libe2ap.a *.o && \
    rm -f *.o

# 5. COMPILAÇÃO DO C: Compila os arquivos C do E2SM e gera a biblioteca estática libe2sm.a
RUN gcc -c -I./e2sm/headers -I./e2sm -I./e2sm/lib ./e2sm/wrapper.c ./e2sm/lib/*.c -fPIC && \
    ar rcs /app/control/libe2sm.a *.o && \
    rm -f *.o

# 6. Compila o binário em Go linkando a biblioteca estática recém-criada
RUN CGO_ENABLED=1 GOOS=linux go build -o testar-e2sim .

# Estágio final (runtime enxuto)
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /usr/local/lib/librmr_si.so* /usr/local/lib/
COPY --from=builder /app/testar-e2sim .
COPY config/ ./config/

ENV LD_LIBRARY_PATH=/usr/local/lib
ENV CFG_FILE=/root/config/config.json

CMD ["./testar-e2sim"]
