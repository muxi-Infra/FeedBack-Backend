FROM python:3.12-slim-bookworm AS builder

WORKDIR /app

ARG UV_INDEX_URL=https://pypi.org/simple

ENV PATH="/root/.local/bin:/app/.venv/bin:$PATH" \
    UV_INDEX_URL=${UV_INDEX_URL} \
    UV_LINK_MODE=copy

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        curl \
    && rm -rf /var/lib/apt/lists/*

RUN curl -Ls https://astral.sh/uv/install.sh | sh

COPY pyproject.toml uv.lock README.md ./
RUN uv sync --extra dev --no-install-project

COPY main.py ./
COPY models/ ./models/

RUN pyinstaller \
    --noconfirm \
    --clean \
    --onefile \
    --name app \
    main.py \
    && mkdir -p /runtime \
    && mv /app/dist/app /runtime/app \
    && mv /app/models /runtime/models

FROM debian:bookworm-slim

WORKDIR /runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libgomp1 \
        libstdc++6 \
        libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /runtime/app ./app
COPY --from=builder /runtime/models ./models

EXPOSE 8000

CMD ["./app"]
