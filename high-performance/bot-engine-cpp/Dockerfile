FROM ubuntu:22.04 AS builder

RUN apt-get update && apt-get install -y \
    build-essential cmake g++ \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

RUN mkdir build && cd build && cmake .. -DCMAKE_BUILD_TYPE=Release && make -j$(nproc)

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/build/bot_engine /usr/local/bin/bot_engine

ENTRYPOINT ["bot_engine"]
